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

	"github.com/gx-org/gx/build/fmterr"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/algexpr"
	"github.com/gx-org/gx/internal/interp/compeval/cmp"
	"github.com/gx-org/gx/interp/engine"
)

type binary struct {
	core
	expr *ir.BinaryExpr
	x, y Element
}

// NewBinary returns a binary operation between two elements.
func NewBinary(env engine.Env, expr *ir.BinaryExpr, x, y Element) (_ Element, err error) {
	el := &binary{
		expr: expr,
		x:    x,
		y:    y,
	}
	el.core = core{el: el}
	return el, err
}

// NewBinaryFrom returns a new binary operator from a generic receiver.
// This function is used when converting a constant into a proxy,
// that is when a binary operator is used between constant and non-constant elements.
func NewBinaryFrom(env engine.Env, expr *ir.BinaryExpr, x Element, y ir.Element) (_ Element, err error) {
	yEl, yOk := y.(Element)
	if !yOk {
		from := env.File()
		return nil, fmterr.InternalAt(from.FileSet(), expr.Src, "operator (%T)%s(%T) not supported for %T in %s", x, expr.Src.Op, y, y, expr.SourceString(from))
	}
	return NewBinary(env, expr, x, yEl)
}

func (a *binary) Value() ir.Expr {
	return a.expr
}

// Type of the element.
func (a *binary) Type() ir.Type {
	return a.expr.Type()
}

func (a *binary) AlgExpr(eva ir.Evaluator) (cmp.Expr, error) {
	x, err := cmp.ToAlgExpr(eva, a.x)
	if err != nil {
		return nil, err
	}
	y, err := cmp.ToAlgExpr(eva, a.y)
	if err != nil {
		return nil, err
	}
	return algexpr.NewBinary(a.expr.Src.Op, a.Type(), x, y)
}

// Expr returns the IR expression represented by the variable.
func (a *binary) Expr(ev ir.Evaluator, src ast.Expr) ([]ir.Expr, error) {
	x, err := ir.ToSingleExpr(ev, src, a.x)
	if err != nil {
		return nil, err
	}
	y, err := ir.ToSingleExpr(ev, src, a.y)
	if err != nil {
		return nil, err
	}
	return []ir.Expr{&ir.BinaryExpr{
		Src: &ast.BinaryExpr{
			Op: a.expr.Src.Op,
			X:  x.Expr(),
			Y:  y.Expr(),
		},
		X:   x,
		Y:   y,
		Typ: a.expr.Typ,
	}}, nil
}

func (a *binary) ShortString(from *ir.File) string {
	x := a.x.ShortString(from)
	y := a.y.ShortString(from)
	return fmt.Sprintf("%v%v%v", x, a.expr.Src.Op, y)
}

func (a *binary) SourceString(from *ir.File) string {
	return a.expr.SourceString(from)
}
