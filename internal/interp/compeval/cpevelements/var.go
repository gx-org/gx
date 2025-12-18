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
	"go/ast"

	"github.com/gx-org/backend/shape"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/interp/canonical"
	"github.com/gx-org/gx/interp/elements"
	"github.com/gx-org/gx/interp/evaluator"
)

type variable struct {
	src  elements.StorageAt
	name string
}

var (
	_ Element           = (*variable)(nil)
	_ ir.StorageElement = (*variable)(nil)
	_ elements.WithAxes = (*variable)(nil)
	_ ir.Canonical      = (*variable)(nil)
	_ elements.Slicer   = (*variable)(nil)
)

// NewVariable returns a new variable element given a GX variable name.
func NewVariable(src elements.StorageAt) ir.Element {
	return newVariable(src)
}

func newVariable(src elements.StorageAt) *variable {
	return &variable{src: src, name: src.Node().NameDef().Name}
}

// UnaryOp applies a unary operator on x.
func (a *variable) UnaryOp(env evaluator.Env, expr *ir.UnaryExpr) (evaluator.NumericalElement, error) {
	return newUnary(env, expr, a)
}

// BinaryOp applies a binary operator to x and y.
func (a *variable) BinaryOp(env evaluator.Env, expr *ir.BinaryExpr, x, y evaluator.NumericalElement) (evaluator.NumericalElement, error) {
	return newBinary(env, expr, x, y)
}

// Cast an element into a given data type.
func (a *variable) Cast(env evaluator.Env, expr ir.AssignableExpr, target ir.Type) (evaluator.NumericalElement, error) {
	return newCast(env, expr, a, target)
}

// Reshape the variable into a different shape.
func (a *variable) Reshape(env evaluator.Env, expr ir.AssignableExpr, axisLengths []evaluator.NumericalElement) (evaluator.NumericalElement, error) {
	return newReshape(env, expr, a, axisLengths)
}

// Shape of the value represented by the element.
func (a *variable) Shape() *shape.Shape {
	return &shape.Shape{}
}

// Store returns the storage represented by this variable.
func (a *variable) Store() ir.Storage {
	return a.src.Node()
}

// Type of the element.
func (a *variable) Type() ir.Type {
	return a.src.Node().Type()
}

// Axes returns the axes of the value as a slice element.
func (a *variable) Axes(ev ir.Evaluator) (*elements.Slice, error) {
	return axesFromType(ev, a.src.Node().Type())
}

// Slice computes a slice from the variable.
func (a *variable) Slice(expr *ir.IndexExpr, index evaluator.NumericalElement) (ir.Element, error) {
	store := &ir.LocalVarStorage{Src: &ast.Ident{}, Typ: expr.Type()}
	return NewRuntimeValue(a.src.File(), store)
}

// Compare to another element.
func (a *variable) Compare(x canonical.Comparable) (bool, error) {
	other, ok := x.(*variable)
	if !ok {
		return false, nil
	}
	return a.name == other.name, nil
}

// Expr returns the IR expression represented by the variable.
func (a *variable) Expr() (ir.AssignableExpr, error) {
	return &ir.ValueRef{
		Src: &ast.Ident{
			Name: a.name,
		},
		Stor: a.src.Node(),
	}, nil
}

func (a *variable) CanonicalExpr() canonical.Canonical {
	return a
}

func (a *variable) String() string {
	return a.name
}
