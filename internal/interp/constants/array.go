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

package constants

import (
	"go/ast"

	"github.com/gx-org/backend/ops"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/build/ir/irkind"
	"github.com/gx-org/gx/internal/interp/coreiface"
	"github.com/gx-org/gx/interp/engine"
)

type array struct {
	lit       *ir.ArrayLitExpr
	vals      []engine.AtomConstant
	axLengths []engine.NumericalElement
}

// NewArray returns a new constant array.
func NewArray(lit *ir.ArrayLitExpr, vals []engine.AtomConstant, axLengths []engine.NumericalElement) engine.Constant {
	return &array{
		lit:       lit,
		vals:      vals,
		axLengths: axLengths,
	}
}

func (n *array) Src() ast.Expr {
	return n.lit.Expr()
}

func (n *array) BuildNode(g ops.Graph, kind irkind.Kind) (ops.Node, error) {
	axes, err := n.AxisLengths()
	if err != nil {
		return nil, err
	}
	total := 1
	for _, ax := range axes {
		total *= ax
	}
	cvr, err := NewConverter(kind)
	if err != nil {
		return nil, err
	}
	vals, err := cvr.convertSlice(total, n.vals)
	if err != nil {
		return nil, err
	}
	return g.Core().NewArrayLiteral(vals, axes...)
}

func (n *array) Type() ir.Type {
	return n.lit.Typ
}

func (n *array) Number() engine.ScalarNumber {
	return nil
}

func (n *array) Expr(ev ir.Evaluator, src ast.Expr) ([]ir.Expr, error) {
	return []ir.Expr{n.lit}, nil
}

// Shape of the array.
func (n *array) AxisLengths() ([]int, error) {
	return coreiface.MapSlice(func(el ir.Element) (int, error) {
		return Convert[int](CInt, el)
	}, n.axLengths)
}

func (n *array) ShortString(from *ir.File) string {
	return n.SourceString(from)
}

func (n *array) SourceString(from *ir.File) string {
	return n.lit.SourceString(from)
}

// ArrayElements returns the elements of a constant array.
func ArrayElements(el engine.Constant) ([]engine.AtomConstant, bool) {
	cst, isCst := el.(*array)
	if !isCst {
		return nil, false
	}
	return cst.vals, true
}
