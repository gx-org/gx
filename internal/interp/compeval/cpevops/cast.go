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

	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/interp/engine"
	"github.com/gx-org/gx/src/gx/ast"
)

type arrayCast struct {
	core
	x   engine.Element
	typ ir.Type
}

func newArrayCast(xEl engine.Element, typ ir.Type) engine.Element {
	el := &arrayCast{
		x:   xEl,
		typ: typ,
	}
	el.core = core{el: el}
	return el
}

// NewCast applies a cast operator to an element.
func NewCast(env *engine.Env, expr ir.Expr, xEl engine.Element, target ir.Type) engine.Element {
	return newArrayCast(xEl, expr.Type())
}

// NewReshape returns a reshape elements.
func NewReshape(env *engine.Env, expr ir.Expr, xEl engine.Element, axisLengths []engine.NumericalElement, typ ir.Type) (engine.Element, error) {
	return newArrayCast(xEl, typ), nil
}

// Type of the element.
func (a *arrayCast) Type() ir.Type {
	return a.typ
}

// Expr returns the IR expression represented by the variable.
func (a *arrayCast) Expr(ev ir.Evaluator, src ast.Expr) ([]ir.Expr, error) {
	x, err := ir.ToSingleExpr(ev, src, a.x)
	return []ir.Expr{&ir.CastExpr{
		Typ: a.Type(),
		X:   x,
	}}, err
}

func (a *arrayCast) SourceString(from *ir.File) string {
	return fmt.Sprintf("%s(%s)", a.Type().ReferString(from), a.x.SourceString(from))
}
