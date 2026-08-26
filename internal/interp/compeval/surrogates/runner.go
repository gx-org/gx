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

package surrogates

import (
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/interp/context"
	"github.com/gx-org/gx/interp/engine"
	"github.com/gx-org/gx/interp/fun"
)

type runner struct{}

var rn = runner{}

// Runner returns a function runner only building runtime values for results.
// No function is being executed.
func Runner() fun.Runners {
	return rn
}

// FuncDecl runs a function implemented in GX.
func (runner) FuncDecl(fDecl *ir.FuncDecl, env *fun.CallEnv, call *ir.FuncCallExpr, recv engine.Copier, args []ir.Element) ([]ir.Element, error) {
	return Call(call)
}

// FuncLit runs a function literal.
func (runner) FuncLit(lit *ir.FuncLit, env *fun.CallEnv, ctx *context.Context, call *ir.FuncCallExpr, args []ir.Element) ([]ir.Element, error) {
	return Call(call)
}

// Builtin runs a function builtin in GX or provided by a backend.
func (runner) Builtin(fn ir.Func, impl ir.FuncImpl, env *fun.CallEnv, call *ir.FuncCallExpr, recv engine.Copier, args []ir.Element) ([]ir.Element, error) {
	return Call(call)
}
