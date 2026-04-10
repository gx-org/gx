// Copyright 2025 Google LLC
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

// Package fun provides abstractions and elements for functions.
package fun

import (
	"fmt"
	"go/ast"
	"strings"

	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/concrete"
	"github.com/gx-org/gx/interp/context"
	"github.com/gx-org/gx/interp/engine"
)

type (
	// NewRunFunc is a function to create a function that will be executed by the interpreter.
	NewRunFunc func(fn ir.Func, recv *Receiver) Func

	// CallEnv is the environment of a function call.
	CallEnv struct {
		ctx  *context.Context
		expr ir.Evaluator
		fun  Factory
	}

	// Factory provides core primitives for the interpreter.
	Factory interface {
		// NewFunc creates a new function given its definition and a receiver.
		NewFunc(ir.Func, *Receiver) Func

		// NewFuncLit calls a function literal.
		NewFuncLit(*CallEnv, *ir.FuncLit) (Func, error)

		// NewRunFunc creates a function that will always be run by the interpreter.
		NewRunFunc(ir.Func, *Receiver) Func
	}

	// Func is an element owning a callable function.
	Func interface {
		ir.Element
		Func() ir.Func
		Recv() *Receiver
		Call(env *CallEnv, call *ir.FuncCallExpr, args []ir.Element) ([]ir.Element, error)
	}

	// NewFunc creates function elements from function IRs.
	NewFunc func(ir.Func, *Receiver) Func
)

var _ engine.Env = &CallEnv{}

// NewCallEnv returns a function context.
func NewCallEnv(ctx *context.Context, exprEval ir.Evaluator, fun Factory) *CallEnv {
	return &CallEnv{ctx: ctx, expr: exprEval, fun: fun}
}

// Run a function by the interpreter.
func (env *CallEnv) Run(fn ir.Func, recv *Receiver, call *ir.FuncCallExpr, args []ir.Element) ([]ir.Element, error) {
	return env.fun.NewRunFunc(fn, recv).Call(env, call, args)
}

// File returns the current file where the code is being interpreted.
func (env *CallEnv) File() *ir.File {
	return env.ctx.File()
}

// Context returns the context for the current interpreter.
func (env *CallEnv) Context() *context.Context {
	return env.ctx
}

// ExprEval returns the expression evaluator of the environment.
func (env *CallEnv) ExprEval() ir.Evaluator {
	return env.expr
}

// FuncEval returns the function evaluator of the environment.
func (env *CallEnv) FuncEval() Factory {
	return env.fun
}

// Engine returns the interpreter engine.
func (env *CallEnv) Engine() engine.Engine {
	return env.ctx.Engine()
}

// ToConcrete returns the concrete type given the current context.
func (env *CallEnv) ToConcrete(src ast.Expr, tp ir.Type) (ir.Type, ir.CompEvalError, error) {
	return concrete.Concrete(env.ExprEval(), src, tp)
}

func (env *CallEnv) String() string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("Expression evaluator: %T\n", env.expr))
	b.WriteString(fmt.Sprintf("Function evaluator: %T\n", env.fun))
	b.WriteString(fmt.Sprintf("Context:\n%s", env.ctx.String()))
	return b.String()
}
