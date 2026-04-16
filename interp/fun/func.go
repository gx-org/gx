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
	"strings"

	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/interp/context"
	"github.com/gx-org/gx/interp/engine"
)

type (
	// CallEnv is the environment of a function call.
	CallEnv struct {
		ctx  *context.Context
		expr ir.Evaluator
		eng  engine.Engine
		fun  Factory
		run  Runners
	}

	// Runners provides implementations to run functions.
	Runners interface {
		// FuncDecl runs a function implemented in GX.
		FuncDecl(fDecl *ir.FuncDecl, env *CallEnv, call *ir.FuncCallExpr, recv engine.Copier, args []ir.Element) ([]ir.Element, error)
		// FuncLit runs a function literal.
		FuncLit(lit *ir.FuncLit, env *CallEnv, ctx *context.Context, call *ir.FuncCallExpr, args []ir.Element) ([]ir.Element, error)
		// Builtin runs a function builtin in GX or provided by a backend.
		Builtin(fn ir.Func, impl ir.FuncImpl, env *CallEnv, call *ir.FuncCallExpr, recv engine.Copier, args []ir.Element) ([]ir.Element, error)
	}

	// Factory provides core primitives for the interpreter.
	Factory interface {
		// NewFunc creates a new function given its definition and a receiver.
		NewFunc(ir.Func, *Receiver) Func

		// NewFuncLit calls a function literal.
		NewFuncLit(*ir.FuncLit, *context.Context) Func
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

// NewCallEnv returns a function context.
func NewCallEnv(ctx *context.Context, exprEval ir.Evaluator, eng engine.Engine, fun Factory, run Runners) *CallEnv {
	return &CallEnv{ctx: ctx, expr: exprEval, eng: eng, fun: fun, run: run}
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

// Engine returns the engine used for evaluations.
func (env *CallEnv) Engine() engine.Engine {
	return env.eng
}

// Runners used to run functions.
func (env *CallEnv) Runners() Runners {
	return env.run
}

func (env *CallEnv) String() string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("Expression evaluator: %T\n", env.expr))
	b.WriteString(fmt.Sprintf("Function evaluator: %T\n", env.fun))
	b.WriteString(fmt.Sprintf("Context:\n%s", env.ctx.String()))
	return b.String()
}
