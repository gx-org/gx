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
	"fmt"

	"github.com/gx-org/backend/shape"
	"github.com/gx-org/gx/api/values"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/golang/backend/kernels"
	"github.com/gx-org/gx/internal/interp/canonical"
	"github.com/gx-org/gx/interp/elements"
)

type cast struct {
	src    elements.ValueAt
	target ir.Type
	x      Element
	val    *values.HostArray
}

var (
	_ elements.Materialiser        = (*cast)(nil)
	_ elements.ElementWithConstant = (*cast)(nil)
	_ fmt.Stringer                 = (*cast)(nil)
)

func newCast(ctx elements.FileContext, expr ir.AssignableExpr, xEl Element, target ir.Type) (*cast, error) {
	opEl := &cast{
		src:    elements.NewNodeAt[ir.Value](ctx.File(), expr),
		target: target,
		x:      xEl,
	}
	x := elements.ConstantFromElement(xEl)
	if x == nil {
		return opEl, nil
	}
	kx, kxRelease, err := toKernelArray(x)
	defer kxRelease()
	if err != nil {
		return nil, err
	}
	// Convert the interpreter element a.x into a GX value.
	// Use the factory to get the kernel matching the cast operator.
	op, _, _, err := kx.Factory().Cast(target.Kind().DType(), x.Shape().AxisLengths)
	if err != nil {
		return nil, err
	}
	// Apply the kernel.
	res, err := op(kx)
	if err != nil {
		return nil, err
	}
	// Return the result as a GX value.
	val, err := values.NewHostArray(expr.Type(), kernels.NewBuffer(res))
	if err != nil {
		return nil, err
	}
	opEl.val = val
	return opEl, nil

}

func newReshape(ctx elements.FileContext, expr ir.AssignableExpr, xEl Element, axisLengths []elements.NumericalElement) (Element, error) {
	x := elements.ConstantFromElement(xEl)
	if x == nil {
		return xEl, nil
	}
	val, err := values.NewHostArray(expr.Type(), x.Buffer())
	if err != nil {
		return nil, err
	}
	return &cast{
		src:    elements.NewValueAt(ctx.File(), expr),
		target: expr.Type(),
		x:      xEl,
		val:    val,
	}, nil
}

func (a *cast) UnaryOp(ctx elements.FileContext, expr *ir.UnaryExpr) (elements.NumericalElement, error) {
	return newUnary(ctx, expr, a)
}

func (a *cast) BinaryOp(ctx elements.FileContext, expr *ir.BinaryExpr, x, y elements.NumericalElement) (elements.NumericalElement, error) {
	return newBinary(ctx, expr, x, y)
}

func (a *cast) Cast(ctx elements.FileContext, expr ir.AssignableExpr, target ir.Type) (elements.NumericalElement, error) {
	return newCast(ctx, expr, a, target)
}

func (a *cast) Reshape(ctx elements.FileContext, expr ir.AssignableExpr, axisLengths []elements.NumericalElement) (elements.NumericalElement, error) {
	return newReshape(ctx, expr, a, axisLengths)
}

// Shape of the value represented by the element.
func (a *cast) Shape() *shape.Shape {
	return a.val.Shape()
}

func (a *cast) Flatten() ([]elements.Element, error) {
	return []elements.Element{a}, nil
}

// Kind of the element.
func (a *cast) Kind() ir.Kind {
	return a.src.Node().Type().Kind()
}

// Unflatten creates a GX value from the next handles available in the Unflattener.
func (a *cast) Unflatten(handles *elements.Unflattener) (values.Value, error) {
	return a.val, nil
}

// NumericalConstant returns the value of a constant represented by a node.
func (a *cast) NumericalConstant() *values.HostArray {
	return a.val
}

// Materialise returns the element with all its values from the graph.
func (a *cast) Materialise(ao elements.ArrayOps) (elements.Node, error) {
	return ao.ElementFromArray(a.src.ToExprAt(), a.val)
}

// Axes of the result of the cast.
func (a *cast) Axes(fetcher ir.Fetcher) (*elements.Slice, error) {
	return sliceElementFromIRType(fetcher, a.src.ExprSrc(), a.target)
}

// Compare to another element.
func (a *cast) Compare(x canonical.Comparable) bool {
	if valEqual(a, x.(Element)) {
		return true
	}
	other, ok := x.(*cast)
	if !ok {
		return false
	}
	if a.target != other.target {
		return false
	}
	return a.x.Compare(other.x)
}

func (a *cast) CanonicalExpr() canonical.Canonical {
	return a
}

func (a *cast) String() string {
	return fmt.Sprintf("%v(%v)", a.target, fmt.Sprint(a.x))
}
