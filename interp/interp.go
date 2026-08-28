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
	"go/ast"

	"github.com/gx-org/backend/dtypes"
	"github.com/gx-org/gx/api/options"
	"github.com/gx-org/gx/build/fmterr"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/interp/numbers"
	"github.com/gx-org/gx/interp/context"
	"github.com/gx-org/gx/interp/elements"
	"github.com/gx-org/gx/interp/engine"
	"github.com/gx-org/gx/interp/materialise"
	"github.com/gx-org/gx/interp/procoptions"
)

// Base provides everything required to create new interpreters.
type Base struct {
	eng     engine.Engine
	runners engine.Runners
	core    *context.Core

	options *procoptions.Options
}

// New returns a new interpreter.
func New(eng engine.Engine, runners engine.Runners, options []options.PackageOption) (*Base, error) {
	itp := &Base{eng: eng, runners: runners}
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
	return toInterp(ctx, itp.eng, itp.runners), err

}

// Engine returns the evaluator used by the interpreter
func (itp *Base) Engine() engine.Engine {
	return itp.eng
}

// Interpreter returns an interpreter given the scope of a file from within a package.
type Interpreter struct {
	env *engine.Env
}

var _ ir.Evaluator = (*Interpreter)(nil)

func toInterp(ctx *context.Context, eng engine.Engine, runners engine.Runners) *Interpreter {
	fitp := &Interpreter{}
	fitp.env = engine.NewEnv(ctx, fitp, eng, runners)
	return fitp
}

var (
	errorIdent  = &ast.Ident{Name: "Error"}
	selectError = &ir.SelectorExpr{
		Src:  &ast.SelectorExpr{Sel: errorIdent},
		Stor: &ir.LocalVarStorage{Src: errorIdent},
	}
)

// EvalExpr evaluates an expression for a given context.
func (fitp *Interpreter) EvalExpr(expr ir.Expr) (ir.Element, error) {
	return evalExpr(fitp, expr)
}

// Env returns the environment.
func (fitp *Interpreter) Env() *engine.Env {
	return fitp.env
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
	if file != nil && file.Package != nil {
		core := fitp.Context().Core()
		ctx, err = core.NewFileContext(file)
		fitp = toInterp(ctx, fitp.Engine(), fitp.env.Runners())
	}
	if vals == nil {
		return fitp, nil
	}
	ctx = ctx.Sub(vals)
	sub := &Interpreter{}
	sub.env = engine.NewEnv(ctx, sub, fitp.Engine(), fitp.env.Runners())
	return sub, err
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

func elementFromInt[T dtypes.AlgebraType](fitp *Interpreter, val T, tp ir.Type) (engine.NumericalElement, error) {
	cst, err := numbers.NewConstant(tp, val)
	if err != nil {
		return nil, err
	}
	return fitp.Engine().ArrayOps().ElementFromHostValue(fitp.env.ExprEval(), cst)
}

// String representation of the receiver.
func (fitp *Interpreter) String() string {
	return fitp.env.String()
}
