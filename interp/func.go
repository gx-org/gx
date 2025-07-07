// Copyright 2024 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package interp

import (
	"fmt"
	"go/ast"

	"github.com/pkg/errors"
	"github.com/gx-org/gx/api/values"
	"github.com/gx-org/gx/build/fmterr"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/interp/elements"
)

type funcBase struct {
	fn   ir.Func
	recv *elements.Receiver
}

var _ elements.Func = (*funcBase)(nil)

// NewRunFunc creates a function given an IR and a receiver.
// The function is run when being called.
func NewRunFunc(fn ir.Func, recv *elements.Receiver) elements.Func {
	switch fnT := fn.(type) {
	case *ir.FuncDecl:
		return &funcDecl{
			funcBase: funcBase{fn: fnT, recv: recv},
			fnT:      fnT,
		}
	case *ir.FuncBuiltin:
		return &funcBuiltin{
			funcBase: funcBase{fn: fnT, recv: recv},
			fnT:      fnT,
		}
	case *ir.FuncLit:
		return &funcLit{
			funcBase: funcBase{fn: fnT, recv: recv},
			fnT:      fnT,
		}
	}
	return &funcBase{fn: fn, recv: recv}
}

// Type of the function.
func (st *funcBase) Type() ir.Type {
	return st.fn.FuncType()
}

// Flatten returns the element in a slice.
func (st *funcBase) Flatten() ([]elements.Element, error) {
	return []elements.Element{st}, nil
}

// Func returns the function represented by the node.
func (st *funcBase) Func() ir.Func {
	return st.fn
}

// Recv returns the receiver of the function or nil if the function has no receiver.
func (st *funcBase) Recv() *elements.Receiver {
	return st.recv
}

// Unflatten creates a GX value from the next handles available in the Unflattener.
func (st *funcBase) Unflatten(handles *elements.Unflattener) (values.Value, error) {
	return values.NewIRNode(st.fn)
}

// Kind of the element.
func (*funcBase) Kind() ir.Kind {
	return ir.FuncKind
}

// Call the function.
func (st *funcBase) Call(ctx ir.Evaluator, call *ir.CallExpr, args []elements.Element) ([]elements.Element, error) {
	return nil, fmterr.Internalf(ctx.File().FileSet(), st.fn.Source(), "function type %T not supported", st.fn)
}

// String representation of the node.
func (st *funcBase) String() string {
	return fmt.Sprintf("func(%s)", st.Func().Name())
}

type funcDecl struct {
	funcBase
	fnT *ir.FuncDecl
}

func (f *funcDecl) Call(fctx ir.Evaluator, call *ir.CallExpr, args []elements.Element) (outs []elements.Element, err error) {
	ctx := fctx.(*Context)
	if f.fnT.Body == nil {
		return nil, fmterr.Errorf(ctx.File().FileSet(), f.fnT.Source(), "missing function body")
	}
	// Create a new function frame.
	funcFrame, err := ctx.pushFuncFrame(f.fnT)
	if err != nil {
		return nil, err
	}
	defer ctx.popFrame()
	for _, resultName := range fieldNames(f.fnT.FType.Results.List) {
		funcFrame.Define(resultName.Name, nil)
	}
	// Add the receiver name to the function frame if present.
	if recv := f.Recv(); recv != nil {
		recvNode := recv.Element.RecvCopy()
		if recv.Ident != nil {
			funcFrame.Define(recv.Ident.Name, recvNode)
		}
	}
	assignArgumentValues(f.fnT.FType, funcFrame, args)
	// Evaluate the function within the frame.
	return evalFuncBody(ctx, f.fnT.Body)
}

type funcBuiltin struct {
	funcBase
	fnT *ir.FuncBuiltin
}

func (f *funcBuiltin) Call(fctx ir.Evaluator, call *ir.CallExpr, args []elements.Element) (outs []elements.Element, err error) {
	defer func() {
		if err != nil {
			err = fmterr.Position(fctx.File().FileSet(), call.Expr(), err)
		}
	}()
	ctx := fctx.(*Context)
	var impl FuncBuiltin
	if f.fnT.Impl != nil {
		impl = f.fnT.Impl.Implementation().(FuncBuiltin)
	}
	if impl == nil {
		err = errors.Errorf("function %s has no implementation", f.fn.Name())
		return
	}
	return impl(ctx, elements.NewNodeAt[*ir.CallExpr](ctx.File(), call), f, f.fnT, args)
}

type funcLit struct {
	funcBase
	fnT *ir.FuncLit
}

func (f *funcLit) Call(fctx ir.Evaluator, call *ir.CallExpr, args []elements.Element) (outs []elements.Element, err error) {
	ctx := fctx.(*Context)
	// TODO(degris): remove this hack.
	f.fnT.FFile = fctx.File()
	return ctx.evaluator.CallFuncLit(ctx, f.fnT, args)
}

func evalFuncBody(ctx *Context, body *ir.BlockStmt) ([]elements.Element, error) {
	outs, stop, err := evalBlockStmt(ctx, body)
	if !stop {
		// No return statement was processed during the eval of the function.
		return nil, fmterr.Errorf(ctx.File().FileSet(), body.Src, "missing return")
	}
	return outs, err
}

func fieldNames(fields []*ir.FieldGroup) (r []*ast.Ident) {
	for _, arg := range fields {
		for _, name := range arg.Src.Names {
			r = append(r, name)
		}
	}
	return
}

func assignArgumentValues(funcType *ir.FuncType, funcFrame *blockFrame, args []elements.Element) {
	// For each parameter of the function, assign its argument value to the frame.
	names := fieldNames(funcType.Params.List)
	for i, arg := range args {
		copyable, ok := arg.(elements.Copier)
		if ok {
			arg = copyable.Copy()
		}
		funcFrame.Define(names[i].Name, arg)
	}
}

func evalCallExpr(ctx *Context, expr *ir.CallExpr) (elements.Element, error) {
	outs, err := evalCall(ctx, expr)
	if err != nil {
		return nil, err
	}
	return ToSingleElement(ctx, expr, outs)
}

func evalCall(ctx *Context, expr *ir.CallExpr) ([]elements.Element, error) {
	// Fetch the function and check that it is callable.
	fnNode, err := ctx.evalExpr(expr.Callee.X)
	if err != nil {
		return nil, err
	}
	fn, ok := fnNode.(elements.Func)
	if !ok {
		return nil, fmterr.Errorf(ctx.File().FileSet(), expr.Source(), "%T is not callable", fnNode)
	}

	// Evaluate the arguments to pass to the function.
	args := make([]elements.Element, len(expr.Args))
	for i, arg := range expr.Args {
		el, err := ctx.evalExpr(arg)
		if err != nil {
			return nil, err
		}
		args[i] = el
	}
	return fn.Call(ctx, expr, args)
}
