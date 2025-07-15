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

// Package evaluator defines interfaces for the interpreter to use to evaluate GX code.
package evaluator

import (
	"github.com/gx-org/backend/ops"
	"github.com/gx-org/gx/api/values"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/tracer/processor"
)

type (
	// Context in which an operator is being executed.
	Context interface {
		ir.Evaluator

		// Evaluator returns the evaluator used by the interpreter.
		Evaluator() Evaluator
	}

	// Evaluator implements GX operators.
	Evaluator interface {
		// Importer returns an importer to import package.
		Importer() ir.Importer

		// ArrayOps returns the operator implementations for arrays.
		ArrayOps() ArrayOps

		// Processor returns the used by the array evaluator.
		Processor() *processor.Processor

		// ElementFromAtom returns an element from an atomic GX value.
		ElementFromAtom(ctx ir.Evaluator, expr ir.AssignableExpr, val values.Array) (NumericalElement, error)

		// Trace register a call to the trace builtin function.
		Trace(ctx ir.Evaluator, call *ir.CallExpr, args []ir.Element) error
	}

	// NumericalElement is a node representing a numerical value.
	NumericalElement interface {
		ir.Element

		// UnaryOp applies a unary operator on x.
		UnaryOp(ctx ir.Evaluator, expr *ir.UnaryExpr) (NumericalElement, error)

		// BinaryOp applies a binary operator to x and y.
		// Note that the receiver can be either the left or right argument.
		BinaryOp(ctx ir.Evaluator, expr *ir.BinaryExpr, x, y NumericalElement) (NumericalElement, error)

		// Cast an element into a given data type.
		Cast(ctx ir.Evaluator, expr ir.AssignableExpr, target ir.Type) (NumericalElement, error)

		// Reshape an element.
		Reshape(ctx ir.Evaluator, expr ir.AssignableExpr, axisLengths []NumericalElement) (NumericalElement, error)
	}

	// ArrayOps are the operator implementations for arrays.
	ArrayOps interface {
		// Graph returns the graph to which new nodes are being added.
		Graph() ops.Graph

		// SubGraph returns a new graph builder.
		SubGraph(name string) (ArrayOps, error)

		// Einsum calls an einstein sum on x and y given the expression in ref.
		Einsum(ctx ir.Evaluator, expr *ir.EinsumExpr, x, y NumericalElement) (NumericalElement, error)

		// BroadcastInDim the data of an array across dimensions.
		BroadcastInDim(ctx ir.Evaluator, expr ir.AssignableExpr, x NumericalElement, axisLengths []NumericalElement) (NumericalElement, error)

		// Concat concatenates scalars elements into an array with one axis.
		Concat(ctx ir.Evaluator, expr ir.AssignableExpr, xs []NumericalElement) (NumericalElement, error)

		// Set a slice in an array.
		Set(ctx ir.Evaluator, expr *ir.CallExpr, x, updates, index ir.Element) (ir.Element, error)

		// ElementFromArray returns an element from an array GX value.
		ElementFromArray(ctx ir.Evaluator, expr ir.AssignableExpr, val values.Array) (NumericalElement, error)
	}
)
