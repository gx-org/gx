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
	"fmt"

	"github.com/gx-org/backend/shape"
	"github.com/gx-org/gx/api/values"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/golang/backend/kernels"
	"github.com/gx-org/gx/internal/concrete"
	"github.com/gx-org/gx/internal/interp/canonical"
	"github.com/gx-org/gx/internal/interp/flatten"
	"github.com/gx-org/gx/interp/elements"
	"github.com/gx-org/gx/interp/engine"
	"github.com/gx-org/gx/interp/materialise"
)

type cast struct {
	canonical.AtomStringImpl
	expr ir.Expr
	typ  ir.Type
	x    Element
	val  *values.HostArray
}

var (
	_ materialise.ElementMaterialiser = (*cast)(nil)
	_ elements.WithAxes               = (*cast)(nil)
	_ engine.Copier                   = (*cast)(nil)
	_ elements.ElementWithConstant    = (*cast)(nil)
	_ ir.StorageElement               = (*cast)(nil)
)

// NewCast applies a cast operator to an element.
func NewCast(env engine.Env, expr ir.Expr, xEl Element, target ir.Type) (engine.NumericalElement, error) {
	x, err := elements.ConstantFromElement(xEl)
	if err != nil {
		return nil, err
	}
	typ, cpErr, err := concrete.Concrete(env.ExprEval(), expr.Expr(), target)
	opEl := &cast{
		expr: expr,
		typ:  typ,
		x:    xEl,
	}
	if err != nil {
		return opEl, nil
	}
	if cpErr != nil {
		return opEl, cpErr
	}
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

// NewReshape returns a reshape elements.
func NewReshape(env engine.Env, expr ir.Expr, xEl Element, axisLengths []engine.NumericalElement) (Element, error) {
	x, err := elements.ConstantFromElement(xEl)
	if err != nil {
		return xEl, err
	}
	if x == nil {
		return xEl, nil
	}
	typ, cpErr, err := concrete.Concrete(env.ExprEval(), expr.Expr(), expr.Type())
	c := &cast{
		expr: expr,
		typ:  typ,
		x:    xEl,
	}
	if err != nil {
		return c, err
	}
	if cpErr != nil {
		return c, cpErr
	}
	c.val, err = values.NewHostArray(typ, x.Buffer())
	return c, err
}

func (a *cast) UnaryOp(env engine.Env, expr *ir.UnaryExpr) (engine.NumericalElement, error) {
	return NewUnary(env, expr, a)
}

func (a *cast) BinaryOp(env engine.Env, expr *ir.BinaryExpr, x, y engine.NumericalElement) (engine.NumericalElement, error) {
	return NewBinary(env, expr, x, y)
}

func (a *cast) Cast(env engine.Env, expr ir.Expr, target ir.Type) (engine.NumericalElement, error) {
	return NewCast(env, expr, a, target)
}

func (a *cast) Reshape(env engine.Env, expr ir.Expr, axisLengths []engine.NumericalElement) (engine.NumericalElement, error) {
	return NewReshape(env, expr, a, axisLengths)
}

// Shape of the value represented by the element.
func (a *cast) Shape() *shape.Shape {
	return a.val.Shape()
}

// Type of the element.
func (a *cast) Type() ir.Type {
	return a.typ
}

// Unflatten creates a GX value from the next handles available in the parser.
func (a *cast) Unflatten(handles *flatten.Parser) (values.Value, error) {
	return handles.ParseArray(a.typ)
}

// NumericalConstant returns the value of a constant represented by a node.
func (a *cast) NumericalConstant() (*values.HostArray, error) {
	return a.val, nil
}

// Copy the element by returning itself.
func (a *cast) Copy() engine.Copier {
	return a
}

func (a *cast) Store() ir.Storage {
	return a.typ
}

// Materialise returns the element with all its values from the graph.
func (a *cast) Materialise(ao materialise.Materialiser) (materialise.Node, error) {
	return ao.NodeFromArray(a.val)
}

// Axes of the result of the cast.
func (a *cast) Axes(ev ir.Evaluator) (*elements.Slice, error) {
	return AxesFromType(ev, a.typ)
}

// Compare to another element.
func (a *cast) Compare(x canonical.Comparable) (bool, error) {
	eq, err := valEqual(a, x.(Element))
	if err != nil {
		return false, err
	}
	if eq {
		return true, nil
	}
	other, ok := x.(*cast)
	if !ok {
		return false, nil
	}
	if a.typ != other.typ {
		return false, nil
	}
	return a.x.Compare(other.x)
}

func (a *cast) CanonicalExpr() canonical.Canonical {
	return a
}

func (a *cast) ShortString() string {
	return a.SourceString(nil)
}

func (a *cast) SourceString(from *ir.File) string {
	return fmt.Sprintf("%v(%v)", ir.StringerWithFrom(from, a.typ), ir.StringerWithFrom(from, a.x))
}
