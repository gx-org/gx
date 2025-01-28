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
	"github.com/pkg/errors"
	"github.com/gx-org/backend/dtype"
	"github.com/gx-org/backend/graph"
	"github.com/gx-org/backend/platform"
	"github.com/gx-org/backend/shape"
	"github.com/gx-org/gx/api"
	"github.com/gx-org/gx/api/values"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/golang/backend/kernels"
)

type (
	// NodeFile is a node located in a file.
	NodeFile[T ir.Node] struct {
		file *ir.File
		node T
	}

	// ExprAt is a generic GX expression.
	ExprAt = NodeFile[ir.Expr]

	// CallAt is a function call GX expression.
	CallAt = NodeFile[*ir.CallExpr]

	// FieldAt is a typed field at a given position.
	FieldAt = NodeFile[*ir.Field]
)

// Context is the current evaluation context.
type Context interface {
	// Receiver on which the function call was done.
	// Can be nil.
	Receiver() values.Value

	// Args returns list of arguments passed to the interpreter at call time.
	Args() []values.Value
}

func flattenAll(elts []Element) ([]Element, error) {
	var flat []Element
	for _, elt := range elts {
		subs, err := elt.Flatten()
		if err != nil {
			return nil, err
		}
		flat = append(flat, subs...)
	}
	return flat, nil
}

// NodeFromElement converts an element into a graph node.
// We unpack the value of *graph.OutputNode to prevent
// from changing the output of Materialise accidently.
// Returns an error if the element is not a numerical element.
func NodeFromElement(el Element) (graph.Node, *shape.Shape, error) {
	materialiser, ok := el.(Materialiser)
	if !ok {
		return nil, nil, errors.Errorf("cannot convert %T to a backend node graph: does not implement Materialiser", el)
	}
	out, err := materialiser.Materialise()
	if err != nil {
		return nil, nil, err
	}
	return out.Node, out.Shape, nil
}

// MaterialiseAll materialises a slice of elements into a slice of output graph nodes.
func MaterialiseAll(els []Element) ([]graph.OutputNode, error) {
	nodes := make([]graph.OutputNode, len(els))
	for i, el := range els {
		node, shape, err := NodeFromElement(el)
		if err != nil {
			return nil, err
		}
		nodes[i] = graph.OutputNode{
			Node:  node,
			Shape: shape,
		}
	}
	return nodes, nil
}

// AxesFromElement returns a shape from a state element.
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
	withValue, ok := el.(ElementWithArrayFromContext)
	if !ok {
		return nil, errors.Errorf("state element %T does not support returning a value given a context", el)
	}
	array, err := withValue.ArrayFromContext(ctx)
	if err != nil {
		return nil, err
	}
	return array.ToHostArray(kernels.Allocator())
}

type (
	handleParser struct {
		dev        *api.Device
		ctx        Context
		compOutput []platform.DeviceHandle
		nextPos    int
	}

	handleProcessor interface {
		valueFromHandle(handles *handleParser) (values.Value, error)
	}
)

func newHandleParser(dev *api.Device, ctx Context, handles []platform.DeviceHandle) *handleParser {
	return &handleParser{
		dev:        dev,
		ctx:        ctx,
		compOutput: handles,
	}
}

func (h *handleParser) next() platform.DeviceHandle {
	n := h.compOutput[h.nextPos]
	h.nextPos++
	return n
}

func (h *handleParser) context() Context {
	return h.ctx
}

func (h *handleParser) size() int {
	return len(h.compOutput)
}

func (h *handleParser) device() *api.Device {
	return h.dev
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

func toDevice(dev *api.Device, arr values.Array) (platform.DeviceHandle, error) {
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
