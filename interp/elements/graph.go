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

package elements

import (
	"github.com/gx-org/backend/graph"
	"github.com/gx-org/gx/api/values"
	"github.com/gx-org/gx/build/ir"
)

type (

	// Node is an element representing a numerical node in a compute graph.
	Node interface {
		Element
		OutNode() *graph.OutputNode
	}

	// ArrayOps are the operator implementations for arrays.
	ArrayOps interface {
		// Graph returns the graph to which new nodes are being added.
		Graph() graph.Graph

		// SubGraph returns a new graph builder.
		SubGraph(name string) (ArrayOps, error)

		// Einsum calls an einstein sum on x and y given the expression in ref.
		Einsum(ref NodeFile[*ir.EinsumExpr], x, y NumericalElement) (NumericalElement, error)

		// Reshape an element into a given shape.
		Reshape(expr ExprAt, x NumericalElement, axisLengths []NumericalElement) (NumericalElement, error)

		// BroadcastInDim the data of an array across dimensions.
		BroadcastInDim(expr ExprAt, x NumericalElement, axisLengths []NumericalElement) (NumericalElement, error)

		// Concat concatenates scalars elements into an array with one axis.
		Concat(expr ExprAt, xs []NumericalElement) (NumericalElement, error)

		// Set a slice in an array.
		Set(call NodeFile[*ir.CallExpr], x, updates, index Element) (Element, error)

		// ElementFromArray returns an element from an array GX value.
		ElementFromArray(expr ExprAt, val values.Array) (Node, error)
	}

	// Materialiser is an element that can return an instance of itself composed only of elements from the backend graph.
	Materialiser interface {
		Element

		// Materialise returns the element with all its values from the graph.
		Materialise(ArrayOps) (Node, error)
	}
)
