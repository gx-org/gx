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

package context

import (
	"fmt"
	"go/ast"
	"strings"

	"github.com/pkg/errors"
	"github.com/gx-org/gx/api/options"
	"github.com/gx-org/gx/api/values"
	"github.com/gx-org/gx/build/fmterr"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/interp/flatten"
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

// Func returns the function represented by the node.
func (st *funcBase) Func() ir.Func {
	return st.fn
}

// Recv returns the receiver of the function or nil if the function has no receiver.
func (st *funcBase) Recv() *elements.Receiver {
	return st.recv
}

// Unflatten creates a GX value from the next handles available in the parser.
func (st *funcBase) Unflatten(handles *flatten.Parser) (values.Value, error) {
	return values.NewIRNode(st.fn)
}

// Kind of the element.
func (*funcBase) Kind() ir.Kind {
	return ir.FuncKind
}

// Call the function.
func (st *funcBase) Call(ctx ir.Evaluator, call *ir.CallExpr, args []ir.Element) ([]ir.Element, error) {
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

func (f *funcDecl) Call(fctx ir.Evaluator, call *ir.CallExpr, args []ir.Element) (outs []ir.Element, err error) {
	ctx := fctx.(*Context)
	if f.fnT.Body == nil {
		return nil, fmterr.Errorf(ctx.File().FileSet(), f.fnT.Source(), "missing function body")
	}
	// Create a new function frame.
	funcFrame, err := ctx.pushFuncFrame(f.fnT)
	if err != nil {
		return nil, err
	}
	defer ctx.PopFrame()
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

func (f *funcBuiltin) Call(fctx ir.Evaluator, call *ir.CallExpr, args []ir.Element) (outs []ir.Element, err error) {
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

func (f *funcLit) Call(fctx ir.Evaluator, call *ir.CallExpr, args []ir.Element) (outs []ir.Element, err error) {
	ctx := fctx.(*Context)
	// TODO(degris): remove this hack.
	f.fnT.FFile = fctx.File()
	return ctx.evaluator.CallFuncLit(ctx, f.fnT, args)
}

func evalFuncBody(ctx *Context, body *ir.BlockStmt) ([]ir.Element, error) {
	outs, stop, err := ctx.interp.EvalStmt(ctx, body)
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

func assignArgumentValues(funcType *ir.FuncType, funcFrame *blockFrame, args []ir.Element) {
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

// EvalFunc evaluates a function.
func EvalFunc(interp Interpreter, eval Evaluator, fn *ir.FuncDecl, in *elements.InputElements, options []options.PackageOption) (outs []ir.Element, err error) {
	if fn.Body == nil {
		return nil, errors.Errorf("%s: missing function body", fn.Name())
	}
	ectx, err := New(interp, eval, options)
	if err != nil {
		return nil, err
	}
	ctx, err := ectx.newFileContext(fn.File())
	if err != nil {
		return nil, err
	}
	// Create a frame for the function to evaluate.
	frame, err := ctx.pushFuncFrame(fn)
	if err != nil {
		return nil, err
	}
	defer ctx.PopFrame()
	// Add the result names to the Context.
	for _, resultName := range fieldNames(fn.FType.Results.List) {
		frame.Define(resultName.Name, nil)
	}
	// Add the receiver to the Context.
	recv := fn.FType.ReceiverField()
	if recv != nil {
		if in.Receiver == nil {
			return nil, errors.Errorf("function has a receiver but a nil value has been passed as a receiver value")
		}
		frame.Define(recv.Name.Name, in.Receiver)
	}
	// Add the parameters to the Context.
	paramFields := fn.FType.Params.Fields()
	for i, param := range paramFields {
		if i >= len(in.Args) {
			missingParams := paramFields[len(in.Args):]
			builder := strings.Builder{}
			for n, param := range missingParams {
				if n > 0 {
					builder.WriteString(", ")
				}
				builder.WriteString(param.Name.String())
			}
			return nil, errors.Errorf("missing parameter(s): %s", builder.String())
		}
		frame.Define(param.Name.Name, in.Args[i])
	}
	// Evaluate the function body.
	outs, err = evalFuncBody(ctx, fn.Body)
	return
}
