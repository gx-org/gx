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
	"github.com/gx-org/gx/build/fmterr"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/golang/backend/kernels"
	"github.com/gx-org/gx/internal/interp/canonical"
	"github.com/gx-org/gx/interp/elements"
)

type binary struct {
	canonical canonical.Canonical
	src       elements.NodeFile[*ir.BinaryExpr]
	x, y      Element
	val       *values.HostArray
}

var (
	_ elements.Materialiser        = (*binary)(nil)
	_ ir.Canonical                 = (*binary)(nil)
	_ elements.ElementWithConstant = (*binary)(nil)
	_ fmt.Stringer                 = (*binary)(nil)
)

func newBinary(ctx ir.Evaluator, expr *ir.BinaryExpr, xEl, yEl elements.NumericalElement) (_ elements.NumericalElement, err error) {
	// If the other element is not a compeval element,
	// we are not in compeval mode, so forward the binary operation to the other element.
	x, xOk := xEl.(Element)
	if !xOk {
		return xEl.BinaryOp(ctx, expr, xEl, yEl)
	}
	y, yOk := yEl.(Element)
	if !yOk {
		return yEl.BinaryOp(ctx, expr, xEl, yEl)
	}
	defer func() {
		if err != nil {
			err = fmterr.Position(ctx.File().FileSet(), expr.Src, err)
		}
	}()
	var val *values.HostArray
	cx := elements.ConstantFromElement(x)
	cy := elements.ConstantFromElement(y)
	if cx != nil && cy != nil {
		// Both operand values are known: compute the constant for this operand.
		val, err = buildBinaryVal(expr, cx, cy)
		if err != nil {
			return nil, err
		}
	}
	el := &binary{
		src: elements.NewNodeAt(ctx.File(), expr),
		x:   x,
		y:   y,
		val: val,
	}
	el.canonical = canonical.FromBinary(expr.Src.Op, x.CanonicalExpr(), y.CanonicalExpr()).Simplify()
	return el, err
}

func buildBinaryVal(expr *ir.BinaryExpr, cx, cy *values.HostArray) (*values.HostArray, error) {
	// Both x and y are host atomic value.
	kx, kxRelease, err := toKernelArray(cx)
	if err != nil {
		return nil, err
	}
	defer kxRelease()
	var ky kernels.Array
	if cx != cy {
		var kyRelease releaseFunc
		ky, kyRelease, err = toKernelArray(cy)
		if err != nil {
			return nil, err
		}
		defer kyRelease()
	} else {
		// Take care to avoid acquiring an array twice (i.e. when LHS and RHS are the same), since that
		// would cause a deadlock.
		ky = kx
	}
	// Convert the interpreter element a.x into a GX value.
	// Use the factory to get the kernel matching the binary operator.
	op, _, err := kx.Factory().BinaryOp(expr.Src.Op, kx.Shape(), ky.Shape())
	if err != nil {
		return nil, err
	}
	// Apply the kernel.
	res, err := op(kx, ky)
	if err != nil {
		return nil, err
	}
	// Return the result as a GX value.
	val, err := values.NewHostArray(expr.Type(), kernels.NewBuffer(res))
	if err != nil {
		return nil, err
	}
	return val, nil
}

// UnaryOp applies a unary operator on x.
func (a *binary) UnaryOp(ctx ir.Evaluator, expr *ir.UnaryExpr) (elements.NumericalElement, error) {
	return newUnary(ctx, expr, a)
}

// BinaryOp applies a binary operator to x and y.
func (a *binary) BinaryOp(ctx ir.Evaluator, expr *ir.BinaryExpr, x, y elements.NumericalElement) (elements.NumericalElement, error) {
	return newBinary(ctx, expr, x, y)
}

// Cast an element into a given data type.
func (a *binary) Cast(ctx ir.Evaluator, expr ir.AssignableExpr, target ir.Type) (elements.NumericalElement, error) {
	return newCast(ctx, expr, a, target)
}

// Reshape the element into a new shape.
func (a *binary) Reshape(ctx ir.Evaluator, expr ir.AssignableExpr, axisLengths []elements.NumericalElement) (elements.NumericalElement, error) {
	return newReshape(ctx, expr, a, axisLengths)
}

// Shape of the value represented by the element.
func (a *binary) Shape() *shape.Shape {
	return a.val.Shape()
}

// Axes returns the axes of the value as a slice element.
func (a *binary) Axes(fetcher ir.Fetcher) (*elements.Slice, error) {
	return sliceElementFromIRType(fetcher, a.src.Node().Type())
}

func (a *binary) Value() ir.AssignableExpr {
	return a.src.Node()
}

func (a *binary) Flatten() ([]elements.Element, error) {
	return []elements.Element{a}, nil
}

// Kind of the element.
func (a *binary) Kind() ir.Kind {
	return a.src.Node().Type().Kind()
}

// Unflatten creates a GX value from the next handles available in the Unflattener.
func (a *binary) Unflatten(handles *elements.Unflattener) (values.Value, error) {
	return handles.ParseArray(a.src.ToExprAt())
}

// NumericalConstant returns the value of a constant represented by a node.
func (a *binary) NumericalConstant() *values.HostArray {
	return a.val
}

// Materialise returns the element with all its values from the graph.
func (a *binary) Materialise(ao elements.ArrayOps) (elements.Node, error) {
	return ao.ElementFromArray(a.src.ToExprAt(), a.val)
}

// Compare to another element.
func (a *binary) Compare(other canonical.Comparable) bool {
	otherT, ok := other.(Element)
	if !ok {
		return false
	}
	if valEqual(a, otherT) {
		return true
	}
	return a.canonical.Compare(otherT.CanonicalExpr())
}

// Canonical representation of the expression.
func (a *binary) CanonicalExpr() canonical.Canonical {
	return a.canonical
}

// Expr returns the IR expression represented by the variable.
func (a *binary) Expr() (ir.AssignableExpr, error) {
	return a.src.Node(), nil
}

func (a *binary) String() string {
	var x string
	switch a.x.(type) {
	case *binary:
		x = fmt.Sprintf("(%v)", a.x)
	default:
		x = fmt.Sprint(a.x)
	}
	var y string
	switch a.y.(type) {
	case *atom, *cast, *variable:
		y = fmt.Sprint(a.y)
	default:
		y = fmt.Sprintf("(%v)", a.y)
	}
	return fmt.Sprintf("%v%v%v", x, a.src.Node().Src.Op, y)
}
