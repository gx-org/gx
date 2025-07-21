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
	"github.com/gx-org/gx/interp/materialise"
)

type (
	// Evaluator provides core primitives for the interpreter.
	Evaluator interface {
		evaluator.Evaluator

		// NewFunc creates a new function given its definition and a receiver.
		NewFunc(*Interpreter, ir.PkgFunc, *Receiver) Func

		// NewFuncLit calls a function literal.
		NewFuncLit(fitp *FileScope, ref *ir.FuncLit) (Func, error)
	}
)

type intern struct {
	itp *Interpreter
}

// EvalExpr evaluates an expression block.
func (itn *intern) EvalExpr(ctx *context.Context, expr ir.Expr) (ir.Element, error) {
	fitp := &FileScope{
		itp:       itn.itp,
		initScope: ctx.File(),
		ctx:       ctx,
	}
	return evalExpr(fitp, expr)
}

// EvalStmt evaluates a statement block.
func (itn *intern) EvalStmt(ctx *context.Context, stmt *ir.BlockStmt) ([]ir.Element, bool, error) {
	fitp := &FileScope{
		itp:       itn.itp,
		initScope: ctx.File(),
		ctx:       ctx,
	}
	return evalBlockStmt(fitp, stmt)
}

// Interpreter runs GX code given an evaluator and package options.
type Interpreter struct {
	eval Evaluator
	core *context.Core

	options        []options.PackageOption
	packageOptions map[string][]packageOption
}

// New returns a new interpreter.
func New(eval Evaluator, options []options.PackageOption) (*Interpreter, error) {
	itp := &Interpreter{
		eval:           eval,
		options:        options,
		packageOptions: make(map[string][]packageOption),
	}
	var err error
	itp.core, err = context.New(&intern{itp: itp}, eval.Importer())
	if err != nil {
		return nil, err
	}
	if itp.packageOptions, err = processOptions(options); err != nil {
		return nil, err
	}
	return itp, nil
}

// Core returns the core context.
func (itp *Interpreter) Core() *context.Core {
	return itp.core
}

// NewFunc creates function elements from function IRs.
func (itp *Interpreter) NewFunc(fn ir.PkgFunc, recv *Receiver) Func {
	return itp.eval.NewFunc(itp, fn, recv)
}

// FileScope returns an interpreter given the scope of a file from within a package.
type FileScope struct {
	itp       *Interpreter
	initScope *ir.File

	ctx *context.Context
}

// ForFile returns an interpreter for a file context.
func (itp *Interpreter) ForFile(file *ir.File) (*FileScope, error) {
	fitp := &FileScope{itp: itp, initScope: file}
	var err error
	fitp.ctx, err = itp.core.NewFileContext(file)
	if err != nil {
		return nil, err
	}
	return fitp, nil
}

// EvalExpr evaluates an expression for a given context.
func (fitp *FileScope) EvalExpr(expr ir.Expr) (ir.Element, error) {
	return evalExpr(fitp, expr)
}

// InitScope returns the initial file scope (as opposed to the file of the current context).
func (fitp *FileScope) InitScope() *ir.File {
	return fitp.initScope
}

// Evaluator returns the evaluator used by the interpreter
func (fitp *FileScope) Evaluator() evaluator.Evaluator {
	return fitp.itp.eval
}

// Materialiser returns the materialiser to convert elements into graph nodes.
func (fitp *FileScope) Materialiser() materialise.Materialiser {
	return fitp.itp.eval.ArrayOps().(materialise.Materialiser)
}

// Sub returns a new interpreter with additional values defined in the context.
func (fitp *FileScope) Sub(vals map[string]ir.Element) *FileScope {
	sub := *fitp
	sub.ctx = fitp.ctx.Sub(vals)
	return &sub
}

// NewFunc creates function elements from function IRs.
func (fitp *FileScope) NewFunc(fn ir.PkgFunc, recv *Receiver) Func {
	return fitp.itp.NewFunc(fn, recv)
}

// EvalFunc evaluates a function.
func (fitp *FileScope) EvalFunc(f ir.PkgFunc, call *ir.CallExpr, args []ir.Element) ([]ir.Element, error) {
	fnEl := NewRunFunc(f, nil)
	return fnEl.Call(fitp, call, args)
}

// Context used by the interpreter.
func (fitp *FileScope) Context() *context.Context {
	return fitp.ctx
}

// File returns the current file of the current execution.
func (fitp *FileScope) File() *ir.File {
	return fitp.ctx.File()
}
