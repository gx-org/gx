// Copyright 2026 Google LLC
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

package engine

import (
	"github.com/gomlx/compute/shapes"
	"github.com/gx-org/gx/build/ir"
)

type (
	// NumericalElement is a node representing a numerical value manipulated by the interpreter.
	NumericalElement interface {
		ir.Element

		// UnaryOp applies a unary operator on x.
		UnaryOp(env *Env, expr *ir.UnaryExpr) (NumericalElement, error)

		// BinaryOp applies a binary operator to x and y.
		// Note that the receiver can be either the left or right argument.
		BinaryOp(env *Env, expr *ir.BinaryExpr, y NumericalElement) (NumericalElement, error)

		// Cast an element into a given data type.
		Cast(env *Env, expr ir.Expr, target ir.Type) (NumericalElement, error)

		// Reshape an element.
		Reshape(env *Env, expr ir.Expr, axisLengths []NumericalElement) (NumericalElement, error)
	}

	// Element returned after an evaluation at compeval.
	Element interface {
		NumericalElement
		ir.WithExpr
		ir.StringSourcer
		WithAxes
		EvalShaper
		Slicer
	}

	// WithAxes is an element able to return its axes as a slice of element.
	WithAxes interface {
		Axes(ev ir.Evaluator) (ir.FixedSlice, error)
	}

	// EvalShaper is an (array) element from which the shape has been fully determined at evaluation time.
	EvalShaper interface {
		ir.Element
		EvalShape() (shapes.Shape, error)
	}

	// Slicer is a state element that can be sliced.
	Slicer interface {
		SliceAt(env *Env, expr *ir.IndexExpr, index NumericalElement) (ir.Element, error)
		Slice(env *Env, expr *ir.SliceExpr, low, high NumericalElement) (ir.Element, error)
	}
)
