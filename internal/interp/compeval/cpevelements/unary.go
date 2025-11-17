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
	"go/token"

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

type unary struct {
	canonical canonical.Canonical
	src       elements.NodeFile[*ir.UnaryExpr]
	x         Element
	val       *values.HostArray
}

var (
	_ materialise.ElementMaterialiser = (*unary)(nil)
	_ elements.ElementWithConstant    = (*unary)(nil)
	_ ir.Canonical                    = (*unary)(nil)
	_ fmt.Stringer                    = (*unary)(nil)
)

func newUnary(env evaluator.Env, expr *ir.UnaryExpr, xEl Element) (_ *unary, err error) {
	defer func() {
		if err != nil {
			err = fmterr.AtNode(env.File().FileSet(), expr.Src, err)
		}
	}()
	opEl := &unary{
		src: elements.NewNodeAt(env.File(), expr),
		x:   xEl,
	}
	defer func() {
		opEl.canonical = opEl.toCanonical()
	}()
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
	// Use the factory to get the kernel matching the binary operator.
	op, _, err := kx.Factory().UnaryOp(expr.Src.Op, kx.Shape())
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

func (a *unary) Reshape(env evaluator.Env, expr ir.AssignableExpr, axisLengths []evaluator.NumericalElement) (evaluator.NumericalElement, error) {
	return newReshape(env, expr, a, axisLengths)
}

// UnaryOp applies a unary operator on x.
func (a *unary) UnaryOp(env evaluator.Env, expr *ir.UnaryExpr) (evaluator.NumericalElement, error) {
	return newUnary(env, expr, a)
}

// BinaryOp applies a binary operator to x and y.
func (a *unary) BinaryOp(env evaluator.Env, expr *ir.BinaryExpr, x, y evaluator.NumericalElement) (evaluator.NumericalElement, error) {
	return newBinary(env, expr, x, y)
}

// Cast an element into a given data type.
func (a *unary) Cast(env evaluator.Env, expr ir.AssignableExpr, target ir.Type) (evaluator.NumericalElement, error) {
	return newCast(env, expr, a, target)
}

// Shape of the value represented by the element.
func (a *unary) Shape() *shape.Shape {
	if a.val != nil {
		return a.val.Shape()
	}
	return nil
}

// Type of the element.
func (a *unary) Type() ir.Type {
	return a.src.Node().Type()
}

// Unflatten creates a GX value from the next handles available in the parser.
func (a *unary) Unflatten(handles *flatten.Parser) (values.Value, error) {
	return handles.ParseArray(a.src.Node().Type())
}

// NumericalConstant returns the value of a constant represented by a node.
func (a *unary) NumericalConstant() *values.HostArray {
	return a.val
}

// Materialise returns the element with all its values from the graph.
func (a *unary) Materialise(ao materialise.Materialiser) (materialise.Node, error) {
	return ao.NodeFromArray(a.src.File(), a.src.Node(), a.val)
}

func (a *unary) Expr() (ir.AssignableExpr, error) {
	return a.src.Node(), nil
}

// Compare to another element.
func (a *unary) Compare(x canonical.Comparable) bool {
	if valEqual(a, x.(Element)) {
		return true
	}
	other, ok := x.(*unary)
	if !ok {
		return false
	}
	if a.src.Node().Src.Op != other.src.Node().Src.Op {
		return false
	}
	return a.x.Compare(other.x)
}

func (a *unary) CanonicalExpr() canonical.Canonical {
	return a.canonical
}

func (a *unary) toCanonical() canonical.Canonical {
	x := a.x.CanonicalExpr()
	switch a.src.Node().Src.Op {
	case token.ADD:
		return x
	case token.SUB:
		return canonical.NewExpr(token.SUB, x)
	default:
		return a
	}
}

func (a *unary) String() string {
	return fmt.Sprintf("%v%v", a.src.Node().Src.Op, fmt.Sprint(a.x))
}
