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
	"go/ast"
	"reflect"

	"github.com/pkg/errors"
	"github.com/gx-org/backend/shape"
	"github.com/gx-org/gx/api/values"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/interp/canonical"
	"github.com/gx-org/gx/internal/interp/coreops"
	"github.com/gx-org/gx/interp/elements"
	"github.com/gx-org/gx/interp/engine"
	"github.com/gx-org/gx/interp/fun"
)

type storedValue struct {
	storage *proxy
	val     ir.Element
}

var (
	_ coreops.Element              = (*storedValue)(nil)
	_ elements.WithAxes            = (*storedValue)(nil)
	_ ir.WithStore                 = (*storedValue)(nil)
	_ ir.WithLength                = (*storedValue)(nil)
	_ ir.Canonical                 = (*storedValue)(nil)
	_ elements.Slicer              = (*storedValue)(nil)
	_ elements.ElementWithConstant = (*storedValue)(nil)
	_ elements.Selector            = (*storedValue)(nil)
	_ elements.Under               = (*storedValue)(nil)
	_ elements.WithElements        = (*storedValue)(nil)
)

// NewStoredValue returns a new element representing a value stored in a variable.
func NewStoredValue(file *ir.File, storage ir.Storage, value ir.Element) ir.Element {
	return &storedValue{
		storage: newProxy(elements.NewNodeAt[ir.Storage](file, storage)),
		val:     value,
	}
}

// NumericalConstant returns the value of a constant represented by a node.
func (v *storedValue) NumericalConstant() (*values.HostArray, error) {
	return elements.ConstantFromElement(v.val)
}

func (v *storedValue) canonical() (coreops.Element, error) {
	can, ok := v.val.(coreops.Element)
	if !ok {
		return nil, errors.Errorf("%T is not a canonical element", v.val)
	}
	return can, nil
}

// UnaryOp applies a unary operator on x.
func (v *storedValue) UnaryOp(env engine.Env, expr *ir.UnaryExpr) (engine.NumericalElement, error) {
	can, err := v.canonical()
	if err != nil {
		return can, nil
	}
	return coreops.NewUnary(env, expr, can)
}

// BinaryOp applies a binary operator to x and y.
func (v *storedValue) BinaryOp(env engine.Env, expr *ir.BinaryExpr, x, y engine.NumericalElement) (engine.NumericalElement, error) {
	return coreops.NewBinary(env, expr, x, y)
}

// Cast an element into a given data type.
func (v *storedValue) Cast(env engine.Env, expr ir.Expr, target ir.Type) (engine.NumericalElement, error) {
	can, err := v.canonical()
	if err != nil {
		return can, nil
	}
	return coreops.NewCast(env, expr, can, target)
}

// Reshape the variable into a different shape.
func (v *storedValue) Reshape(env engine.Env, expr ir.Expr, axisLengths []engine.NumericalElement) (engine.NumericalElement, error) {
	can, err := v.canonical()
	if err != nil {
		return can, nil
	}
	return coreops.NewReshape(env, expr, can, axisLengths)
}

// Store returns the storage represented by this variable.
func (v *storedValue) Store() ir.Storage {
	return v.storage.Store()
}

func (v *storedValue) Shape() *shape.Shape {
	val, ok := v.val.(elements.FixedShape)
	if !ok {
		return nil
	}
	return val.Shape()
}

// Elements returns the elements of a slice.
func (v *storedValue) Elements() []ir.Element {
	val, ok := v.val.(elements.WithElements)
	if !ok {
		return nil
	}
	return val.Elements()
}

// Value returns the value being stored.
func (v *storedValue) Value() ir.Element {
	if val, ok := v.val.(*storedValue); ok {
		return val.Value()
	}
	return v.val
}

// Type of the element.
func (v *storedValue) Type() ir.Type {
	return v.val.Type()
}

// Axes returns the axes of the value as a slice element.
func (v *storedValue) Axes(ev ir.Evaluator) (*elements.Slice, error) {
	return coreops.AxesFromType(ev, v.Type())
}

// Compare to another element.
func (v *storedValue) Compare(x canonical.Comparable) (bool, error) {
	other, ok := x.(*storedValue)
	if !ok {
		return false, nil
	}
	return v == other, nil
}

// SliceAt computes a slice from the variable.
func (v *storedValue) SliceAt(expr *ir.IndexExpr, index engine.NumericalElement) (ir.Element, error) {
	slicer, ok := v.val.(elements.Slicer)
	if ok {
		return slicer.SliceAt(expr, index)
	}
	return v.storage.SliceAt(expr, index)
}

func (v *storedValue) Slice(expr *ir.SliceExpr, low, high engine.NumericalElement) (ir.Element, error) {
	slicer, ok := v.val.(elements.Slicer)
	if ok {
		return slicer.Slice(expr, low, high)
	}
	return v.storage.Slice(expr, low, high)
}

func (v *storedValue) Length(ev ir.Evaluator) (int, error) {
	withLen, ok := v.val.(ir.WithLength)
	if !ok {
		return 0, errors.Errorf("cannot cast %T to %s", v.val, reflect.TypeFor[ir.WithLength]().Name())
	}
	return withLen.Length(ev)
}

// Expr returns the IR expression represented by the variable.
func (v *storedValue) Expr(ev ir.Evaluator, src ast.Expr) (ir.Expr, ir.CompEvalError, error) {
	valExpr, ok := v.val.(ir.WithExpr)
	if ok {
		return valExpr.Expr(ev, src)
	}
	return v.storage.Expr(ev, src)
}

func (v *storedValue) CanonicalExpr() canonical.Canonical {
	return v.storage
}
func (v *storedValue) Func() ir.Func {
	return v.val.(fun.Func).Func()
}

func (v *storedValue) Recv() *fun.Receiver {
	return v.val.(fun.Func).Recv()
}

func (v *storedValue) Call(env *fun.CallEnv, call *ir.FuncCallExpr, args []ir.Element) ([]ir.Element, error) {
	return v.val.(fun.Func).Call(env, call, args)
}

func (v *storedValue) Select(expr *ir.SelectorExpr) (ir.Element, error) {
	return v.val.(elements.Selector).Select(expr)
}

func (v *storedValue) Under() ir.Element {
	return v.val
}

func (v *storedValue) ShortString() string {
	return fmt.Sprint(v.val)
}

func (v *storedValue) SourceString(from *ir.File) string {
	return fmt.Sprintf("%s -> %T:%s", v.storage.SourceString(from), v.val, v.val.Type().ReferString(from))
}

// StoredValueOf returns the value encapsulated and it has been associated with its storage.
func StoredValueOf(el ir.Element) ir.Element {
	if storedValue, ok := el.(*storedValue); ok {
		return storedValue.Value()
	}
	return el
}
