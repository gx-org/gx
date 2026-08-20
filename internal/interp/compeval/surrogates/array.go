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

package surrogates

import (
	"go/ast"

	"github.com/gx-org/backend/shape"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/algexpr"
	"github.com/gx-org/gx/internal/base/cast"
	"github.com/gx-org/gx/internal/interp/compeval/cmp"
	"github.com/gx-org/gx/internal/interp/compeval/cpevops"
	"github.com/gx-org/gx/internal/interp/compeval/surrogates/storepath"
	"github.com/gx-org/gx/interp/elements"
	"github.com/gx-org/gx/interp/engine"
)

// Array is a surrogate array.
type Array interface {
	cpevops.Element
	Element
}

// array element storing a GX value array.
type array struct {
	path storepath.Path
	typ  ir.ArrayType
}

// Need to support canonical to be able to compare axes using unpack.
var _ cmp.Canonical = (*array)(nil)

// NewArrayFrom returns a new array from a type.
func NewArrayFrom(p storepath.Path, typ ir.Type) (Array, error) {
	arrayType, err := cast.To[ir.ArrayType](typ)
	if err != nil {
		return nil, err
	}
	return NewArray(p, arrayType), nil
}

// NewArray returns a new array from a generic type.
func NewArray(p storepath.Path, typ ir.ArrayType) Array {
	return &array{path: p, typ: typ}
}

func (a *array) Type() ir.Type {
	return a.typ
}

func (a *array) EvalShape() (*shape.Shape, error) {
	if a.typ.Rank().IsAtomic() {
		return &shape.Shape{
			DType: a.typ.DataType().Kind().DType(),
		}, nil
	}
	return nil, nil
}

// Length returns the evaluation of the len built-in.
func (a *array) Length(ev ir.Evaluator) (int, error) {
	sh, err := a.EvalShape()
	if sh == nil || err != nil {
		return -1, err
	}
	return sh.OuterAxisLength(), nil
}

// UnaryOp applies a unary operator on x.
func (a *array) UnaryOp(env engine.Env, expr *ir.UnaryExpr) (engine.NumericalElement, error) {
	return cpevops.NewUnary(env, expr, a)
}

func (a *array) Axes(ev ir.Evaluator) (*elements.Slice, error) {
	return cpevops.AxesFromType(ev, a.typ)
}

// BinaryOp applies a binary operator to x and y.
func (a *array) BinaryOp(env engine.Env, expr *ir.BinaryExpr, y engine.NumericalElement) (engine.NumericalElement, error) {
	return cpevops.NewBinaryFrom(env, expr, a, y)
}

// Cast an element into a given data type.
func (a *array) Cast(env engine.Env, expr ir.Expr, target ir.Type) (engine.NumericalElement, error) {
	return cpevops.NewCast(env, expr, a, target), nil
}

// Reshape an element.
func (a *array) Reshape(env engine.Env, expr ir.Expr, axisLengths []engine.NumericalElement) (engine.NumericalElement, error) {
	return cpevops.NewReshape(env, expr, a, axisLengths)
}

// SliceAt returns an element of the array given an index.
func (a *array) SliceAt(env engine.Env, expr *ir.IndexExpr, index engine.NumericalElement) (ir.Element, error) {
	return cpevops.NewIndex(env, expr, a, index)
}

// Slice the array given a low and high value.
func (a *array) Slice(env engine.Env, expr *ir.SliceExpr, low, high engine.NumericalElement) (ir.Element, error) {
	return cpevops.NewSlice(env, expr, a, low, high)
}

// Expr returns the IR expression represented by the variable.
func (a *array) Expr(ir.Evaluator, ast.Expr) ([]ir.Expr, error) {
	return []ir.Expr{a.path.Expr()}, nil
}

// AlgExpr returns an algebraic expression.
func (a *array) AlgExpr(eva ir.Evaluator) (cmp.Expr, error) {
	return algexpr.NewSymbFrom(a, a.algCmp, a.path.Expr()), nil
}

func (a *array) algCmp(other ir.Element) bool {
	otherT, ok := other.(*array)
	if !ok {
		return false
	}
	return a.path.Same(otherT.path)
}

func (a *array) Store() ir.Storage {
	return a.path.Store()
}

func (a *array) ShortString(from *ir.File) string {
	return a.SourceString(from)
}

func (a *array) SourceString(from *ir.File) string {
	return a.path.Expr().SourceString(from)
}
