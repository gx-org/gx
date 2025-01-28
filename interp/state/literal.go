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
	"github.com/gx-org/backend/graph"
	"github.com/gx-org/backend/shape"
	"github.com/gx-org/gx/api/values"
	"github.com/gx-org/gx/interp/elements"
)

// ValueLiteral value literal specified in the code.
type ValueLiteral struct {
	state *State
	expr  elements.ExprAt
	value *values.HostArray
	node  *graph.OutputNode
}

var (
	_ ElementWithConstant = (*ValueLiteral)(nil)
	_ Materialiser        = (*ValueLiteral)(nil)
)

// NewValueLiteral returns a GX value as an element.
func (s *State) NewValueLiteral(expr elements.ExprAt, value *values.HostArray) (*ValueLiteral, error) {
	return &ValueLiteral{
		expr:  expr,
		state: s,
		value: value,
	}, nil
}

// State returns the state owning the element.
func (n *ValueLiteral) State() *State {
	return n.state
}

// Shape of the value represented by the element.
func (n *ValueLiteral) Shape() *shape.Shape {
	return n.value.Shape()
}

// Materialise the value into a node in the backend graph.
func (n *ValueLiteral) Materialise() (*graph.OutputNode, error) {
	if n.node != nil {
		return n.node, nil
	}
	node, err := n.state.backendGraph.Core().NewConstant(n.value.Buffer())
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
func (n *ValueLiteral) Flatten() ([]Element, error) {
	return []Element{n}, nil
}

// NumericalConstant returns the value of a constant represented by a node.
func (n *ValueLiteral) NumericalConstant() *values.HostArray {
	return n.value
}

func (n *ValueLiteral) valueFromHandle(handles *handleParser) (values.Value, error) {
	// TODO(b/388207169): Always transfer the value to device because C++ bindings do not support HostValue.
	return n.NumericalConstant().ToDevice(handles.device())
}

// Slice of the value on the first axis given an index.
func (n *ValueLiteral) Slice(expr elements.ExprAt, i int) (Element, error) {
	x, err := n.Materialise()
	if err != nil {
		return nil, err
	}
	sliceNode, err := n.state.backendGraph.Core().NewSlice(x.Node, i)
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
func (n *ValueLiteral) ToExpr() *HostArrayExpr {
	return &HostArrayExpr{
		Typ: n.value.Type(),
		Val: n.NumericalConstant(),
	}
}
