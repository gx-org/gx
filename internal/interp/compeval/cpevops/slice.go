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

package cpevops

import (
	"fmt"
	"go/ast"

	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/base/cast"
	"github.com/gx-org/gx/interp/engine"
)

type arraySlice struct {
	core
	expr      ir.Expr
	x         Element
	low, high Element
}

// NewSlice returns a slice operation.
func NewSlice(env *engine.Env, expr *ir.SliceExpr, x Element, low, high engine.NumericalElement) (_ Element, err error) {
	lowEl, err := cast.To[Element](low)
	if err != nil {
		return nil, err
	}
	highEl, err := cast.To[Element](high)
	if err != nil {
		return nil, err
	}
	el := &arraySlice{
		expr: expr,
		x:    x,
		low:  lowEl,
		high: highEl,
	}
	el.core = core{el: el}
	return el, nil
}

// Type of the element.
func (a *arraySlice) Type() ir.Type {
	return a.expr.Type()
}

// Expr returns the IR expression represented by the variable.
func (a *arraySlice) Expr(ev ir.Evaluator, src ast.Expr) ([]ir.Expr, error) {
	x, err := ir.ToSingleExpr(ev, src, a.x)
	return []ir.Expr{&ir.CastExpr{
		Typ: a.expr.Type(),
		X:   x,
	}}, err
}

func (a *arraySlice) ShortString(from *ir.File) string {
	return fmt.Sprintf("%s[%s:%s]", a.x.ShortString(from), a.low.ShortString(from), a.high.ShortString(from))
}

func (a *arraySlice) SourceString(from *ir.File) string {
	return fmt.Sprintf("%s[%s:%s]", a.x.SourceString(from), a.low.SourceString(from), a.high.SourceString(from))
}
