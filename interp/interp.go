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
	"github.com/gx-org/gx/interp/elements"
	"github.com/gx-org/gx/interp/evaluator"
)

// FuncBuiltin defines a builtin function provided by a backend.
type FuncBuiltin = context.FuncBuiltin

type (
	// intern is an internal interpreter for the context.
	intern struct {
		itp *Interpreter
	}

	// Interpreter runs GX code given an evaluator and package options.
	Interpreter struct {
		core *context.Core
	}
)

// EvalExpr evaluates an expression for a given context.
func (itn *intern) EvalExpr(ctx *context.Context, expr ir.Expr) (ir.Element, error) {
	return evalExpr(ctx, expr)
}

func (itn *intern) EvalStmt(ctx *context.Context, block *ir.BlockStmt) ([]ir.Element, bool, error) {
	return evalBlockStmt(ctx, block)
}

// New returns a new interpreter.
func New(eval context.Evaluator, options []options.PackageOption) (*Interpreter, error) {
	itp := &Interpreter{}
	var err error
	itp.core, err = context.New(&intern{itp: itp}, eval, options)
	if err != nil {
		return nil, err
	}
	return itp, nil
}

// Core returns the core context.
func (itp *Interpreter) Core() *context.Core {
	return itp.core
}

// EvalFunc evaluates a function.
func (itp *Interpreter) EvalFunc(fn *ir.FuncDecl, in *elements.InputElements) (outs []ir.Element, err error) {
	return context.EvalFunc(itp.core, fn, in)
}

// FileScope returns an interpreter given the scope of a file from within a package.
type FileScope struct {
	itp       *Interpreter
	initScope *ir.File

	ctx *context.Context
}

var _ evaluator.Context = (*FileScope)(nil)

// ForFile returns an interpreter for a file context.
func (itp *Interpreter) ForFile(file *ir.File) (*FileScope, error) {
	ctx, err := itp.core.NewFileContext(file)
	if err != nil {
		return nil, err
	}
	return itp.ForContext(ctx), nil
}

// ForContext returns a new file interpreter given a context.
func (itp *Interpreter) ForContext(ctx *context.Context) *FileScope {
	return &FileScope{itp: itp, ctx: ctx}
}

// EvalExpr evaluates an expression for a given context.
func (fitp *FileScope) EvalExpr(expr ir.Expr) (ir.Element, error) {
	return evalExpr(fitp.ctx, expr)
}

// InitScope returns the initial file scope (as opposed to the file of the current context).
func (fitp *FileScope) InitScope() *ir.File {
	return fitp.initScope
}

// Evaluator returns the evaluator used by the interpreter
func (fitp *FileScope) Evaluator() evaluator.Evaluator {
	return fitp.ctx.Evaluator()
}

// Sub returns a new interpreter with additional values defined in the context.
func (fitp *FileScope) Sub(vals map[string]ir.Element) *FileScope {
	sub := *fitp
	sub.ctx = fitp.ctx.Sub(vals)
	return &sub
}

// Context used by the interpreter.
func (fitp *FileScope) Context() *context.Context {
	return fitp.ctx
}

// File returns the current file of the current execution.
func (fitp *FileScope) File() *ir.File {
	return fitp.ctx.File()
}
