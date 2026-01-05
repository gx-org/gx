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

// Package materialise defines interfaces and helper functions to transform elements into graph nodes.
package materialise

import (
	"github.com/pkg/errors"
	"github.com/gx-org/backend/ops"
	"github.com/gx-org/backend/shape"
	"github.com/gx-org/gx/api/values"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/interp/flatten"
)

type (

	// Node is an element representing a numerical node in a compute graph.
	Node interface {
		ir.Element
		OutNode() *ops.OutputNode
	}

	// Materialiser materialises an array.
	Materialiser interface {
		// Graph returns the graph to which new nodes are being added.
		Graph() ops.Graph

		// Materialise returns the element with all its values from the graph.
		NodeFromArray(file *ir.File, expr ir.Expr, val values.Array) (Node, error)

		// ElementsFromNodes returns a slice of elements from nodes
		ElementsFromNodes(file *ir.File, expr ir.Expr, nodes ...*ops.OutputNode) ([]ir.Element, error)
	}

	// ElementMaterialiser is an element that can return an instance of itself composed only of elements from the backend ops.
	ElementMaterialiser interface {
		ir.Element

		// Materialise returns the element with all its values from the graph.
		Materialise(Materialiser) (Node, error)
	}
)

// Element converts an element into a graph node.
// We unpack the value of *ops.OutputNode to prevent
// from changing the output of Materialise accidentally.
// Returns an error if the element is not a numerical element.
func Element(am Materialiser, el ir.Element) (ops.Node, *shape.Shape, error) {
	materialiser, ok := el.(ElementMaterialiser)
	if !ok {
		return nil, nil, errors.Errorf("cannot convert %T to a backend node graph: does not implement Materialiser", el)
	}
	outEl, err := materialiser.Materialise(am)
	if err != nil {
		return nil, nil, err
	}
	outNode := outEl.OutNode()
	return outNode.Node, outNode.Shape, nil
}

// AllWithShapes materialises a slice of elements into a slice of output graph nodes.
func AllWithShapes(am Materialiser, els []ir.Element) ([]*ops.OutputNode, error) {
	nodes := make([]*ops.OutputNode, len(els))
	for i, el := range els {
		node, shape, err := Element(am, el)
		if err != nil {
			return nil, err
		}
		nodes[i] = &ops.OutputNode{
			Node:  node,
			Shape: shape,
		}
	}
	return nodes, nil
}

// All materialises a slice of elements into a slice of graph nodes.
func All(am Materialiser, els []ir.Element) ([]ops.Node, error) {
	nodes := make([]ops.Node, len(els))
	for i, el := range els {
		var err error
		nodes[i], _, err = Element(am, el)
		if err != nil {
			return nil, err
		}
	}
	return nodes, nil
}

// Flatten first flatten all the elements, then materialise them all.
func Flatten(mat Materialiser, elts ...ir.Element) ([]ops.Node, []*shape.Shape, error) {
	elts, err := flatten.Flatten(elts...)
	if err != nil {
		return nil, nil, err
	}
	outputs, err := AllWithShapes(mat, elts)
	if err != nil {
		return nil, nil, err
	}
	result := make([]ops.Node, len(outputs))
	shapes := make([]*shape.Shape, len(outputs))
	for i, output := range outputs {
		result[i] = output.Node
		shapes[i] = output.Shape
	}
	return result, shapes, nil
}
