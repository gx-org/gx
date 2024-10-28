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
)

type (
	// BackendNodeTuple is a state element owning a node in the backend graph.
	BackendNodeTuple struct {
		state *State
		nods  []graph.Node
	}

	// NodeWithShape is a node in the graph with a shape inferred by the backend.
	NodeWithShape interface {
		graph.Node

		// BackendShape returns the shape inferred by the backend.
		BackendShape() *shape.Shape
	}
)

var _ Element = (*BackendNodeTuple)(nil)

// ElementFromNodeTuple returns an element owning a tuple node in the backend graph.
func (s *State) ElementFromNodeTuple(nods []graph.Node) *BackendNodeTuple {
	return &BackendNodeTuple{state: s, nods: nods}
}

// State returns the state owning the element.
func (n *BackendNodeTuple) State() *State {
	return n.state
}

func (n *BackendNodeTuple) nodes() ([]graph.Node, error) {
	return n.nods, nil
}

// BackendNodeNumerical is a state element owning a node in the backend graph.
type BackendNodeNumerical struct {
	state *State
	nod   *graph.OutputNode
	expr  ExprAt
}

var (
	_ Slicer          = (*BackendNodeNumerical)(nil)
	_ handleProcessor = (*BackendNodeNumerical)(nil)
	_ backendElement  = (*BackendNodeNumerical)(nil)
)

func checkShape(want *shape.Shape, backendNode graph.Node) error {
	nodeWithShape, ok := backendNode.(NodeWithShape)
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

// ElementFromNode returns an element owning a node in the backend graph.
func (s *State) ElementFromNode(expr ExprAt, node graph.Node, sh *shape.Shape) (*BackendNodeNumerical, error) {
	if err := checkShape(sh, node); err != nil {
		return nil, err
	}
	return &BackendNodeNumerical{
		state: s,
		expr:  expr,
		nod: &graph.OutputNode{
			Node:  node,
			Shape: sh,
		},
	}, nil
}

// State returns the state owning the element.
func (n *BackendNodeNumerical) State() *State {
	return n.state
}

// Slice of the value on the first axis given an index.
func (n *BackendNodeNumerical) Slice(expr ExprAt, i int) (Element, error) {
	sliceNode, err := n.state.backendGraph.Core().NewSlice(n.nod.Node, i)
	if err != nil {
		return nil, err
	}
	return n.state.ElementFromNode(expr, sliceNode, &shape.Shape{
		DType:       n.Shape().DType,
		AxisLengths: n.Shape().AxisLengths[1:],
	})
}

func (n *BackendNodeNumerical) nodes() ([]*graph.OutputNode, error) {
	return []*graph.OutputNode{n.nod}, nil
}

func (n *BackendNodeNumerical) valueFromHandle(handles *handleParser) (values.Value, error) {
	return values.NewDeviceArray(n.expr.Type(), handles.next()), nil
}

// Shape returns the shape of the element.
func (n *BackendNodeNumerical) Shape() *shape.Shape {
	return n.nod.Shape
}
