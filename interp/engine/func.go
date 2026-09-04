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
	"go/ast"

	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/interp/coreiface"
	"github.com/gx-org/gx/interp/context"
)

type (
	// NamedType is a named type.
	NamedType interface {
		coreiface.Under
		Selector
		Copier
	}

	// Receiver of a function.
	Receiver struct {
		Ident   *ast.Ident
		Element NamedType
	}

	// Func is an element owning a callable function.
	Func interface {
		ir.Element
		IR() ir.Func
		Recv() *Receiver
		Call(env *Env, call *ir.FuncCallExpr, args []ir.Element) ([]ir.Element, error)
	}

	// Selector selects a field given its index.
	Selector interface {
		ir.Element
		Select(env *Env, expr *ir.SelectorExpr) (ir.Element, error)
	}

	// Runners provides implementations to run functions.
	Runners interface {
		// FuncDecl runs a function implemented in GX.
		FuncDecl(fDecl *ir.FuncDecl, env *Env, call *ir.FuncCallExpr, recv Copier, args []ir.Element) ([]ir.Element, error)
		// FuncLit runs a function literal.
		FuncLit(lit *ir.FuncLit, env *Env, ctx *context.Context, call *ir.FuncCallExpr, args []ir.Element) ([]ir.Element, error)
		// Builtin runs a function builtin in GX or provided by a backend.
		Builtin(fn ir.Func, impl ir.FuncImpl, env *Env, call *ir.FuncCallExpr, recv Copier, args []ir.Element) ([]ir.Element, error)
	}
)
