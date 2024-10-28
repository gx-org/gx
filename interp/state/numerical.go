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
	"github.com/gx-org/gx/build/ir"
)

type (
	// Numerical value in the graph.
	Numerical struct {
		state *State
		nods  []*graph.OutputNode
		value *values.HostArray
	}

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

	// ElementWithValueContext is an element able to return a concrete value from the current context.
	// For example, a value passed as an argument to the function.
	ElementWithValueContext interface {
		NumericalElement

		// NumericalConstant returns the value of a constant represented by a node.
		ValueFromContext(Context) (values.Array, error)
	}
)

var (
	_ backendElement      = (*Numerical)(nil)
	_ Slicer              = (*Numerical)(nil)
	_ handleProcessor     = (*Numerical)(nil)
	_ ElementWithConstant = (*Numerical)(nil)
)

// Numerical returns a new node representing a value.
func (g *State) Numerical(value *values.HostArray) NumericalElement {
	return &Numerical{
		state: g,
		value: value,
	}
}

func (n *Numerical) nodes() ([]*graph.OutputNode, error) {
	if n.nods != nil {
		return n.nods, nil
	}
	op, err := n.state.backendGraph.Core().NewConstant(n.value.Buffer())
	if err != nil {
		return nil, err
	}
	n.nods = []*graph.OutputNode{&graph.OutputNode{
		Node:  op,
		Shape: n.value.Shape(),
	}}
	return n.nods, nil
}

func (n *Numerical) node() (*graph.OutputNode, error) {
	nods, err := n.nodes()
	if err != nil {
		return nil, err
	}
	return nods[0], nil
}

// Slice of the value on the first axis given an index.
func (n *Numerical) Slice(expr ExprAt, i int) (Element, error) {
	x, err := n.node()
	if err != nil {
		return nil, err
	}
	sliceNode, err := n.state.backendGraph.Core().NewSlice(x.Node, i)
	if err != nil {
		return nil, err
	}
	return n.state.ElementFromNode(expr, sliceNode, &shape.Shape{
		DType:       x.Shape.DType,
		AxisLengths: x.Shape.AxisLengths[1:],
	})
}

// State owning the element.
func (n *Numerical) State() *State {
	return n.state
}

// Shape of the value of the element.
func (n *Numerical) Shape() *shape.Shape {
	return n.value.Shape()
}

func (n *Numerical) valueFromHandle(handles *handleParser) (values.Value, error) {
	return values.NewDeviceArray(n.value.Type(), handles.next()), nil
}

// NumericalConstant returns the concrete value of a node.
// For example, a hard-coded value in GX code.
func (n *Numerical) NumericalConstant() *values.HostArray {
	return n.value
}

// ToExpr returns the value as a GX IR expression.
func (n *Numerical) ToExpr() *HostArrayExpr {
	return &HostArrayExpr{
		Typ: n.value.Type(),
		Val: n.NumericalConstant(),
	}
}
