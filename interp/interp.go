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

// Package interp evaluates GX code given an evaluator.
//
// All values in the interpreter are represented in elements.
// The GX Context evaluates GX code represented as an
// intermediate representation (IR) tree
// (see [github.com/gx-org/gx/build/ir]),
// evaluates a function given a receiver and arguments passed as interpreter elements.
package interp

import (
	"fmt"
	"go/ast"

	"github.com/pkg/errors"
	"github.com/gx-org/gx/api/options"
	"github.com/gx-org/gx/api/values"
	"github.com/gx-org/gx/build/fmterr"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/interp/context"
	"github.com/gx-org/gx/interp/elements"
	"github.com/gx-org/gx/interp/engine"
	"github.com/gx-org/gx/interp/fun"
	"github.com/gx-org/gx/interp/materialise"
	"github.com/gx-org/gx/interp/procoptions"
)

// Interpreter runs GX code given an evaluator and package options.
type Interpreter struct {
	eng     engine.Engine
	funFact fun.Factory
	core    *context.Core

	options *procoptions.Options
}

// New returns a new interpreter.
func New(eng engine.Engine, funFact fun.Factory, options []options.PackageOption) (*Interpreter, error) {
	itp := &Interpreter{eng: eng, funFact: funFact}
	var errs fmterr.Errors
	var err error
	if itp.options, err = procoptions.New(eng, options); err != nil {
		errs.Append(err)
	}
	itp.core, err = context.New(itp, eng.Importer())
	if err != nil {
		errs.Append(err)
	}
	return itp, errs.ToError()
}

// Core returns the core context.
func (itp *Interpreter) Core() *context.Core {
	return itp.core
}

// NewFunc creates function elements from function IRs.
func (itp *Interpreter) NewFunc(fn ir.Func, recv *fun.Receiver) fun.Func {
	return itp.funFact.NewFunc(fn, recv)
}

// ForFile returns an interpreter for a file context.
func (itp *Interpreter) ForFile(file *ir.File) (*FileScope, error) {
	ctx, err := itp.core.NewFileContext(file)
	fitp, _ := newFileScope(ctx, itp.funFact, file)
	return fitp, err

}

// Engine returns the evaluator used by the interpreter
func (itp *Interpreter) Engine() engine.Engine {
	return itp.eng
}

// FileScope returns an interpreter given the scope of a file from within a package.
type FileScope struct {
	ctx *context.Context
	env *fun.CallEnv
}

func newFileScope(ctx *context.Context, funFact fun.Factory, file *ir.File) (*FileScope, error) {
	fitp := &FileScope{ctx: ctx}
	fitp.env = fun.NewCallEnv(ctx, fitp, funFact)
	return fitp, nil
}

var (
	errorIdent  = &ast.Ident{Name: "Error"}
	selectError = &ir.SelectorExpr{
		Src:  &ast.SelectorExpr{Sel: errorIdent},
		Stor: &ir.LocalVarStorage{Src: errorIdent},
	}
)

// ToCompEvalError converts an element to an error.
// If the conversion goes wrong, then the first error (corresponding to the element) is nil,
// and the second error indicates the conversion error.
// In other word, the first error is an error that needs to be reported to the user,
// the second error is an internal error in the compiler.
func (fitp *FileScope) ToCompEvalError(src ast.Expr, el ir.Element) (ir.CompEvalError, error) {
	if el == nil {
		return nil, nil
	}
	if elements.IsNil(el) {
		return nil, nil
	}
	isErr, cpErr, err := el.Type().AssignableTo(fitp, ir.ErrorType())
	if !isErr {
		return nil, errors.Errorf("cannot convert %T to error", el.Type().ReferString(fitp.File()))
	}
	if unErr := ir.UnifyErr(cpErr, err); unErr != nil {
		return nil, unErr
	}
	methods, isSelector := el.(*fun.NamedType)
	if !isSelector {
		return nil, errors.Errorf("cannot convert %T to a method selector", el)
	}
	errorMethod, err := methods.Select(selectError)
	if err != nil {
		return nil, err
	}
	errorFun, isFun := errorMethod.(fun.Func)
	if !isFun {
		return nil, errors.Errorf("%T not a function", errorMethod)
	}
	recv := fun.NewReceiver(methods, errorFun.Func())
	errorFun = NewRunFunc(errorFun.Func(), recv)
	errorExpr := &ir.FuncCallExpr{
		Callee: ir.ErrorCallee(src, errorFun.Func().FuncType()),
	}
	outs, err := errorFun.Call(fitp.env, errorExpr, nil)
	if err != nil {
		return nil, fmt.Errorf("cannot call Error function: %w", err)
	}
	errElement, err := ToSingleElement(fitp, errorExpr, outs)
	if err != nil {
		return nil, err
	}
	str, err := elements.StringFromElement(errElement)
	if err != nil {
		return nil, err
	}
	return ir.CompEvalError(errors.New(str)), nil
}

// EvalExpr evaluates an expression for a given context.
func (fitp *FileScope) EvalExpr(expr ir.Expr) (ir.Element, error) {
	return evalExpr(fitp, expr)
}

// Engine returns the evaluator used by the interpreter
func (fitp *FileScope) Engine() engine.Engine {
	return fitp.env.Engine()
}

// Materialiser returns the materialiser to convert elements into graph nodes.
func (fitp *FileScope) Materialiser() materialise.Materialiser {
	return fitp.Engine().ArrayOps().(materialise.Materialiser)
}

// Sub returns a new interpreter with additional values defined in the context.
func (fitp *FileScope) Sub(file *ir.File, vals *context.SubMap) (*FileScope, error) {
	var err error
	if file != nil && file != fitp.File() {
		fitp, err = newFileScope(fitp.ctx, fitp.env.FuncEval(), file)
	}
	sub := &FileScope{ctx: fitp.ctx.Sub(vals)}
	sub.env = fun.NewCallEnv(sub.ctx, sub, fitp.env.FuncEval())
	return sub, err
}

// NewFunc creates function elements from function IRs.
func (fitp *FileScope) NewFunc(fn ir.Func, recv *fun.Receiver) fun.Func {
	return fitp.env.FuncEval().NewFunc(fn, recv)
}

// EvalFunc evaluates a function.
func (fitp *FileScope) EvalFunc(f ir.PkgFunc, call *ir.FuncCallExpr, args []ir.Element) ([]ir.Element, error) {
	fnEl := NewRunFunc(f, nil)
	return fnEl.Call(fitp.env, call, args)
}

// Context used by the interpreter.
func (fitp *FileScope) Context() *context.Context {
	return fitp.ctx
}

// File returns the current file of the current execution.
func (fitp *FileScope) File() *ir.File {
	return fitp.ctx.File()
}

func (fitp *FileScope) elementFromAtom(expr ir.Expr, val values.Array) (engine.NumericalElement, error) {
	typ, cpErr, err := fitp.env.ToConcrete(expr.Expr(), expr.Type())
	if unErr := ir.UnifyErr(cpErr, err); unErr != nil {
		return nil, unErr
	}
	return fitp.Engine().ArrayOps().ElementFromAtom(fitp.File(), val, expr, typ)
}

// String representation of the receiver.
func (fitp *FileScope) String() string {
	return fitp.env.String()
}
