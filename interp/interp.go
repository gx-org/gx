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
	"github.com/gomlx/compute/dtypes/gotype"
	"github.com/gx-org/gx/api/options"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/interp/numbers"
	"github.com/gx-org/gx/interp/context"
	"github.com/gx-org/gx/interp/elements"
	"github.com/gx-org/gx/interp/engine"
	"github.com/gx-org/gx/interp/procoptions"
	"github.com/gx-org/gx/src/gx/ast"
)

// Base provides everything required to create new interpreters.
type Base struct {
	reg  *engine.Registry
	core *context.Core

	options *procoptions.Options
}

// New returns a new interpreter.
func New(reg *engine.Registry, options []options.PackageOption) (*Base, error) {
	itp := &Base{reg: reg}
	itp.core = context.New(itp, reg.Importer)
	fitp, err := itp.ForFile(builtinFile)
	if err != nil {
		return itp, err
	}
	itp.options, err = procoptions.New(fitp.env, options)
	if err != nil {
		return itp, nil
	}
	return itp, initBuiltinsFor(fitp)
}

// ForFile returns an interpreter for a file context.
func (itp *Base) ForFile(file *ir.File) (*Interpreter, error) {
	ctx, err := itp.core.NewFileContext(file)
	return newInterp(itp.reg, ctx, itp.reg.HostEngine), err
}

// Registry used by the interpreter.
func (itp *Base) Registry() *engine.Registry {
	return itp.reg
}

// Interpreter returns an interpreter given the scope of a file from within a package.
type Interpreter struct {
	env *engine.Env
}

var _ ir.Evaluator = (*Interpreter)(nil)

func newIREval(env *engine.Env) ir.Evaluator {
	return newInterp(env.Registry(), env.Context(), env.Engine())
}

func newInterp(reg *engine.Registry, ctx *context.Context, eng engine.Engine) *Interpreter {
	fitp := &Interpreter{}
	fitp.env = engine.NewEnv(reg, ctx, newIREval, eng)
	return fitp
}

var (
	errorIdent  = &ast.Ident{Name: "Error"}
	selectError = &ir.SelectorExpr{
		Src:  &ast.SelectorExpr{Sel: errorIdent},
		Stor: &ir.LocalVarStorage{Src: errorIdent},
	}
)

var compEvalNature = ir.FuncNature{CompEval: true}

func (fitp *Interpreter) compEvalExpr(expr ir.Expr) (ir.Element, error) {
	return fitp.env.For(compEvalNature).ExprEval().EvalExpr(expr)
}

// EvalExpr evaluates an expression for a given context.
func (fitp *Interpreter) EvalExpr(expr ir.Expr) (ir.Element, error) {
	return evalExpr(fitp, expr)
}

// Env returns the environment.
func (fitp *Interpreter) Env() *engine.Env {
	return fitp.env
}

// SubInterp returns a new interpreter with additional values defined in the context.
// If file is not nil, a new context is built for the file scope, discarding the
// existing context.
func (fitp *Interpreter) SubInterp(file *ir.File, vals map[string]ir.Element) (*Interpreter, error) {
	ctx := fitp.env.Context()
	var err error
	if file != nil && file.Package != nil {
		core := fitp.Context().Core()
		ctx, err = core.NewFileContext(file)
		fitp = newInterp(fitp.env.Registry(), ctx, fitp.engine())
	}
	if vals == nil {
		return fitp, err
	}
	return newInterp(fitp.env.Registry(), ctx.Sub(vals), fitp.engine()), err
}

func (fitp *Interpreter) engine() engine.Engine {
	return fitp.env.Engine()
}

// Sub returns a new interpreter with additional values defined in the context.
// If file is not nil, a new context is built for the file scope, discarding the
// existing context.
func (fitp *Interpreter) Sub(file *ir.File, vals map[string]ir.Element) (ir.Evaluator, error) {
	return fitp.SubInterp(file, vals)
}

// EvalFunc evaluates a function.
func (fitp *Interpreter) EvalFunc(f ir.PkgFunc, call *ir.FuncCallExpr, args []ir.Element) ([]ir.Element, error) {
	fnEl := elements.NewFunc(f, nil)
	return fnEl.Call(fitp.env, call, nil, args)
}

// Context used by the interpreter.
func (fitp *Interpreter) Context() *context.Context {
	return fitp.env.Context()
}

// File returns the current file of the current execution.
func (fitp *Interpreter) File() *ir.File {
	return fitp.Context().File()
}

func elementFromInt[T gotype.ScalarNotComplex](fitp *Interpreter, val T, tp ir.Type) (engine.NumericalElement, error) {
	cst, err := numbers.NewConstant(tp, val)
	if err != nil {
		return nil, err
	}
	return fitp.engine().ElementFromHostValue(fitp.env.ExprEval(), cst)
}

// String representation of the receiver.
func (fitp *Interpreter) String() string {
	return fitp.env.String()
}
