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
	"github.com/gomlx/compute"
	"github.com/gomlx/compute/shapes"
	"github.com/gx-org/gx/api/hostio"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/base/cast"
	"github.com/gx-org/gx/internal/interp/flatten"
)

type (
	// Node is an element representing a numerical node in a compute graph.
	Node interface {
		ir.Element
		Value() compute.Value
	}

	// Materialiser materialises an array.
	Materialiser interface {
		// Graph returns the graph to which new nodes are being added.
		Graph() compute.Function

		// Materialise returns the element with all its values from the graph.
		NodeFromArray(val *hostio.HostArray) (Node, error)

		// ElementsFromNodes returns a slice of elements from nodes and their types.
		ElementsFromNodes(ev ir.Evaluator, nodes []compute.Value, types []ir.Type) ([]ir.Element, error)
	}

	// ElementMaterialiser is an element that can return an instance of itself composed only of elements from the backend backend.
	ElementMaterialiser interface {
		ir.Element

		// Materialise returns the element with all its values from the graph.
		Materialise(Materialiser) (Node, error)
	}
)

// ElementWithShape materialises an element and fetch its shape.
func ElementWithShape(mat Materialiser, el ir.Element) (compute.Value, shapes.Shape, error) {
	val, err := Element(mat, el)
	if err != nil {
		return nil, shapes.Invalid(), err
	}
	outShape, err := mat.Graph().Builder().OpShape(val)
	if err != nil {
		return nil, shapes.Invalid(), err
	}
	return val, outShape, nil
}

// Element converts an element into a graph node.
// Returns an error if the element is not a numerical element.
func Element(am Materialiser, el ir.Element) (compute.Value, error) {
	materialiser, err := cast.To[ElementMaterialiser](el)
	if err != nil {
		return nil, err
	}
	outEl, err := materialiser.Materialise(am)
	if err != nil {
		return nil, err
	}
	return outEl.Value(), nil
}

// All materialises a slice of elements into a slice of graph nodes.
func All(mat Materialiser, els []ir.Element) ([]compute.Value, error) {
	nodes := make([]compute.Value, len(els))
	for i, el := range els {
		var err error
		nodes[i], err = Element(mat, el)
		if err != nil {
			return nil, err
		}
	}
	return nodes, nil
}

// Shapes of a slice of values.
func Shapes(mat Materialiser, vals []compute.Value) ([]shapes.Shape, error) {
	bld := mat.Graph().Builder()
	shapes := make([]shapes.Shape, len(vals))
	for i, val := range vals {
		var err error
		shapes[i], err = bld.OpShape(val)
		if err != nil {
			return nil, err
		}
	}
	return shapes, nil
}

// AllWithShapes of a slice of values.
func AllWithShapes(mat Materialiser, els []ir.Element) ([]compute.Value, []shapes.Shape, error) {
	vals, err := All(mat, els)
	if err != nil {
		return nil, nil, err
	}
	shapes, err := Shapes(mat, vals)
	if err != nil {
		return nil, nil, err
	}
	return vals, shapes, nil
}

// Flatten first flatten all the elements, then materialise them all.
func Flatten(mat Materialiser, elts ...ir.Element) ([]compute.Value, []shapes.Shape, error) {
	elts, err := flatten.Flatten(elts...)
	if err != nil {
		return nil, nil, err
	}
	vals, err := All(mat, elts)
	if err != nil {
		return nil, nil, err
	}
	shapes, err := Shapes(mat, vals)
	if err != nil {
		return nil, nil, err
	}
	return vals, shapes, nil
}

// ElementFromNode converts a node and its type into an element.
func ElementFromNode(ev ir.Evaluator, mat Materialiser, op compute.Value, typ ir.Type) ([]ir.Element, error) {
	return mat.ElementsFromNodes(ev, []compute.Value{op}, []ir.Type{typ})
}
