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
	"github.com/gx-org/gx/api/options"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/interp/context"
	"github.com/gx-org/gx/interp/evaluator"
	"github.com/gx-org/gx/interp/fun"
	"github.com/gx-org/gx/interp/materialise"
	"github.com/gx-org/gx/interp/procoptions"
)

// Interpreter runs GX code given an evaluator and package options.
type Interpreter struct {
	eval fun.Evaluator
	core *context.Core

	options *procoptions.Options
}

// New returns a new interpreter.
func New(eval fun.Evaluator, options []options.PackageOption) (*Interpreter, error) {
	itp := &Interpreter{eval: eval}
	var err error
	if itp.options, err = procoptions.New(eval, options); err != nil {
		return nil, err
	}
	itp.core, err = context.New(itp, eval.Importer())
	if err != nil {
		return nil, err
	}
	return itp, nil
}

// Core returns the core context.
func (itp *Interpreter) Core() *context.Core {
	return itp.core
}

// NewFunc creates function elements from function IRs.
func (itp *Interpreter) NewFunc(fn ir.Func, recv *fun.Receiver) fun.Func {
	return itp.eval.NewFunc(fn, recv)
}

// FileScope returns an interpreter given the scope of a file from within a package.
type FileScope struct {
	ctx *context.Context
	env *fun.CallEnv
}

func newFileScope(ctx *context.Context, eval fun.Evaluator, file *ir.File) *FileScope {
	fitp := &FileScope{ctx: ctx}
	fitp.env = fun.NewCallEnv(fitp, eval, fitp.ctx)
	return fitp
}

// ForFile returns an interpreter for a file context.
func (itp *Interpreter) ForFile(file *ir.File) (*FileScope, error) {
	ctx, err := itp.core.NewFileContext(file)
	if err != nil {
		return nil, err
	}
	return newFileScope(ctx, itp.eval, file), nil
}

// EvalExpr evaluates an expression for a given context.
func (fitp *FileScope) EvalExpr(expr ir.Expr) (ir.Element, error) {
	return evalExpr(fitp, expr)
}

// Evaluator returns the evaluator used by the interpreter
func (fitp *FileScope) Evaluator() evaluator.Evaluator {
	return fitp.env.FuncEval()
}

// Materialiser returns the materialiser to convert elements into graph nodes.
func (fitp *FileScope) Materialiser() materialise.Materialiser {
	return fitp.Evaluator().ArrayOps().(materialise.Materialiser)
}

// Sub returns a new interpreter with additional values defined in the context.
func (fitp *FileScope) Sub(vals *context.SubMap) *FileScope {
	sub := *fitp
	sub.ctx = fitp.ctx.Sub(vals)
	sub.env = fun.NewCallEnv(fitp, fitp.env.FuncEval(), sub.ctx)
	return &sub
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

// String representation of the receiver.
func (fitp *FileScope) String() string {
	return fitp.env.String()
}
