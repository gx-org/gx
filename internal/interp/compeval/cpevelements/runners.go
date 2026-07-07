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

package cpevelements

import (
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/interp/context"
	"github.com/gx-org/gx/interp/engine"
	"github.com/gx-org/gx/interp/fun"
	"github.com/gx-org/gx/interp"
)

type proxyRunner struct{}

var prxRunner = proxyRunner{}

// ProxyRunner returns a function runner only building runtime values for results.
// No function is being executed.
func ProxyRunner() fun.Runners {
	return prxRunner
}

func proxyCall(file *ir.File, ftype *ir.FuncType) ([]ir.Element, error) {
	res := ftype.Results.Fields()
	els := make([]ir.Element, len(res))
	for i, ri := range res {
		var err error
		els[i], err = NewRuntimeValue(file, ir.NewIdent(ri.Type()))
		if err != nil {
			return nil, err
		}
	}
	return els, nil
}

// FuncDecl runs a function implemented in GX.
func (proxyRunner) FuncDecl(fDecl *ir.FuncDecl, env *fun.CallEnv, call *ir.FuncCallExpr, recv engine.Copier, args []ir.Element) ([]ir.Element, error) {
	return proxyCall(env.File(), call.Callee.FuncType())
}

// FuncLit runs a function literal.
func (proxyRunner) FuncLit(lit *ir.FuncLit, env *fun.CallEnv, ctx *context.Context, call *ir.FuncCallExpr, args []ir.Element) ([]ir.Element, error) {
	return proxyCall(env.File(), call.Callee.FuncType())
}

// Builtin runs a function builtin in GX or provided by a backend.
func (proxyRunner) Builtin(fn ir.Func, impl ir.FuncImpl, env *fun.CallEnv, call *ir.FuncCallExpr, recv engine.Copier, args []ir.Element) ([]ir.Element, error) {
	return proxyCall(env.File(), call.Callee.FuncType())
}

type mixedRunner struct{}

var mxdRunner = mixedRunner{}

// MixedRunner returns a function runner running compeval and keyword functions.
// Other (non-compeval) functions are not being executed but use the proxy runner instead.
func MixedRunner() fun.Runners {
	return mxdRunner
}

// FuncDecl runs a function implemented in GX.
func (mixedRunner) FuncDecl(fDecl *ir.FuncDecl, env *fun.CallEnv, call *ir.FuncCallExpr, recv engine.Copier, args []ir.Element) ([]ir.Element, error) {
	if fDecl.FuncType().CompEval {
		return interp.Runners().FuncDecl(fDecl, env, call, recv, args)
	}
	return proxyCall(env.File(), call.Callee.FuncType())
}

// FuncLit runs a function literal.
func (mixedRunner) FuncLit(lit *ir.FuncLit, env *fun.CallEnv, ctx *context.Context, call *ir.FuncCallExpr, args []ir.Element) ([]ir.Element, error) {
	return proxyCall(env.File(), call.Callee.FuncType())
}

// Builtin runs a function builtin in GX or provided by a backend.
func (mixedRunner) Builtin(fn ir.Func, impl ir.FuncImpl, env *fun.CallEnv, call *ir.FuncCallExpr, recv engine.Copier, args []ir.Element) ([]ir.Element, error) {
	_, isKeyword := fn.(*ir.FuncKeyword)
	if isKeyword {
		return interp.Runners().Builtin(fn, impl, env, call, recv, args)
	}
	fType := fn.FuncType()
	if fType != nil && fType.CompEval {
		return interp.Runners().Builtin(fn, impl, env, call, recv, args)
	}
	return proxyCall(env.File(), fType)
}
