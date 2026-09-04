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

type arrayIndex struct {
	core
	expr ir.Expr
	x    Element
	idx  Element
}

// NewIndex returns a binary operation between two elements.
func NewIndex(env *engine.Env, expr *ir.IndexExpr, x Element, idx engine.NumericalElement) (_ Element, err error) {
	idxEl, err := cast.To[Element](idx)
	el := &arrayIndex{
		expr: expr,
		x:    x,
		idx:  idxEl,
	}
	el.core = core{el: el}
	return el, err
}

// Type of the element.
func (a *arrayIndex) Type() ir.Type {
	return a.expr.Type()
}

// Expr returns the IR expression represented by the variable.
func (a *arrayIndex) Expr(ev ir.Evaluator, src ast.Expr) ([]ir.Expr, error) {
	x, err := ir.ToSingleExpr(ev, src, a.x)
	return []ir.Expr{&ir.CastExpr{
		Typ: a.expr.Type(),
		X:   x,
	}}, err
}

func (a *arrayIndex) ShortString(from *ir.File) string {
	return fmt.Sprintf("%s[%s]", a.x.ShortString(from), a.idx.ShortString(from))
}

func (a *arrayIndex) SourceString(from *ir.File) string {
	return fmt.Sprintf("%s[%s]", a.x.SourceString(from), a.idx.SourceString(from))
}
