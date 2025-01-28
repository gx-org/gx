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
	"github.com/gx-org/backend/graph"
	"github.com/gx-org/backend/shape"
	"github.com/gx-org/gx/api/values"
	"github.com/gx-org/gx/interp/elements"
)

// BackendNode is a state element owning a node in the backend graph.
type BackendNode struct {
	state *State
	nod   *graph.OutputNode
	expr  elements.ExprAt
}

var (
	_ elements.Slicer = (*BackendNode)(nil)
	_ Materialiser    = (*BackendNode)(nil)
)

// hasShape is a node in the graph with a shape inferred by the backend.
type hasShape interface {
	// BackendShape returns the shape inferred by the backend.
	BackendShape() *shape.Shape
}

func checkShape(node *graph.OutputNode) error {
	want := node.Shape
	nodeWithShape, ok := node.Node.(hasShape)
	if !ok {
		return nil
	}
	got := nodeWithShape.BackendShape()
	if got.DType != want.DType {
		return errors.Errorf("backend returned a buffer with a %s data type but GX expects a %s data type", got.DType, want.DType)
	}
	if got.Size() != want.Size() {
		return errors.Errorf("backend returned a buffer with axis lengths %v but GX expects %v", got.AxisLengths, want.AxisLengths)
	}
	return nil
}

// ElementFromTuple converts a backend tuple to a state.Tuple element.
func (s *State) ElementFromTuple(expr elements.ExprAt, tpl graph.Tuple, shps []*shape.Shape) (*elements.Tuple, error) {
	elts := make([]Element, tpl.Size())
	for i := range tpl.Size() {
		node, err := tpl.Element(i)
		if err != nil {
			return nil, err
		}
		elts[i], err = s.ElementFromNode(expr, &graph.OutputNode{
			Node:  node,
			Shape: shps[i],
		})
		if err != nil {
			return nil, err
		}
	}
	return elements.NewTuple(expr.File(), expr.Node(), elts), nil
}

// ElementFromNode returns an element owning a node in the backend graph.
func (s *State) ElementFromNode(expr elements.ExprAt, node *graph.OutputNode) (*BackendNode, error) {
	if err := checkShape(node); err != nil {
		return nil, err
	}
	return &BackendNode{
		state: s,
		expr:  expr,
		nod:   node,
	}, nil
}

// State returns the state owning the element.
func (n *BackendNode) State() *State {
	return n.state
}

// Slice of the value on the first axis given an index.
func (n *BackendNode) Slice(expr elements.ExprAt, i int) (Element, error) {
	sliceNode, err := n.state.backendGraph.Core().NewSlice(n.nod.Node, i)
	if err != nil {
		return nil, err
	}
	return n.state.ElementFromNode(expr, &graph.OutputNode{
		Node: sliceNode,
		Shape: &shape.Shape{
			DType:       n.Shape().DType,
			AxisLengths: n.Shape().AxisLengths[1:],
		},
	})
}

// Flatten returns the element in a slice.
func (n *BackendNode) Flatten() ([]Element, error) {
	return []Element{n}, nil
}

// Unflatten consumes the next handles to return a GX value.
func (n *BackendNode) Unflatten(handles *elements.Unflattener) (values.Value, error) {
	return values.NewDeviceArray(n.expr.Node().Type(), handles.Next()), nil
}

// Shape returns the shape of the element.
func (n *BackendNode) Shape() *shape.Shape {
	return n.nod.Shape
}

// Materialise returns itself.
func (n *BackendNode) Materialise() (*graph.OutputNode, error) {
	return n.nod, nil
}
