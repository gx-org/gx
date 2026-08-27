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
	"github.com/gx-org/gx/interp/engine"
)

type arrayCast struct {
	core
	expr ir.Expr
	x    Element
}

// NewCast applies a cast operator to an element.
func NewCast(env *engine.Env, expr ir.Expr, xEl Element, target ir.Type) Element {
	el := &arrayCast{
		expr: expr,
		x:    xEl,
	}
	el.core = core{el: el}
	return el
}

// NewReshape returns a reshape elements.
func NewReshape(env *engine.Env, expr ir.Expr, xEl Element, axisLengths []engine.NumericalElement) (Element, error) {
	return NewCast(env, expr, xEl, expr.Type()), nil
}

// Type of the element.
func (a *arrayCast) Type() ir.Type {
	return a.expr.Type()
}

// Expr returns the IR expression represented by the variable.
func (a *arrayCast) Expr(ev ir.Evaluator, src ast.Expr) ([]ir.Expr, error) {
	x, err := ir.ToSingleExpr(ev, src, a.x)
	return []ir.Expr{&ir.CastExpr{
		Typ: a.Type(),
		X:   x,
	}}, err
}

func (a *arrayCast) ShortString(from *ir.File) string {
	return fmt.Sprintf("%s(%s)", a.Type().ReferString(from), a.x.ShortString(from))
}

func (a *arrayCast) SourceString(from *ir.File) string {
	return fmt.Sprintf("%s(%s)", a.Type().ReferString(from), a.x.SourceString(from))
}
