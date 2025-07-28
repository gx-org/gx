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
	"github.com/gx-org/gx/internal/interp/flatten"
	"github.com/gx-org/gx/interp/elements"
	"github.com/gx-org/gx/interp/evaluator"
	"github.com/gx-org/gx/interp"
	"github.com/gx-org/gx/interp/materialise"
)

type cast struct {
	src    elements.ExprAt
	target ir.Type
	x      Element
	val    *values.HostArray
}

var (
	_ materialise.ElementMaterialiser = (*cast)(nil)
	_ interp.WithAxes                 = (*cast)(nil)
	_ interp.Copier                   = (*cast)(nil)
	_ elements.ElementWithConstant    = (*cast)(nil)
	_ fmt.Stringer                    = (*cast)(nil)
)

func newCast(ctx ir.Evaluator, expr ir.AssignableExpr, xEl Element, target ir.Type) (*cast, error) {
	opEl := &cast{
		src:    elements.NewNodeAt(ctx.File(), expr),
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

func newReshape(ctx ir.Evaluator, expr ir.AssignableExpr, xEl Element, axisLengths []evaluator.NumericalElement) (Element, error) {
	x := elements.ConstantFromElement(xEl)
	if x == nil {
		return xEl, nil
	}
	val, err := values.NewHostArray(expr.Type(), x.Buffer())
	if err != nil {
		return nil, err
	}
	return &cast{
		src:    elements.NewExprAt(ctx.File(), expr),
		target: expr.Type(),
		x:      xEl,
		val:    val,
	}, nil
}

func (a *cast) UnaryOp(ctx ir.Evaluator, expr *ir.UnaryExpr) (evaluator.NumericalElement, error) {
	return newUnary(ctx, expr, a)
}

func (a *cast) BinaryOp(ctx ir.Evaluator, expr *ir.BinaryExpr, x, y evaluator.NumericalElement) (evaluator.NumericalElement, error) {
	return newBinary(ctx, expr, x, y)
}

func (a *cast) Cast(ctx ir.Evaluator, expr ir.AssignableExpr, target ir.Type) (evaluator.NumericalElement, error) {
	return newCast(ctx, expr, a, target)
}

func (a *cast) Reshape(ctx ir.Evaluator, expr ir.AssignableExpr, axisLengths []evaluator.NumericalElement) (evaluator.NumericalElement, error) {
	return newReshape(ctx, expr, a, axisLengths)
}

// Shape of the value represented by the element.
func (a *cast) Shape() *shape.Shape {
	return a.val.Shape()
}

// Type of the element.
func (a *cast) Type() ir.Type {
	return a.src.Node().Type()
}

// Unflatten creates a GX value from the next handles available in the parser.
func (a *cast) Unflatten(handles *flatten.Parser) (values.Value, error) {
	return handles.ParseArray(a.src.Node().Type())
}

// NumericalConstant returns the value of a constant represented by a node.
func (a *cast) NumericalConstant() *values.HostArray {
	return a.val
}

// Copy the element by returning itself.
func (a *cast) Copy() interp.Copier {
	return a
}

// Materialise returns the element with all its values from the graph.
func (a *cast) Materialise(ao materialise.Materialiser) (materialise.Node, error) {
	return ao.NodeFromArray(a.src.File(), a.src.Node(), a.val)
}

// Axes of the result of the cast.
func (a *cast) Axes(ev ir.Evaluator) (*interp.Slice, error) {
	return axesFromType(ev, a.target)
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
