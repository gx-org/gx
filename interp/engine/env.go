// Copyright 2026 Google LLC
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

package engine

import (
	"fmt"
	"strings"

	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/interp/context"
)

// NewEval creates a new expression evaluator from an environment.
type NewEval func(*Env) ir.Evaluator

// Env is the environment of a function call.
type Env struct {
	reg   *Registry
	ctx   *context.Context
	newEv NewEval
	eng   Engine
}

// NewEnv returns a new evaluation environment.
func NewEnv(reg *Registry, ctx *context.Context, newEv NewEval, eng Engine) *Env {
	return &Env{reg: reg, ctx: ctx, newEv: newEv, eng: eng}
}

// File returns the current file where the code is being interpreted.
func (env *Env) File() *ir.File {
	return env.ctx.File()
}

// Context returns the context for the current interpreter.
func (env *Env) Context() *context.Context {
	return env.ctx
}

// ExprEval returns the expression evaluator of the environment.
func (env *Env) ExprEval() ir.Evaluator {
	return env.newEv(env)
}

// For returns an environment for a given function nature.
func (env *Env) For(nature ir.FuncNature) *Env {
	return NewEnv(env.reg, env.ctx, env.newEv, env.reg.Engine(nature))
}

// Sub returns an environment with a new registry and engine.
func (env *Env) Sub(reg *Registry, eng Engine) *Env {
	return NewEnv(reg, env.ctx, env.newEv, eng)
}

// Engine returns the current engine used by the interpreter.
func (env *Env) Engine() Engine {
	return env.eng
}

// Registry returns the engine used for evaluations.
func (env *Env) Registry() *Registry {
	return env.reg
}

// Runners used to run functions.
func (env *Env) Runners() Runners {
	return env.reg.Runners
}

func (env *Env) String() string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("Expression evaluator: %T\n", env.newEv))
	ctx := env.ctx
	if ctx == nil {
		return b.String()
	}
	b.WriteString(fmt.Sprintf("Context:\n%s", ctx.String()))
	return b.String()
}
