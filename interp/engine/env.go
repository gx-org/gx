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

type envContext interface {
	context() *context.Context
	file() *ir.File
}

type evalContext struct {
	ctx *context.Context
}

func (ec evalContext) context() *context.Context {
	return ec.ctx
}

func (ec evalContext) file() *ir.File {
	return ec.ctx.File()
}

type proxyContext struct {
	f *ir.File
}

func (proxyContext) context() *context.Context {
	return nil
}

func (pc proxyContext) file() *ir.File {
	return pc.f
}

// Env is the environment of a function call.
type Env struct {
	ctx  envContext
	expr ir.Evaluator
	eng  Engine
	run  Runners
}

// ProxyEnv returns a proxy implementation of the Env interface.
func ProxyEnv(eng Engine, file *ir.File) *Env {
	return &Env{ctx: proxyContext{f: file}, eng: eng}
}

// NewEnv returns a new evaluation environment.
func NewEnv(ctx *context.Context, exprEval ir.Evaluator, eng Engine, run Runners) *Env {
	return &Env{ctx: evalContext{ctx: ctx}, expr: exprEval, eng: eng, run: run}
}

// WithRunners return a new function context for a given runners.
func (env *Env) WithRunners(run Runners) *Env {
	return NewEnv(env.ctx.context(), env.expr, env.eng, run)
}

// File returns the current file where the code is being interpreted.
func (env *Env) File() *ir.File {
	return env.ctx.file()
}

// Context returns the context for the current interpreter.
func (env *Env) Context() *context.Context {
	return env.ctx.context()
}

// ExprEval returns the expression evaluator of the environment.
func (env *Env) ExprEval() ir.Evaluator {
	return env.expr
}

// Engine returns the engine used for evaluations.
func (env *Env) Engine() Engine {
	return env.eng
}

// Runners used to run functions.
func (env *Env) Runners() Runners {
	return env.run
}

func (env *Env) String() string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("Expression evaluator: %T\n", env.expr))
	ctx := env.ctx.context()
	if ctx == nil {
		return b.String()
	}
	b.WriteString(fmt.Sprintf("Context:\n%s", ctx.String()))
	return b.String()
}
