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

package cpevops

import (
	"fmt"
	"go/ast"

	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/algexpr"
	"github.com/gx-org/gx/internal/interp/compeval/cmp"
	"github.com/gx-org/gx/interp/engine"
)

type unary struct {
	core
	expr *ir.UnaryExpr
	x    Element
}

// NewUnary applies an unary operator to an element.
func NewUnary(env engine.Env, expr *ir.UnaryExpr, xEl Element) (_ Element, err error) {
	el := &unary{
		expr: expr,
		x:    xEl,
	}
	el.core = core{el: el}
	return el, err

}

// Type of the element.
func (a *unary) Type() ir.Type {
	return a.expr.Type()
}

func (a *unary) AlgExpr(eva ir.Evaluator) (cmp.Expr, error) {
	x, err := cmp.ToAlgExpr(eva, a.x)
	if err != nil {
		return nil, err
	}
	return algexpr.NewUnary(a.expr.Src.Op, x)
}

// Expr returns the IR expression represented by the variable.
func (a *unary) Expr(ev ir.Evaluator, src ast.Expr) ([]ir.Expr, error) {
	x, err := ir.ToSingleExpr(ev, src, a.x)
	return []ir.Expr{&ir.UnaryExpr{
		Src: a.expr.Src,
		X:   x,
	}}, err
}

func (a *unary) ShortString(from *ir.File) string {
	return fmt.Sprintf("%v%v", a.expr.Src.Op, a.x.ShortString(from))
}

func (a *unary) SourceString(from *ir.File) string {
	return fmt.Sprintf("%v%v", a.expr.Src.Op, a.x.SourceString(from))
}
