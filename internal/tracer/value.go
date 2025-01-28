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

package tracer

import (
	"github.com/gx-org/backend/graph"
	"github.com/gx-org/backend/shape"
	"github.com/gx-org/gx/api/values"
	"github.com/gx-org/gx/golang/backend/kernels"
	"github.com/gx-org/gx/interp/elements"
	"github.com/gx-org/gx/interp/state"
)

// valueElement is a GX value represented as a node in the graph.
type valueElement struct {
	state *state.State
	expr  elements.NodeAt
	value *values.HostArray
	node  *graph.OutputNode
}

var (
	_ elements.ElementWithConstant = (*valueElement)(nil)
	_ state.Materialiser           = (*valueElement)(nil)
)

func newValueElement(s *state.State, src elements.NodeAt, value values.Array) (*valueElement, error) {
	hostValue, err := value.ToHostArray(kernels.Allocator())
	if err != nil {
		return nil, err
	}
	return &valueElement{
		expr:  src,
		state: s,
		value: hostValue,
	}, nil
}

// Shape of the value represented by the element.
func (n *valueElement) Shape() *shape.Shape {
	return n.value.Shape()
}

// Materialise the value into a node in the backend graph.
func (n *valueElement) Materialise() (*graph.OutputNode, error) {
	if n.node != nil {
		return n.node, nil
	}
	node, err := n.state.BackendGraph().Core().NewConstant(n.value.Buffer())
	if err != nil {
		return nil, err
	}
	n.node = &graph.OutputNode{
		Node:  node,
		Shape: n.value.Shape(),
	}
	return n.node, nil
}

// Flatten returns the element in a slice.
func (n *valueElement) Flatten() ([]elements.Element, error) {
	return []elements.Element{n}, nil
}

// NumericalConstant returns the value of a constant represented by a node.
func (n *valueElement) NumericalConstant() *values.HostArray {
	return n.value
}

// Unflatten creates a GX value from the next handles available in the Unflattener.
func (n *valueElement) Unflatten(handles *elements.Unflattener) (values.Value, error) {
	// TODO(b/388207169): Always transfer the value to device because C++ bindings do not support HostValue.
	return n.NumericalConstant().ToDevice(handles.Device())
}

// Slice of the value on the first axis given an index.
func (n *valueElement) Slice(expr elements.ExprAt, i int) (elements.Element, error) {
	x, err := n.Materialise()
	if err != nil {
		return nil, err
	}
	sliceNode, err := n.state.BackendGraph().Core().NewSlice(x.Node, i)
	if err != nil {
		return nil, err
	}
	sh := n.Shape()
	return n.state.ElementFromNode(expr, &graph.OutputNode{
		Node: sliceNode,
		Shape: &shape.Shape{
			DType:       sh.DType,
			AxisLengths: sh.AxisLengths[1:],
		},
	})
}

// ToExpr returns the value as a GX IR expression.
func (n *valueElement) ToExpr() *elements.HostArrayExpr {
	return &elements.HostArrayExpr{
		Typ: n.value.Type(),
		Val: n.NumericalConstant(),
	}
}
