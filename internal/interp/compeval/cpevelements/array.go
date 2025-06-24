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
	"github.com/pkg/errors"
	"github.com/gx-org/backend/graph"
	"github.com/gx-org/backend/shape"
	"github.com/gx-org/gx/api/values"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/interp/canonical"
	"github.com/gx-org/gx/interp/elements"
)

// array element storing a GX value array.
type array struct {
	src   elements.ExprAt
	typ   ir.ArrayType
	shape *shape.Shape
}

var (
	_ elements.Node   = (*array)(nil)
	_ IRArrayElement  = (*array)(nil)
	_ Element         = (*array)(nil)
	_ elements.Slicer = (*array)(nil)
	_ elements.Copier = (*array)(nil)
)

// NewArray returns a new array from a code position and a type.
func NewArray(src elements.ExprAt, typ ir.ArrayType) elements.NumericalElement {
	shape := &shape.Shape{
		DType: typ.DataType().Kind().DType(),
	}
	if typ.Rank().NumAxes() > 0 {
		shape.AxisLengths = make([]int, typ.Rank().NumAxes())
	}
	return &array{src: src, shape: shape, typ: typ}
}

func (a *array) Flatten() ([]elements.Element, error) {
	return []elements.Element{a}, nil
}

func (a *array) Unflatten(handles *elements.Unflattener) (values.Value, error) {
	return nil, errors.Errorf("not implemented")
}

func (a *array) Kind() ir.Kind {
	return ir.ArrayKind
}

func (a *array) Axes(fetcher ir.Fetcher) (*elements.Slice, error) {
	return sliceElementFromIRType(fetcher, a.src.ExprSrc(), a.typ)
}

func (a *array) UnaryOp(ctx elements.FileContext, expr *ir.UnaryExpr) (elements.NumericalElement, error) {
	return NewArray(elements.NewExprAt(ctx.File(), expr), expr.Type().(ir.ArrayType)), nil
}

func (a *array) BinaryOp(ctx elements.FileContext, expr *ir.BinaryExpr, x, y elements.NumericalElement) (elements.NumericalElement, error) {
	return NewArray(elements.NewExprAt(ctx.File(), expr), expr.Type().(ir.ArrayType)), nil
}

func (a *array) Cast(ctx elements.FileContext, expr ir.AssignableExpr, target ir.Type) (elements.NumericalElement, error) {
	return NewArray(elements.NewExprAt(ctx.File(), expr), expr.Type().(ir.ArrayType)), nil
}

func (a *array) Reshape(ctx elements.FileContext, expr ir.AssignableExpr, axisLengths []elements.NumericalElement) (elements.NumericalElement, error) {
	return NewArray(elements.NewExprAt(ctx.File(), expr), expr.Type().(ir.ArrayType)), nil
}

func (a *array) Slice(ctx elements.FileContext, expr ir.AssignableExpr, index elements.NumericalElement) (elements.Element, error) {
	return NewArray(elements.NewExprAt(ctx.File(), expr), expr.Type().(ir.ArrayType)), nil
}

func (a *array) Copy() elements.Copier {
	return a
}

func (a *array) Shape() *shape.Shape {
	return a.shape
}

func (a *array) Graph() graph.Graph {
	return nil
}

func (a *array) Compare(x canonical.Comparable) bool {
	return false
}

func (a *array) CanonicalExpr() canonical.Canonical {
	return a
}

func (a *array) OutNode() *graph.OutputNode {
	return &graph.OutputNode{Node: a}
}

func (a *array) String() string {
	return a.typ.String()
}
