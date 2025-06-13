// Copyright 2025 Google LLC
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
	"github.com/gx-org/backend/shape"
	"github.com/gx-org/gx/api/values"
	"github.com/gx-org/gx/build/ir"
)

type (
	// FileContext is context in which an op is being executed.
	FileContext interface {
		// File in which the operator is being executed.
		File() *ir.File
	}

	// NumericalElement is a node representing a numerical value.
	NumericalElement interface {
		Element

		// UnaryOp applies a unary operator on x.
		UnaryOp(ctx FileContext, expr *ir.UnaryExpr) (NumericalElement, error)

		// BinaryOp applies a binary operator to x and y.
		// Note that the receiver can be either the left or right argument.
		BinaryOp(ctx FileContext, expr *ir.BinaryExpr, x, y NumericalElement) (NumericalElement, error)

		// Cast an element into a given data type.
		Cast(ctx FileContext, expr ir.AssignableExpr, target ir.Type) (NumericalElement, error)

		// Shape of the value represented by the element.
		Shape() *shape.Shape
	}

	// ElementWithConstant is an element with a concrete value that is already known.
	ElementWithConstant interface {
		NumericalElement

		// NumericalConstant returns the value of a constant represented by a node.
		NumericalConstant() *values.HostArray
	}

	// ElementWithArrayFromContext is an element able to return a concrete value from the current context.
	// For example, a value passed as an argument to the function.
	ElementWithArrayFromContext interface {
		NumericalElement

		// ArrayFromContext fetches an array from the argument.
		ArrayFromContext(*InputValues) (values.Array, error)
	}
)
