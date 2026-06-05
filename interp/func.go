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
	"github.com/gx-org/gx/build/ir/irkind"
	"github.com/gx-org/gx/internal/interp/flatten"
	"github.com/gx-org/gx/interp/context"
	"github.com/gx-org/gx/interp/elements"
	"github.com/gx-org/gx/interp/engine"
	"github.com/gx-org/gx/interp/fun"
)

// FuncBuiltin defines a builtin function provided by a backend.
type FuncBuiltin func(env engine.Env, call *ir.FuncCallExpr, recv ir.Element, args []ir.Element) ([]ir.Element, error)

type caller func(env *fun.CallEnv, call *ir.FuncCallExpr, recv engine.Copier, args []ir.Element) ([]ir.Element, error)

type elFunc struct {
	fn      ir.Func
	recv    *fun.Receiver
	storage ir.Storage

	call caller
}

var (
	_ fun.Func     = (*elFunc)(nil)
	_ ir.WithStore = (*elFunc)(nil)
)

// NewRunFunc creates a function given an IR and a receiver.
// The function is run when being called.
func NewRunFunc(fn ir.Func, recv *fun.Receiver) fun.Func {
	base := elFunc{fn: fn, recv: recv}
	switch fnT := fn.(type) {
	case *ir.FuncDecl:
		base.storage = fnT
		base.call = funcDecl{fnT: fnT}.callDecl
	case *ir.FuncBuiltin:
		base.storage = fnT
		base.call = funcBuiltin{fun: fnT, impl: fnT.Impl}.callBuiltin
	case *ir.FuncKeyword:
		base.storage = fnT
		base.call = funcBuiltin{fun: fnT, impl: fnT.Impl}.callBuiltin
	case *ir.Macro:
		base.storage = fnT
		base.call = funcMacro{fnT: fnT}.callMacro
	}
	return &base
}

// Type of the function.
func (f *elFunc) Type() ir.Type {
	return f.fn.FuncType()
}

// Func returns the function represented by the node.
func (f *elFunc) Func() ir.Func {
	return f.fn
}

// Recv returns the receiver of the function or nil if the function has no receiver.
func (f *elFunc) Recv() *fun.Receiver {
	return f.recv
}

// Unflatten creates a GX value from the next handles available in the parser.
func (f *elFunc) Unflatten(handles *flatten.Parser) (values.Value, error) {
	return values.NewIRNode(f.fn)
}

// Kind of the element.
func (*elFunc) Kind() irkind.Kind {
	return irkind.Func
}

// Storage of the function.
func (f *elFunc) Store() ir.Storage {
	return f.storage
}

// Call the function.
func (f *elFunc) Call(env *fun.CallEnv, call *ir.FuncCallExpr, args []ir.Element) ([]ir.Element, error) {
	if f.call == nil {
		return nil, fmterr.Internalf(env.File().FileSet(), f.fn.Node(), "function type %T not supported", f.fn)
	}
	var recv *fun.NamedType
	if f.Recv() != nil {
		recv = f.Recv().Element
	}
	return f.call(env, call, recv, args)
}

// String representation of the node.
func (f *elFunc) String() string {
	return f.fn.DefineString(nil)
}

type funcDecl struct {
	fnT *ir.FuncDecl
}

func (f funcDecl) callDecl(env *fun.CallEnv, call *ir.FuncCallExpr, recv engine.Copier, args []ir.Element) (outs []ir.Element, err error) {
	return env.Runners().FuncDecl(f.fnT, env, call, recv, args)
}

type funcBuiltin struct {
	fun  ir.Func
	impl ir.FuncImpl
}

func (f funcBuiltin) callBuiltin(env *fun.CallEnv, call *ir.FuncCallExpr, recv engine.Copier, args []ir.Element) (outs []ir.Element, err error) {
	return env.Runners().Builtin(f.fun, f.impl, env, call, recv, args)
}

type funcMacro struct {
	fnT *ir.Macro
}

func (f funcMacro) callMacro(env *fun.CallEnv, call *ir.FuncCallExpr, _ engine.Copier, args []ir.Element) (outs []ir.Element, err error) {
	defer func() {
		if err != nil {
			err = fmterr.Error(env.File().FileSet(), call.Expr(), err)
		}
	}()
	synth, err := f.fnT.BuildSynthetic(env.File(), call, f.fnT, args)
	if err != nil {
		return nil, err
	}
	return []ir.Element{synth}, nil
}

func evalFuncBody(fitp *Interpreter, body *ir.BlockStmt) ([]ir.Element, error) {
	outs, _, err := evalBlockStmt(fitp, body)
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

func toConcreteType(ctx *context.Context, src ast.Node, frame *context.Frame, tp ir.Type) (ir.Type, error) {
	typeParam, isTypeParam := tp.(*ir.GenericTypeParam)
	if !isTypeParam {
		return tp, nil
	}
	el, err := frame.Find(typeParam.OrigField().Name)
	if err != nil {
		return nil, fmterr.Internalf(ctx.File().FileSet(), src, "cannot cast to %s: %v", tp.ReferString(ctx.File()), err)
	}
	tp, isType := ir.BareValue(el).(ir.Type)
	if !isType {
		return nil, fmterr.Internalf(ctx.File().FileSet(), src, "element %T is not a type", el)
	}
	return tp, nil
}

func assignTypeParameters(ctx *context.Context, callee ir.Callee, funcFrame *context.Frame, args []ir.Element) []ir.Element {
	funRef, ok := callee.(*ir.FuncValExpr)
	if !ok {
		return args
	}
	genVals := funRef.FuncType().GenericValues
	for i, genVal := range genVals {
		funcFrame.Define(genVal.Generic().NameDef(), args[i])
	}
	return args[len(genVals):]
}

func assignArgumentValues(ftype *ir.FuncType, funcFrame *context.Frame, args []ir.Element) error {
	fields := ftype.Params.Fields()
	if len(args) != len(fields) {
		return fmterr.Internal(errors.Errorf("number of arguments (%d) does not match the number of parameters (%d) in function type %s", len(args), len(fields), ftype.ReferString(nil)))
	}
	// For each parameter of the function, assign its argument value to the frame.
	for i, arg := range args {
		arg = engine.Copy(arg)
		funcFrame.Define(fields[i].Name, arg)
	}
	return nil
}

// EvalFunc evaluates a function.
func (itp *Base) EvalFunc(fn *ir.FuncDecl, in *elements.InputElements) (outs []ir.Element, err error) {
	if fn.Body == nil {
		return nil, errors.Errorf("%s: missing function body", fn.Name())
	}
	fitp, err := itp.ForFile(fn.File())
	if err != nil {
		return nil, err
	}
	// Create a frame for the function to evaluate.
	frame, err := fitp.Context().PushFuncFrame(fn)
	if err != nil {
		return nil, err
	}
	defer fitp.Context().PopFrame()
	// Add the result names to the Context.
	if fn.FType.Results != nil {
		for _, resultName := range fieldNames(fn.FType.Results.List) {
			frame.Define(resultName, nil)
		}
	}
	// Add the receiver to the Context.
	recv := fn.FType.ReceiverField()
	if recv != nil {
		if in.Receiver == nil {
			return nil, errors.Errorf("function has a receiver but a nil value has been passed as a receiver value")
		}
		frame.Define(recv.Name, in.Receiver)
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
		frame.Define(param.Name, in.Args[i])
	}
	// Evaluate the function body.
	outs, err = evalFuncBody(fitp, fn.Body)
	return
}
