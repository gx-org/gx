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

package state

import (
	"go/token"
	"reflect"

	"github.com/pkg/errors"
	"github.com/gx-org/backend/dtype"
	"github.com/gx-org/backend/graph"
	"github.com/gx-org/backend/platform"
	"github.com/gx-org/backend/shape"
	"github.com/gx-org/gx/api/values"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/golang/backend/kernels"
)

type (
	// ExprFile is an expression with the file in which it is declared.
	ExprFile[T ir.Expr] struct {
		file *ir.File
		node T
	}

	// ExprAt is a generic GX expression.
	ExprAt = ExprFile[ir.Expr]

	// CallAt is a function call GX expression.
	CallAt = ExprFile[*ir.CallExpr]

	// FieldAt is a typed field at a given position.
	FieldAt = ExprFile[*ir.Field]

	// Context is the current evaluation context.
	Context interface {
		// Receiver on which the function call was done.
		// Can be nil.
		Receiver() values.Value

		// Args returns list of arguments passed to the interpreter at call time.
		Args() []values.Value
	}
)

// NewExprAt returns a new expression at a given position.
func NewExprAt[T ir.Expr](file *ir.File, expr T) ExprFile[T] {
	return ExprFile[T]{file: file, node: expr}
}

// FSet returns the fileset of the expression.
func (ea ExprFile[T]) FSet() *token.FileSet {
	return ea.file.Package.FSet
}

// ExprT returns the expression.
func (ea ExprFile[T]) ExprT() T {
	return ea.node
}

// ToExprAt converts a type position into a generic node position.
func (ea ExprFile[T]) ToExprAt() ExprAt {
	return NewExprAt[ir.Expr](ea.file, ea.node)
}

// Type of the expression.
func (ea ExprFile[T]) Type() ir.Type {
	return ea.node.Type()
}

// File returns the file in which the expression is declared.
func (ea ExprFile[T]) File() *ir.File {
	return ea.file
}

// NodeFromElement converts an element into a graph node.
// Returns an error if the element is not a numerical element.
func NodeFromElement(el Element) (graph.Node, *shape.Shape, error) {
	nodes, err := OutputsFromElement(el)
	if err != nil {
		return nil, nil, err
	}
	if len(nodes) > 1 {
		return nil, nil, errors.Errorf("element %T converted into more than one (%d) graph node", el, len(nodes))
	}
	return nodes[0].Node, nodes[0].Shape, nil
}

// OutputsFromElement returns a generic graph node from a state element.
func OutputsFromElement(el Element) ([]*graph.OutputNode, error) {
	bEl, ok := el.(backendElement)
	if !ok {
		return nil, errors.Errorf("cannot convert element %T to a backend element %s", el, reflect.TypeFor[backendElement]().Name())
	}
	return bEl.nodes()
}

// OutputsFromElements returns a list of backend graph node from a list of element.
func OutputsFromElements(elements []Element) ([]*graph.OutputNode, error) {
	all := []*graph.OutputNode{}
	for _, el := range elements {
		elNodes, err := OutputsFromElement(el)
		if err != nil {
			return nil, err
		}
		all = append(all, elNodes...)
	}
	return all, nil
}

// AxesFromElement returns a shape from a state element.
func AxesFromElement(el Element) ([]int, error) {
	dimElements := el.(*Slice).Elements()
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
func HostValueFromContext(ctx Context, el Element) (*values.HostArray, error) {
	withValue, ok := el.(ElementWithValueContext)
	if !ok {
		return nil, errors.Errorf("state element %T does not support returning a value given a context", el)
	}
	array, err := withValue.ValueFromContext(ctx)
	if err != nil {
		return nil, err
	}
	return array.ToHostArray(kernels.Allocator())
}

type (
	handleParser struct {
		handles []platform.DeviceHandle
		nextPos int
	}

	handleProcessor interface {
		valueFromHandle(handles *handleParser) (values.Value, error)
	}
)

func (h *handleParser) next() platform.DeviceHandle {
	n := h.handles[h.nextPos]
	h.nextPos++
	return n
}

func (h *handleParser) size() int {
	return len(h.handles)
}

func (h *handleParser) parse(el Element) (values.Value, error) {
	toValue, ok := el.(handleProcessor)
	if !ok {
		return nil, errors.Errorf("state element %T cannot convert backend handle to GX value", el)
	}
	val, err := toValue.valueFromHandle(h)
	if err != nil {
		return nil, err
	}
	if val == nil {
		return nil, errors.Errorf("state element %T returned a nil value", el)
	}
	return val, nil
}

type newCompValue func(ir.Type, []values.Value) (values.Value, error)

func (h *handleParser) parseComposite(ncv newCompValue, typ ir.Type, els []Element) (values.Value, error) {
	vals := make([]values.Value, len(els))
	for i, el := range els {
		var err error
		vals[i], err = h.parse(el)
		if err != nil {
			return nil, err
		}
	}
	return ncv(typ, vals)
}

func toDevice(dev platform.Device, arr values.Array) (platform.DeviceHandle, error) {
	deviceArray, err := arr.ToDevice(dev)
	if err != nil {
		return nil, err
	}
	return deviceArray.DeviceHandle(), nil
}

func parseCompositeOf[T values.Value](
	f func(ir.Type, []values.Value) (T, error),
) func(ir.Type, []values.Value) (values.Value, error) {
	return func(typ ir.Type, vals []values.Value) (values.Value, error) {
		return f(typ, vals)
	}
}
