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
	"go/ast"

	"github.com/gx-org/backend/ops"
	"github.com/gx-org/backend/shape"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/base/cast"
	"github.com/gx-org/gx/internal/interp/canonical"
	"github.com/gx-org/gx/internal/interp/coreops"
	"github.com/gx-org/gx/interp/elements"
	"github.com/gx-org/gx/interp/engine"
	"github.com/gx-org/gx/interp/materialise"
)

// array element storing a GX value array.
type array struct {
	expr  ir.Expr
	typ   ir.ArrayType
	shape *shape.Shape
}

var (
	_ materialise.Node  = (*array)(nil)
	_ elements.WithAxes = (*array)(nil)
	_ coreops.Element   = (*array)(nil)
	_ elements.Slicer   = (*array)(nil)
	_ engine.Copier     = (*array)(nil)
	_ ir.WithLength     = (*array)(nil)
	_ ir.WithExpr       = (*array)(nil)
)

// NewArray returns a new array from a code position and a type.
func NewArray(expr ir.Expr) (engine.NumericalElement, error) {
	typ := expr.Type()
	if typeVal, isTypeVal := expr.(*ir.TypeValExpr); isTypeVal {
		typ = typeVal.Val()
	}
	arrayType, err := cast.To[ir.ArrayType](typ)
	if err != nil {
		return nil, err
	}
	shape := &shape.Shape{
		DType: arrayType.DataType().Kind().DType(),
	}
	if !arrayType.Rank().IsAtomic() {
		shape.AxisLengths = make([]int, len(arrayType.Rank().Axes()))
	}
	return &array{expr: expr, typ: arrayType, shape: shape}, nil
}

func (a *array) Type() ir.Type {
	return a.typ
}

func (a *array) Axes(fetcher ir.Evaluator) (*elements.Slice, error) {
	return coreops.AxesFromType(fetcher, a.typ)
}

func (a *array) UnaryOp(env engine.Env, expr *ir.UnaryExpr) (engine.NumericalElement, error) {
	return NewArray(expr)
}

func (a *array) BinaryOp(env engine.Env, expr *ir.BinaryExpr, x, y engine.NumericalElement) (engine.NumericalElement, error) {
	return NewArray(expr)
}

func (a *array) Cast(env engine.Env, expr ir.Expr, target ir.Type) (engine.NumericalElement, error) {
	return NewArray(expr)
}

func (a *array) Reshape(env engine.Env, expr ir.Expr, axisLengths []engine.NumericalElement) (engine.NumericalElement, error) {
	return NewArray(expr)
}

func (a *array) SliceAt(expr *ir.IndexExpr, index engine.NumericalElement) (ir.Element, error) {
	return NewArray(expr)
}

func (a *array) Slice(expr *ir.SliceExpr, low, high engine.NumericalElement) (ir.Element, error) {
	return NewArray(expr)
}

func (a *array) Copy() engine.Copier {
	return a
}

func (a *array) Shape() *shape.Shape {
	return a.shape
}

func (a *array) Graph() ops.Graph {
	return nil
}

// Length returns the evaluation of the len built-in.
func (a *array) Length(ev ir.Evaluator) (int, error) {
	return a.Shape().OuterAxisLength(), nil
}

func (a *array) storage() ir.Storage {
	withStore, hasStorage := a.expr.(ir.WithStore)
	if !hasStorage {
		return nil
	}
	return withStore.Store()
}

func (a *array) Compare(x canonical.Comparable) (bool, error) {
	otherT, isArray := x.(*array)
	if !isArray {
		return false, nil
	}
	aStorage := a.storage()
	if aStorage == nil {
		return false, nil
	}
	otherStorage := otherT.storage()
	if otherStorage == nil {
		return false, nil
	}
	return aStorage.Same(otherStorage), nil
}

func (a *array) CanonicalExpr() canonical.Canonical {
	return a
}

func (a *array) OutNode() *ops.OutputNode {
	return &ops.OutputNode{Node: a}
}

func (a *array) ShortString() string {
	return a.SourceString(nil)
}

func (a *array) SourceString(from *ir.File) string {
	return a.typ.ReferString(from)
}

func (a *array) Expr(ir.Evaluator, ast.Expr) ([]ir.Expr, error) {
	return []ir.Expr{a.expr}, nil
}
