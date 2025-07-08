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
	"math/big"

	"github.com/gx-org/backend/dtype"
	"github.com/gx-org/backend/shape"
	"github.com/gx-org/gx/api/values"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/golang/backend/kernels"
	"github.com/gx-org/gx/internal/interp/canonical"
	"github.com/gx-org/gx/internal/interp/flatten"
	"github.com/gx-org/gx/interp/elements"
)

type atom struct {
	src   elements.ExprAt
	val   *values.HostArray
	float *big.Float
}

var (
	_ elements.ElementWithConstant = (*atom)(nil)
	_ elements.Materialiser        = (*atom)(nil)
	_ Element                      = (*atom)(nil)
	_ elements.Copier              = (*atom)(nil)
	_ canonical.Evaluable          = (*atom)(nil)
	_ elements.WithAxes            = (*atom)(nil)
	_ ir.Canonical                 = (*atom)(nil)
)

// NewAtom returns a new atom element given a GX atom value.
func NewAtom(src elements.ExprAt, val *values.HostArray) (Element, error) {
	var float *big.Float
	if dtype.IsAlgebra(val.Shape().DType) {
		var err error
		float, err = val.ToFloatNumber()
		if err != nil {
			return nil, err
		}
	}
	return &atom{src: src, val: val, float: float}, nil
}

// UnaryOp applies a unary operator on x.
func (a *atom) UnaryOp(ctx ir.Evaluator, expr *ir.UnaryExpr) (elements.NumericalElement, error) {
	return newUnary(ctx, expr, a)
}

// Copy the atom.
func (a *atom) Copy() elements.Copier {
	return &atom{
		src:   a.src,
		val:   a.val,
		float: a.float,
	}
}

// BinaryOp applies a binary operator to x and y.
func (a *atom) BinaryOp(ctx ir.Evaluator, expr *ir.BinaryExpr, x, y elements.NumericalElement) (elements.NumericalElement, error) {
	return newBinary(ctx, expr, x, y)
}

// Cast an element into a given data type.
func (a *atom) Cast(ctx ir.Evaluator, expr ir.AssignableExpr, dtype ir.Type) (elements.NumericalElement, error) {
	return newCast(ctx, expr, a, dtype)
}

func (a *atom) Reshape(ctx ir.Evaluator, expr ir.AssignableExpr, axisLengths []elements.NumericalElement) (elements.NumericalElement, error) {
	return newReshape(ctx, expr, a, axisLengths)
}

// Shape of the value represented by the element.
func (a *atom) Shape() *shape.Shape {
	return a.val.Shape()
}

// NumericalConstant returns the value of a constant represented by a node.
func (a *atom) NumericalConstant() *values.HostArray {
	return a.val
}

// Unflatten creates a GX value from the next handles available in the parser.
func (a *atom) Unflatten(handles *flatten.Parser) (values.Value, error) {
	return handles.ParseArray(a.src.Node().Type())
}

// Type of the element.
func (a *atom) Type() ir.Type {
	return a.val.Type()
}

// Materialise the value into a node in the backend graph.
func (a *atom) Materialise(ao elements.ArrayOps) (elements.Node, error) {
	return ao.ElementFromArray(a.src, a.val)
}

// Compare to another element.
func (a *atom) Compare(x canonical.Comparable) bool {
	xEl, ok := x.(ir.Element)
	if !ok {
		return false
	}
	cx := elements.ConstantFromElement(xEl)
	if cx == nil {
		return false
	}
	return equalArray(a.val, cx)
}

func (a *atom) Axes(ir.Evaluator) (*elements.Slice, error) {
	return elements.NewSlice(ir.IntLenSliceType(), nil), nil
}

// Expr returns the IR expression represented by the variable.
func (a *atom) Expr() (ir.AssignableExpr, error) {
	return a.src.Node(), nil
}

func (a *atom) CanonicalExpr() canonical.Canonical {
	return a
}

func (a *atom) Float() *big.Float {
	return a.float
}

func (a *atom) String() string {
	return a.val.String()
}

type releaseFunc func()

func toKernelArray(array *values.HostArray) (kernels.Array, releaseFunc, error) {
	// Convert the GX value into a Go array with a kernel factory.
	data := array.Buffer().Acquire()
	kArray, err := kernels.NewArrayFromRaw(data, array.Shape())
	if err != nil {
		array.Buffer().Release()
		return nil, nil, err
	}
	return kArray, array.Buffer().Release, nil
}

func equalArray(x, y *values.HostArray) bool {
	if !x.Shape().Equal(y.Shape()) {
		return false
	}
	xBuf := x.Buffer()
	yBuf := y.Buffer()
	xData := xBuf.Acquire()
	defer xBuf.Release()
	yData := yBuf.Acquire()
	defer yBuf.Release()
	for i, xi := range xData {
		if yData[i] != xi {
			return false
		}
	}
	return true
}
