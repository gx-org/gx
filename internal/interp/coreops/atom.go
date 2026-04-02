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

package coreops

import (
	"go/ast"
	"math/big"

	"github.com/gx-org/backend/dtype"
	"github.com/gx-org/backend/shape"
	"github.com/gx-org/gx/api/values"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/interp/canonical"
	"github.com/gx-org/gx/internal/interp/flatten"
	"github.com/gx-org/gx/interp/elements"
	"github.com/gx-org/gx/interp/evaluator"
	"github.com/gx-org/gx/interp/materialise"
)

type atom struct {
	canonical.AtomStringImpl
	val   *values.HostArray
	expr  ir.Expr
	typ   ir.Type
	float *big.Float
}

var (
	_ elements.ElementWithConstant    = (*atom)(nil)
	_ materialise.ElementMaterialiser = (*atom)(nil)
	_ Element                         = (*atom)(nil)
	_ elements.Copier                 = (*atom)(nil)
	_ canonical.Evaluable             = (*atom)(nil)
	_ elements.WithAxes               = (*atom)(nil)
	_ ir.Canonical                    = (*atom)(nil)
)

// NewAtom returns a new atom element given a GX atom value.
func NewAtom(val *values.HostArray, expr ir.Expr, typ ir.Type) (Element, error) {
	var float *big.Float
	var err error
	if dtype.IsAlgebra(val.Shape().DType) {
		float, err = val.ToFloatNumber()
	}
	return &atom{val: val, expr: expr, float: float, typ: typ}, err
}

// UnaryOp applies a unary operator on x.
func (a *atom) UnaryOp(env evaluator.Env, expr *ir.UnaryExpr) (evaluator.NumericalElement, error) {
	return NewUnary(env, expr, a)
}

// Copy the atom.
func (a *atom) Copy() elements.Copier {
	return a
}

// BinaryOp applies a binary operator to x and y.
func (a *atom) BinaryOp(env evaluator.Env, expr *ir.BinaryExpr, x, y evaluator.NumericalElement) (evaluator.NumericalElement, error) {
	return NewBinary(env, expr, x, y)
}

// Cast an element into a given data type.
func (a *atom) Cast(env evaluator.Env, expr ir.Expr, dtype ir.Type) (evaluator.NumericalElement, error) {
	return NewCast(env, expr, a, dtype)
}

func (a *atom) Reshape(env evaluator.Env, expr ir.Expr, axisLengths []evaluator.NumericalElement) (evaluator.NumericalElement, error) {
	return NewReshape(env, expr, a, axisLengths)
}

// Shape of the value represented by the element.
func (a *atom) Shape() *shape.Shape {
	return a.val.Shape()
}

// NumericalConstant returns the value of a constant represented by a node.
func (a *atom) NumericalConstant() (*values.HostArray, error) {
	return a.val, nil
}

// Unflatten creates a GX value from the next handles available in the parser.
func (a *atom) Unflatten(handles *flatten.Parser) (values.Value, error) {
	return handles.ParseArray(a.typ)
}

// Type of the element.
func (a *atom) Type() ir.Type {
	return a.typ
}

// Materialise the value into a node in the backend graph.
func (a *atom) Materialise(ao materialise.Materialiser) (materialise.Node, error) {
	return ao.NodeFromArray(a.val, a.typ)
}

// Compare to another element.
func (a *atom) Compare(x canonical.Comparable) (bool, error) {
	xEl, ok := x.(ir.Element)
	if !ok {
		return false, nil
	}
	cx, err := elements.ConstantFromElement(xEl)
	if err != nil {
		return false, err
	}
	if cx == nil {
		return false, nil
	}
	return EqualArray(a.val, cx), nil
}

func (a *atom) Axes(ir.Evaluator) (*elements.Slice, error) {
	return elements.NewSlice(ir.IntLenSliceType(), nil), nil
}

// Expr returns the IR expression represented by the variable.
func (a *atom) Expr(ir.Evaluator, ast.Expr) (ir.Expr, ir.CompEvalError, error) {
	return a.expr, nil, nil
}

func (a *atom) CanonicalExpr() canonical.Canonical {
	return a
}

func (a *atom) Float() *big.Float {
	return a.float
}

func (a *atom) ShortString() string {
	return a.SourceString(nil)
}

func (a *atom) SourceString(from *ir.File) string {
	return a.val.SourceString(from)
}
