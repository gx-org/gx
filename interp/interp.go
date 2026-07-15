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
	"github.com/gx-org/gx/internal/concrete"
	"github.com/gx-org/gx/internal/interp/numbers"
	"github.com/gx-org/gx/internal/interp/proxies"
	"github.com/gx-org/gx/interp/context"
	"github.com/gx-org/gx/interp/elements"
	"github.com/gx-org/gx/interp/engine"
	"github.com/gx-org/gx/interp/fun"
	"github.com/gx-org/gx/interp/materialise"
	"github.com/gx-org/gx/interp/procoptions"
)

// Base provides everything required to create new interpreters.
type Base struct {
	eng     engine.Engine
	funFact fun.Factory
	runners fun.Runners
	core    *context.Core

	options *procoptions.Options
}

// New returns a new interpreter.
func New(eng engine.Engine, funFact fun.Factory, runners fun.Runners, options []options.PackageOption) (*Base, error) {
	itp := &Base{eng: eng, funFact: funFact, runners: runners}
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

// ForFile returns an interpreter for a file context.
func (itp *Base) ForFile(file *ir.File) (*Interpreter, error) {
	ctx, err := itp.core.NewFileContext(file)
	return toInterp(ctx, itp.eng, itp.funFact, itp.runners), err

}

// Engine returns the evaluator used by the interpreter
func (itp *Base) Engine() engine.Engine {
	return itp.eng
}

// Interpreter returns an interpreter given the scope of a file from within a package.
type Interpreter struct {
	env *fun.CallEnv
}

var _ ir.Evaluator = (*Interpreter)(nil)

func toInterp(ctx *context.Context, eng engine.Engine, funFact fun.Factory, runners fun.Runners) *Interpreter {
	fitp := &Interpreter{}
	fitp.env = fun.NewCallEnv(ctx, fitp, eng, funFact, runners)
	return fitp
}

var (
	errorIdent  = &ast.Ident{Name: "Error"}
	selectError = &ir.SelectorExpr{
		Src:  &ast.SelectorExpr{Sel: errorIdent},
		Stor: &ir.LocalVarStorage{Src: errorIdent},
	}
)

// toCompEvalError converts an element to a compiler error.
func (fitp *Interpreter) toCompEvalError(el ir.Element) (err error) {
	if el == nil {
		return nil
	}
	if elements.IsNil(el) {
		return nil
	}
	methods, isSelector := el.(*fun.NamedType)
	if !isSelector {
		return errors.Errorf("cannot convert %T to a method selector", el)
	}
	errorMethod, err := methods.Select(selectError)
	if err != nil {
		return err
	}
	errorFun, isFun := errorMethod.(fun.Func)
	if !isFun {
		return errors.Errorf("%T not a function", errorMethod)
	}
	recv := fun.NewReceiver(methods, errorFun.Func())
	errorFun = NewRunFunc(errorFun.Func(), recv)
	errorExpr := &ir.FuncCallExpr{
		Callee: ir.ErrorCallee(errorFun.Func()),
	}
	outs, err := errorFun.Call(fitp.env, errorExpr, nil)
	if err != nil {
		return fmt.Errorf("cannot call Error function: %w", err)
	}
	if len(outs) != 1 {
		return fmt.Errorf("Error function returned %d element(s), expect 1", len(outs))
	}
	errElement := outs[0]
	if proxies.IsProxy(errElement) {
		return nil
	}
	str, err := elements.StringFromElement(errElement)
	if err != nil {
		return err
	}
	return ir.NewCompileError(errors.New(str))
}

// EvalExpr evaluates an expression for a given context.
func (fitp *Interpreter) EvalExpr(expr ir.Expr) (ir.Element, error) {
	return evalExpr(fitp, expr)
}

// Engine returns the evaluator used by the interpreter
func (fitp *Interpreter) Engine() engine.Engine {
	return fitp.env.Engine()
}

// Materialiser returns the materialiser to convert elements into graph nodes.
func (fitp *Interpreter) Materialiser() materialise.Materialiser {
	return fitp.Engine().ArrayOps().(materialise.Materialiser)
}

// SubInterp returns a new interpreter with additional values defined in the context.
// If file is not nil, a new context is built for the file scope, discarding the
// existing context.
func (fitp *Interpreter) SubInterp(file *ir.File, vals map[string]ir.Element) (*Interpreter, error) {
	ctx := fitp.env.Context()
	var err error
	if file != nil {
		core := fitp.Context().Core()
		ctx, err = core.NewFileContext(file)
		fitp = toInterp(ctx, fitp.Engine(), fitp.env.FuncEval(), fitp.env.Runners())
	}
	if vals == nil {
		return fitp, nil
	}
	ctx = ctx.Sub(vals)
	sub := &Interpreter{}
	sub.env = fun.NewCallEnv(ctx, sub, fitp.Engine(), fitp.env.FuncEval(), fitp.env.Runners())
	return sub, err
}

// Sub returns a new interpreter with additional values defined in the context.
// If file is not nil, a new context is built for the file scope, discarding the
// existing context.
func (fitp *Interpreter) Sub(file *ir.File, vals map[string]ir.Element) (ir.Evaluator, error) {
	return fitp.SubInterp(file, vals)
}

// NewFunc creates function elements from function IRs.
func (fitp *Interpreter) NewFunc(fn ir.Func, recv *fun.Receiver) fun.Func {
	return fitp.env.FuncEval().NewFunc(fn, recv)
}

// EvalFunc evaluates a function.
func (fitp *Interpreter) EvalFunc(f ir.PkgFunc, call *ir.FuncCallExpr, args []ir.Element) ([]ir.Element, error) {
	fnEl := NewRunFunc(f, nil)
	return fnEl.Call(fitp.env, call, args)
}

// Context used by the interpreter.
func (fitp *Interpreter) Context() *context.Context {
	return fitp.env.Context()
}

// File returns the current file of the current execution.
func (fitp *Interpreter) File() *ir.File {
	return fitp.Context().File()
}

func (fitp *Interpreter) elementFromAtomLit(expr *ir.NumberCastExpr) (engine.NumericalElement, error) {
	var number engine.AtomLitElement
	switch xT := expr.X.(type) {
	case *ir.NumberFloat:
		number = numbers.NewFloat(xT.Val, expr)
	case *ir.NumberInt:
		number = numbers.NewInt(xT.Val, expr)
	default:
		return nil, errors.Errorf("cannot convert %T to an atomic literal element: not supported", xT)
	}
	return fitp.Engine().ArrayOps().ElementFromAtomLit(fitp.File(), number)
}

func (fitp *Interpreter) elementFromAtom(expr ir.Expr, val values.Array) (engine.NumericalElement, error) {
	typ, err := concrete.Concrete(fitp.env.ExprEval(), expr.Expr(), expr.Type())
	if err != nil {
		return nil, err
	}
	return fitp.Engine().ArrayOps().ElementFromAtom(fitp.File(), val, expr, typ)
}

// String representation of the receiver.
func (fitp *Interpreter) String() string {
	return fitp.env.String()
}
