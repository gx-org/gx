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
	"go/ast"

	"github.com/gx-org/backend/ops"
	"github.com/gx-org/backend/shape"
	"github.com/gx-org/gx/api/values"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/interp/canonical"
	"github.com/gx-org/gx/internal/tracer/processor"
)

type (
	// Env is the environment in which GX code is evaluated.
	Env interface {
		// File returns the file in which the code is currently being executed.
		File() *ir.File

		// ExprEval returns an expression evaluator.
		ExprEval() ir.Evaluator

		// Engine returns the engine used by the interpreter environment.
		Engine() Engine
	}

	// Engine provides the GX interpreter will all the required primitives.
	Engine interface {
		// Importer returns an importer to import package.
		Importer() ir.Importer

		// ArrayOps returns the operator implementations for arrays.
		ArrayOps() ArrayOps

		// Processor returns the used by the array evaluator.
		Processor() *processor.Processor

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
		Cast(env Env, expr ir.Expr, target ir.Type) (NumericalElement, error)

		// Reshape an element.
		Reshape(env Env, expr ir.Expr, axisLengths []NumericalElement) (NumericalElement, error)
	}

	// AtomLitElement is an element representing a value known at compile time.
	// For example: float32(1), int(5), true
	AtomLitElement interface {
		NumericalElement
		canonical.Comparable
		ir.WithExpr
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
		BroadcastInDim(ctx ir.Evaluator, expr ir.Expr, x NumericalElement, axisLengths []NumericalElement) (NumericalElement, error)

		// Concat concatenates scalars elements into an array with one axis.
		Concat(ctx ir.Evaluator, expr ir.Expr, xs []NumericalElement) (NumericalElement, error)

		// Set a slice in an array.
		Set(ctx ir.Evaluator, expr *ir.FuncCallExpr, x, updates ir.Element, position []ir.Element) (ir.Element, error)

		// ElementFromAtomLit returns transforms a number element into an element specific to the ArrayOps implementation.
		ElementFromAtomLit(ctx *ir.File, el AtomLitElement) (NumericalElement, error)

		// ElementFromAtom returns an element from an atomic GX value.
		ElementFromAtom(ctx *ir.File, val values.Array, expr ir.Expr, typ ir.Type) (NumericalElement, error)

		// ElementFromArray returns an element from an array GX value.
		ElementFromArray(ctx *ir.File, lit *ir.ArrayLitExpr, val values.Array) (NumericalElement, error)
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

type evalEnv struct{}

func (evalEnv) File() *ir.File {
	return nil
}

func (evalEnv) ExprEval() ir.Evaluator {
	return nil
}

func (evalEnv) Engine() Engine {
	return nil
}

func (evalEnv) ToConcrete(_ ast.Expr, tp ir.Type) (ir.Type, ir.CompEvalError, error) {
	return tp, nil, nil
}

var env evalEnv

// ProxyEnv returns a proxy implementation of the Env interface.
func ProxyEnv() Env {
	return env
}
