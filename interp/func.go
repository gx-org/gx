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
	"go/ast"
	"strings"

	"github.com/pkg/errors"
	"github.com/gx-org/gx/api/values"
	"github.com/gx-org/gx/build/fmterr"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/interp/flatten"
	"github.com/gx-org/gx/interp/context"
	"github.com/gx-org/gx/interp/elements"
	"github.com/gx-org/gx/interp/evaluator"
	"github.com/gx-org/gx/interp/fun"
)

// FuncBuiltin defines a builtin function provided by a backend.
type FuncBuiltin func(ctx evaluator.Env, call elements.CallAt, fn fun.Func, irFunc *ir.FuncBuiltin, args []ir.Element) ([]ir.Element, error)

type funcBase struct {
	fn   ir.Func
	recv *fun.Receiver
}

var _ fun.Func = (*funcBase)(nil)

// NewRunFunc creates a function given an IR and a receiver.
// The function is run when being called.
func NewRunFunc(fn ir.Func, recv *fun.Receiver) fun.Func {
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
	case *ir.FuncKeyword:
		return &funcKeyword{
			funcBase: funcBase{fn: fnT},
			fnT:      fnT,
		}
	case *ir.Macro:
		return &funcMacro{
			funcBase: funcBase{fn: fnT, recv: recv},
			fnT:      fnT,
		}
	}
	return &funcBase{fn: fn, recv: recv}
}

func (st *funcBase) toFuncBuiltin(impl ir.FuncImpl) (FuncBuiltin, error) {
	if impl == nil {
		return nil, errors.Errorf("function %s has no implementation", st.fn.ShortString())
	}
	blt, isBuiltin := impl.Implementation().(FuncBuiltin)
	if !isBuiltin {
		return nil, errors.Errorf("type %T is not a function builtin implementation", impl)
	}
	if blt == nil {
		return nil, errors.Errorf("function %s has no implementation", st.fn.ShortString())
	}
	return blt, nil
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
func (st *funcBase) Recv() *fun.Receiver {
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
func (st *funcBase) Call(ctx *fun.CallEnv, call *ir.CallExpr, args []ir.Element) ([]ir.Element, error) {
	return nil, fmterr.Internalf(ctx.File().FileSet(), st.fn.Source(), "function type %T not supported", st.fn)
}

// String representation of the node.
func (st *funcBase) String() string {
	return st.fn.String()
}

type funcDecl struct {
	funcBase
	fnT *ir.FuncDecl
}

func (f *funcDecl) Call(env *fun.CallEnv, call *ir.CallExpr, args []ir.Element) (outs []ir.Element, err error) {
	if f.fnT.Body == nil {
		return nil, fmterr.Errorf(env.File().FileSet(), f.fnT.Source(), "missing function body")
	}
	// Create a new function frame.
	funcFrame, err := env.Context().PushFuncFrame(f.fnT)
	if err != nil {
		return nil, err
	}
	defer env.Context().PopFrame()
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
	assignTypeParameters(call.Callee, funcFrame)
	assignArgumentValues(f.fnT.FType, funcFrame, args)
	ctx := newFileScope(env.Context(), env.FuncEval(), f.fnT.File())
	// Evaluate the function within the frame.
	return evalFuncBody(ctx, f.fnT.Body)
}

type funcBuiltin struct {
	funcBase
	fnT *ir.FuncBuiltin
}

func (f *funcBuiltin) Call(env *fun.CallEnv, call *ir.CallExpr, args []ir.Element) (outs []ir.Element, err error) {
	defer func() {
		if err != nil {
			err = fmterr.Position(env.File().FileSet(), call.Expr(), err)
		}
	}()
	impl, err := f.toFuncBuiltin(f.fnT.Impl)
	if err != nil {
		return nil, err
	}
	return impl(env, elements.NewNodeAt[*ir.CallExpr](env.File(), call), f, f.fnT, args)
}

type funcKeyword struct {
	funcBase
	fnT *ir.FuncKeyword
}

func (f *funcKeyword) Call(env *fun.CallEnv, call *ir.CallExpr, args []ir.Element) (outs []ir.Element, err error) {
	defer func() {
		if err != nil {
			err = fmterr.Position(env.File().FileSet(), call.Expr(), err)
		}
	}()
	impl, err := f.toFuncBuiltin(f.fnT.Impl)
	if err != nil {
		return nil, err
	}
	return impl(env, elements.NewNodeAt[*ir.CallExpr](env.File(), call), f, nil, args)
}

type funcMacro struct {
	funcBase
	fnT *ir.Macro
}

func (f *funcMacro) Call(env *fun.CallEnv, call *ir.CallExpr, args []ir.Element) (outs []ir.Element, err error) {
	defer func() {
		if err != nil {
			err = fmterr.Position(env.File().FileSet(), call.Expr(), err)
		}
	}()
	synth, err := f.fnT.BuildSynthetic(env.File(), call, f.fnT, args)
	if err != nil {
		return nil, err
	}
	return []ir.Element{synth}, nil
}

func evalFuncBody(fitp *FileScope, body *ir.BlockStmt) ([]ir.Element, error) {
	outs, stop, err := evalBlockStmt(fitp, body)
	if !stop {
		// No return statement was processed during the eval of the function.
		return nil, fmterr.Errorf(fitp.File().FileSet(), body.Src, "missing return")
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

func assignTypeParameters(callee ir.Callee, funcFrame *context.Frame) {
	funRef, ok := callee.(*ir.FuncValExpr)
	if !ok {
		return
	}
	for _, tpParam := range funRef.T.TypeParamsValues {
		funcFrame.Define(tpParam.Field.Name.Name, tpParam.Typ)
	}
}

func assignArgumentValues(funcType *ir.FuncType, funcFrame *context.Frame, args []ir.Element) {
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
func (itp *Interpreter) EvalFunc(fn *ir.FuncDecl, in *elements.InputElements) (outs []ir.Element, err error) {
	if fn.Body == nil {
		return nil, errors.Errorf("%s: missing function body", fn.Name())
	}
	fitp, err := itp.ForFile(fn.File())
	if err != nil {
		return nil, err
	}
	// Create a frame for the function to evaluate.
	frame, err := fitp.ctx.PushFuncFrame(fn)
	if err != nil {
		return nil, err
	}
	defer fitp.ctx.PopFrame()
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
	outs, err = evalFuncBody(fitp, fn.Body)
	return
}
