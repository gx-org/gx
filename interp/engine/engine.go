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

// Package engine defines interfaces for the interpreter to use to evaluate GX code.
package engine

import (
	"github.com/gomlx/compute"
	"github.com/gx-org/gx/build/ir"
)

type (
	// Engine are the operator implementations for arrays.
	Engine interface {
		// Graph returns the graph to which new nodes are being added.
		Graph() compute.Function

		// SubGraph returns a new graph builder.
		SubGraph(name string) (Engine, error)

		// Einsum calls an einstein sum on x and y given the expression in ref.
		Einsum(ctx ir.Evaluator, expr *ir.EinsumExpr, x, y NumericalElement) (NumericalElement, error)

		// BroadcastInDim the data of an array across dimensions.
		BroadcastInDim(ctx ir.Evaluator, expr ir.Expr, x NumericalElement, axisLengths []NumericalElement) (NumericalElement, error)

		// Concat concatenates scalars elements into an array with one axis.
		Concat(ctx ir.Evaluator, expr ir.Expr, xs []NumericalElement) (NumericalElement, error)

		// Set a slice in an array.
		Set(ctx ir.Evaluator, expr *ir.FuncCallExpr, x, updates ir.Element, position []ir.Element) (ir.Element, error)

		// ElementFromHostValue returns transforms a number element into an element specific to the ArrayOps implementation.
		ElementFromHostValue(ctx ir.Evaluator, el Constant) (ConstantElement, error)

		// DefineGlobalConst defines a global constant for the interpreter.
		// Called to define the builtin true and false.
		DefineGlobalConst(c ir.Storage, el ir.Element) (ir.Element, error)

		// EvalCond evaluates a boolean condition, typically for an if statement.
		EvalCond(ctx ir.Evaluator, el ir.Element) (bool, error)
	}

	// ConstantElement is an element (created by the ArrayOps)
	// with an underlying HostValueElement.
	ConstantElement interface {
		NumericalElement
		Constant() Constant
	}

	// Copier is an element that needs to be copied when passed to a function.
	Copier interface {
		ir.Element
		Copy() Copier
	}

	// Slice is an element implementing a slice.
	Slice interface {
		ir.Element
		Append(*ir.FuncCallExpr, []ir.Element) Slice
	}
)

// Copy an element if the element supports it.
// Returns the same element otherwise.
func Copy(el ir.Element) ir.Element {
	copier, isCopier := el.(Copier)
	if !isCopier {
		return el
	}
	return copier.Copy()
}
