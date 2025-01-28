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
)

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
