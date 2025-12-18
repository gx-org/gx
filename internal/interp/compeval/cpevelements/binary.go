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
	"math/big"

	"github.com/gx-org/backend/shape"
	"github.com/gx-org/gx/api/values"
	"github.com/gx-org/gx/build/fmterr"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/golang/backend/kernels"
	"github.com/gx-org/gx/internal/interp/canonical"
	"github.com/gx-org/gx/internal/interp/flatten"
	"github.com/gx-org/gx/interp/elements"
	"github.com/gx-org/gx/interp/evaluator"
	"github.com/gx-org/gx/interp/materialise"
)

type binary struct {
	canonical canonical.Canonical
	src       elements.NodeFile[*ir.BinaryExpr]
	x, y      Element
	val       *values.HostArray
}

var (
	_ materialise.ElementMaterialiser = (*binary)(nil)
	_ ir.Canonical                    = (*binary)(nil)
	_ canonical.Evaluable             = (*binary)(nil)
	_ elements.ElementWithConstant    = (*binary)(nil)
	_ fmt.Stringer                    = (*binary)(nil)
	_ elements.WithAxes               = (*cast)(nil)
)

// NewBinary returns a binary operation between two elements.
func NewBinary(env evaluator.Env, expr *ir.BinaryExpr, xEl, yEl evaluator.NumericalElement) (_ evaluator.NumericalElement, err error) {
	// If the other element is not a compeval element,
	// we are not in compeval mode, so forward the binary operation to the other element.
	x, xOk := xEl.(Element)
	if !xOk {
		return xEl.BinaryOp(env, expr, xEl, yEl)
	}
	y, yOk := yEl.(Element)
	if !yOk {
		return yEl.BinaryOp(env, expr, xEl, yEl)
	}
	defer func() {
		if err != nil {
			err = fmterr.AtNode(env.File().FileSet(), expr.Src, err)
		}
	}()
	var val *values.HostArray
	cx, err := elements.ConstantFromElement(x)
	if err != nil {
		return nil, err
	}
	cy, err := elements.ConstantFromElement(y)
	if err != nil {
		return nil, err
	}
	if cx != nil && cy != nil {
		// Both operand values are known: compute the constant for this operand.
		val, err = buildBinaryVal(expr, cx, cy)
		if err != nil {
			return nil, err
		}
	}
	el := &binary{
		src: elements.NewNodeAt(env.File(), expr),
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
func (a *binary) UnaryOp(env evaluator.Env, expr *ir.UnaryExpr) (evaluator.NumericalElement, error) {
	return newUnary(env, expr, a)
}

// BinaryOp applies a binary operator to x and y.
func (a *binary) BinaryOp(env evaluator.Env, expr *ir.BinaryExpr, x, y evaluator.NumericalElement) (evaluator.NumericalElement, error) {
	return NewBinary(env, expr, x, y)
}

// Cast an element into a given data type.
func (a *binary) Cast(env evaluator.Env, expr ir.AssignableExpr, target ir.Type) (evaluator.NumericalElement, error) {
	return newCast(env, expr, a, target)
}

// Reshape the element into a new shape.
func (a *binary) Reshape(env evaluator.Env, expr ir.AssignableExpr, axisLengths []evaluator.NumericalElement) (evaluator.NumericalElement, error) {
	return NewReshape(env, expr, a, axisLengths)
}

// Shape of the value represented by the element.
func (a *binary) Shape() *shape.Shape {
	return a.val.Shape()
}

// Axes returns the axes of the value as a slice element.
func (a *binary) Axes(ev ir.Evaluator) (*elements.Slice, error) {
	return axesFromType(ev, a.src.Node().Type())
}

func (a *binary) Float() *big.Float {
	return canonical.ToValue(a.canonical)
}

func (a *binary) Value() ir.AssignableExpr {
	return a.src.Node()
}

// Type of the element.
func (a *binary) Type() ir.Type {
	return a.src.Node().Type()
}

// Unflatten creates a GX value from the next handles available in the parser.
func (a *binary) Unflatten(handles *flatten.Parser) (values.Value, error) {
	return handles.ParseArray(a.src.Node().Type())
}

// NumericalConstant returns the value of a constant represented by a node.
func (a *binary) NumericalConstant() (*values.HostArray, error) {
	return a.val, nil
}

// Materialise returns the element with all its values from the graph.
func (a *binary) Materialise(ao materialise.Materialiser) (materialise.Node, error) {
	return ao.NodeFromArray(a.src.File(), a.src.Node(), a.val)
}

// Compare to another element.
func (a *binary) Compare(other canonical.Comparable) (bool, error) {
	otherT, ok := other.(Element)
	if !ok {
		return false, nil
	}
	eq, err := valEqual(a, otherT)
	if err != nil {
		return false, err
	}
	if eq {
		return true, nil
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
	x := canonical.ToString(a.x)
	y := canonical.ToString(a.y)
	return fmt.Sprintf("%v%v%v", x, a.src.Node().Src.Op, y)
}
