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

package surrogates

import (
	"go/ast"

	"github.com/pkg/errors"
	"github.com/gx-org/backend/dtypes"
	"github.com/gx-org/backend/shape"
	"github.com/gx-org/gx/build/fmterr"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/build/ir/irkind"
	"github.com/gx-org/gx/internal/algexpr"
	"github.com/gx-org/gx/internal/interp/compeval/cmp"
	"github.com/gx-org/gx/internal/interp/compeval/cpevops"
	"github.com/gx-org/gx/interp/elements"
	"github.com/gx-org/gx/interp/engine"
)

// tuple represents the element of a slice unpacked.
// Tuples are only used in array axes.
// Implements engine.Numerical by considering the elements of the slice to be equivalent to:
//
//	(slice_0 * ... * slice_n)
//
// From an array point of view, tuples are considered atomic.
type tuple struct {
	slice *slice
}

// Need to support canonical to be able to compare axes using unpack.
var _ cmp.Canonical = (*tuple)(nil)

func newTuple(slice *slice) (cpevops.Element, error) {
	var err error
	knd := slice.typ.DType.Val().Kind()
	if knd != irkind.Int {
		err = errors.Errorf("tuple of slice of %s not supported (only support %s", knd, irkind.Int)
	}
	return &tuple{slice: slice}, err
}

func (t *tuple) expr() ir.Expr {
	return &ir.UnpackExpr{
		X:      t.slice.path.Expr(),
		EltTyp: t.slice.typ.DType.Val(),
	}
}

// Expr returns the IR expression represented by the variable.
func (t *tuple) Expr(ir.Evaluator, ast.Expr) ([]ir.Expr, error) {
	return []ir.Expr{t.expr()}, nil
}

var tupleType = &ir.TupleType{}

func (t *tuple) Type() ir.Type {
	return tupleType
}

// UnaryOp applies a unary operator on x.
func (t *tuple) UnaryOp(env engine.Env, expr *ir.UnaryExpr) (engine.NumericalElement, error) {
	return cpevops.NewUnary(env, expr, t)
}

// BinaryOp applies a binary operator to x and y.
// Note that the receiver can be either the left or right argument.
func (t *tuple) BinaryOp(env engine.Env, expr *ir.BinaryExpr, y engine.NumericalElement) (engine.NumericalElement, error) {
	return cpevops.NewBinaryFrom(env, expr, t, y)
}

// Cast an element into a given data type.
func (t *tuple) Cast(env engine.Env, expr ir.Expr, target ir.Type) (engine.NumericalElement, error) {
	return cpevops.NewCast(env, expr, t, target), nil
}

// Reshape an element.
func (t *tuple) Reshape(env engine.Env, expr ir.Expr, axisLengths []engine.NumericalElement) (engine.NumericalElement, error) {
	return cpevops.NewReshape(env, expr, t, axisLengths)
}

// AlgExpr returns an algebraic expression.
func (t *tuple) AlgExpr(eva ir.Evaluator) (cmp.Expr, error) {
	return algexpr.NewSymbFrom(t, t.algCmp, t.expr()), nil
}

func (t *tuple) algCmp(other ir.Element) bool {
	otherT, ok := other.(*tuple)
	if !ok {
		return false
	}
	return t.slice.path.Same(otherT.slice.path)
}

func (*tuple) SliceAt(env engine.Env, expr *ir.IndexExpr, index engine.NumericalElement) (ir.Element, error) {
	return nil, fmterr.InternalAt(env.File().FileSet(), expr.Node(), "cannot index a tuple")
}

func (*tuple) Slice(env engine.Env, expr *ir.SliceExpr, low, high engine.NumericalElement) (ir.Element, error) {
	return nil, fmterr.InternalAt(env.File().FileSet(), expr.Node(), "cannot slice a tuple")
}

var (
	atomAxes, _ = elements.NewSlice(ir.IntSliceType(), nil)
	atomShape   = &shape.Shape{
		DType: dtypes.Int,
	}
)

// Axes of the result of the cast.
func (*tuple) Axes(ev ir.Evaluator) (*elements.Slice, error) {
	return atomAxes, nil
}

func (*tuple) EvalShape() (*shape.Shape, error) {
	return atomShape, nil
}

func (t *tuple) ShortString(from *ir.File) string {
	return t.SourceString(from)
}

func (t *tuple) SourceString(from *ir.File) string {
	return t.expr().SourceString(from)
}
