// Copyright 2024 Google LLC
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

// Package elements provides generic elements, independent of the evaluator, for the interpreter.
package elements

import (
	"go/token"

	"github.com/pkg/errors"
	"github.com/gx-org/backend/dtype"
	"github.com/gx-org/backend/shape"
	"github.com/gx-org/gx/api/values"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/golang/backend/kernels"
)

// CallInputs is the receiver and arguments with which the function was called.
type CallInputs struct {
	// Receiver on which the function call was done.
	// Can be nil.
	Receiver values.Value

	// Args returns list of arguments passed to the interpreter at call time.
	Args []values.Value
}

type (
	// Element in the state.
	Element interface {
		// Flatten the element, that is:
		// returns itself if the element is atomic,
		// returns its components if the element is a composite.
		Flatten() ([]Element, error)

		// Unflatten creates a GX value from the next handles available in the Unflattener.
		Unflatten(handles *Unflattener) (values.Value, error)
	}

	// Copyable is an interface implemented by nodes that need to be copied when passed to a function.
	Copyable interface {
		Copy() Element
	}

	// FieldSelector selects a field given its index.
	FieldSelector interface {
		SelectField(ExprAt, string) (Element, error)
	}

	// Slicer is a state element that can be sliced.
	Slicer interface {
		Slice(ExprAt, int) (Element, error)
	}

	// ArraySlicer is a state element with an array that can be sliced.
	ArraySlicer interface {
		NumericalElement
		Slice(ExprAt, int) (Element, error)
	}

	// MethodSelector selects a method given its index.
	MethodSelector interface {
		SelectMethod(ir.Func) (*Func, error)
	}
)

type (
	// NodeFile is an expression with the file in which it is declared.
	NodeFile[T ir.Node] struct {
		file *ir.File
		node T
	}

	// NodeAt is a generic GX node.
	NodeAt = NodeFile[ir.Node]

	// ExprAt is a generic GX expression.
	ExprAt = NodeFile[ir.Expr]

	// CallAt is a function call GX expression.
	CallAt = NodeFile[*ir.CallExpr]

	// FieldAt is a typed field at a given position.
	FieldAt = NodeFile[*ir.Field]
)

// NewNodeAt returns a new expression at a given position.
func NewNodeAt[T ir.Node](file *ir.File, expr T) NodeFile[T] {
	return NodeFile[T]{file: file, node: expr}
}

// FSet returns the fileset of the expression.
func (ea NodeFile[T]) FSet() *token.FileSet {
	return ea.file.Package.FSet
}

// Node returns the expression.
func (ea NodeFile[T]) Node() T {
	return ea.node
}

// NodeFile returns a general node.
func (ea NodeFile[T]) NodeFile() NodeFile[ir.Node] {
	return NodeFile[ir.Node]{file: ea.file, node: ea.node}
}

// ToExprAt converts a type position into a generic node position.
func (ea NodeFile[T]) ToExprAt() ExprAt {
	node := any(ea.node)
	return NewNodeAt[ir.Expr](ea.file, node.(ir.Expr))
}

// File returns the file in which the expression is declared.
func (ea NodeFile[T]) File() *ir.File {
	return ea.file
}

// AxesFromElement returns a shape from a state element.
// An error is returned if a concrete shape cannot be returned.
func AxesFromElement(el Element) ([]int, error) {
	dimElements, err := el.Flatten()
	if err != nil {
		return nil, err
	}
	dimensions := make([]int, len(dimElements))
	for i, dimElement := range dimElements {
		var err error
		dimScalarI, err := ConstantScalarFromElement[ir.Int](dimElement)
		if err != nil {
			return nil, err
		}
		dimensions[i] = int(dimScalarI)
	}
	return dimensions, nil
}

type (
	// NumericalElement is a node representing a numerical value.
	NumericalElement interface {
		Element

		// Shape of the value represented by the element.
		Shape() *shape.Shape
	}

	// HostArrayExpr is a host array constant represented as a IR expression.
	HostArrayExpr = ir.RuntimeValueExprT[*values.HostArray]

	// ElementWithConstant is an element with a concrete value that is already known.
	ElementWithConstant interface {
		NumericalElement

		// NumericalConstant returns the value of a constant represented by a node.
		NumericalConstant() *values.HostArray

		// ToExpr returns an IR expression representing the value.
		ToExpr() *HostArrayExpr
	}

	// ElementWithArrayFromContext is an element able to return a concrete value from the current context.
	// For example, a value passed as an argument to the function.
	ElementWithArrayFromContext interface {
		NumericalElement

		// ArrayFromContext fetches an array from the argument.
		ArrayFromContext(*CallInputs) (values.Array, error)
	}
)

// ShapeFromElement returns the shape of a numerical element.
func ShapeFromElement(node Element) (*shape.Shape, error) {
	numerical, ok := node.(NumericalElement)
	if !ok {
		return nil, errors.Errorf("cannot cast %T to a numerical element", node)
	}
	return numerical.Shape(), nil
}

// ConstantScalarFromElement returns a scalar on a host given an element.
func ConstantScalarFromElement[T dtype.GoDataType](el Element) (val T, err error) {
	var hostArray *values.HostArray
	hostArray = ConstantFromElement(el)
	if hostArray == nil {
		err = errors.Errorf("state element %T does not store a constant numerical value", el)
		return
	}
	val = values.ToAtom[T](hostArray)
	return
}

// ConstantFromElement returns the host value represented by an element.
// The function returns (nil, nil) if the element does not host a numerical value.
func ConstantFromElement(el Element) *values.HostArray {
	numerical, ok := el.(ElementWithConstant)
	if !ok {
		return nil
	}
	return numerical.NumericalConstant()
}

// HostValueFromContext returns a host value from the function call.
func HostValueFromContext(ci *CallInputs, el Element) (*values.HostArray, error) {
	withValue, ok := el.(ElementWithArrayFromContext)
	if !ok {
		return nil, errors.Errorf("state element %T does not support returning a value given a context", el)
	}
	array, err := withValue.ArrayFromContext(ci)

	if err != nil {
		return nil, err
	}
	return array.ToHostArray(kernels.Allocator())
}
