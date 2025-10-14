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

package cpevelements

import (
	"github.com/gx-org/backend/ops"
	"github.com/gx-org/backend/shape"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/interp/canonical"
	"github.com/gx-org/gx/interp/elements"
	"github.com/gx-org/gx/interp/evaluator"
	"github.com/gx-org/gx/interp/materialise"
)

// array element storing a GX value array.
type array struct {
	typ   ir.ArrayType
	shape *shape.Shape
}

var (
	_ materialise.Node  = (*array)(nil)
	_ elements.WithAxes = (*array)(nil)
	_ Element           = (*array)(nil)
	_ elements.Slicer   = (*array)(nil)
	_ elements.Copier   = (*array)(nil)
)

// NewArray returns a new array from a code position and a type.
func NewArray(typ ir.ArrayType) evaluator.NumericalElement {
	shape := &shape.Shape{
		DType: typ.DataType().Kind().DType(),
	}
	if !typ.Rank().IsAtomic() {
		shape.AxisLengths = make([]int, len(typ.Rank().Axes()))
	}
	return &array{shape: shape, typ: typ}
}

func (a *array) Type() ir.Type {
	return a.typ
}

func (a *array) Axes(fetcher ir.Evaluator) (*elements.Slice, error) {
	return axesFromType(fetcher, a.typ)
}

func (a *array) UnaryOp(env evaluator.Env, expr *ir.UnaryExpr) (evaluator.NumericalElement, error) {
	return NewArray(expr.Type().(ir.ArrayType)), nil
}

func (a *array) BinaryOp(env evaluator.Env, expr *ir.BinaryExpr, x, y evaluator.NumericalElement) (evaluator.NumericalElement, error) {
	return NewArray(expr.Type().(ir.ArrayType)), nil
}

func (a *array) Cast(env evaluator.Env, expr ir.AssignableExpr, target ir.Type) (evaluator.NumericalElement, error) {
	return NewArray(expr.Type().(ir.ArrayType)), nil
}

func (a *array) Reshape(env evaluator.Env, expr ir.AssignableExpr, axisLengths []evaluator.NumericalElement) (evaluator.NumericalElement, error) {
	return NewArray(expr.Type().(ir.ArrayType)), nil
}

func (a *array) Slice(expr *ir.IndexExpr, index evaluator.NumericalElement) (ir.Element, error) {
	return NewArray(expr.Type().(ir.ArrayType)), nil
}

func (a *array) Copy() elements.Copier {
	return a
}

func (a *array) Shape() *shape.Shape {
	return a.shape
}

func (a *array) Graph() ops.Graph {
	return nil
}

func (a *array) Compare(x canonical.Comparable) bool {
	return false
}

func (a *array) CanonicalExpr() canonical.Canonical {
	return a
}

func (a *array) OutNode() *ops.OutputNode {
	return &ops.OutputNode{Node: a}
}

func (a *array) String() string {
	return a.typ.String()
}
