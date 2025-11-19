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
	"github.com/gx-org/backend/shape"
	"github.com/gx-org/gx/api/values"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/tracer/processor"
)

type (
	// Env is the environment in which GX code is evaluated.
	Env interface {
		// File returns the file in which the code is currently being executed.
		File() *ir.File

		// ExprEval returns an expression evaluator.
		ExprEval() ir.Evaluator

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
		ElementFromAtom(file *ir.File, expr ir.AssignableExpr, val values.Array) (NumericalElement, error)

		// ElementFromStorage returns an element from an atomic GX value and its storage.
		ElementFromStorage(file *ir.File, expr ir.StorageWithValue, val ir.Element) ir.Element

		// Trace register a call to the trace builtin function.
		Trace(ctx ir.Evaluator, call *ir.FuncCallExpr, args []ir.Element) error
	}

	// NumericalElement is a node representing a numerical value.
	NumericalElement interface {
		ir.Element

		// UnaryOp applies a unary operator on x.
		UnaryOp(env Env, expr *ir.UnaryExpr) (NumericalElement, error)

		// BinaryOp applies a binary operator to x and y.
		// Note that the receiver can be either the left or right argument.
		BinaryOp(env Env, expr *ir.BinaryExpr, x, y NumericalElement) (NumericalElement, error)

		// Cast an element into a given data type.
		Cast(env Env, expr ir.AssignableExpr, target ir.Type) (NumericalElement, error)

		// Reshape an element.
		Reshape(env Env, expr ir.AssignableExpr, axisLengths []NumericalElement) (NumericalElement, error)
	}

	// ArrayOps are the operator implementations for arrays.
	ArrayOps interface {
		// Graph returns the graph to which new nodes are being added.
		Graph() ops.Graph

		// SubGraph returns a new graph builder.
		SubGraph(name string, args []*shape.Shape) (ArrayOps, error)

		// Einsum calls an einstein sum on x and y given the expression in ref.
		Einsum(ctx ir.Evaluator, expr *ir.EinsumExpr, x, y NumericalElement) (NumericalElement, error)

		// BroadcastInDim the data of an array across dimensions.
		BroadcastInDim(ctx ir.Evaluator, expr ir.AssignableExpr, x NumericalElement, axisLengths []NumericalElement) (NumericalElement, error)

		// Concat concatenates scalars elements into an array with one axis.
		Concat(ctx ir.Evaluator, expr ir.AssignableExpr, xs []NumericalElement) (NumericalElement, error)

		// Set a slice in an array.
		Set(ctx ir.Evaluator, expr *ir.FuncCallExpr, x, updates, index ir.Element) (ir.Element, error)

		// ElementFromArray returns an element from an array GX value.
		ElementFromArray(ctx ir.Evaluator, expr ir.AssignableExpr, val values.Array) (NumericalElement, error)
	}
)
