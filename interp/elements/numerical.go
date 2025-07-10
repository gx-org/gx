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
	"github.com/gx-org/gx/api/values"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/interp/evaluator"
)

type (
	// FuncEvaluator is a context able to evaluate a function.
	FuncEvaluator interface {
		EvalFunc(f ir.Func, call *ir.CallExpr, args []ir.Element) ([]ir.Element, error)
	}

	// ElementWithConstant is an element with a concrete value that is already known.
	ElementWithConstant interface {
		evaluator.NumericalElement

		// NumericalConstant returns the value of a constant represented by a node.
		NumericalConstant() *values.HostArray
	}

	// ElementWithArrayFromContext is an element able to return a concrete value from the current context.
	// For example, a value passed as an argument to the function.
	ElementWithArrayFromContext interface {
		evaluator.NumericalElement

		// ArrayFromContext fetches an array from the argument.
		ArrayFromContext(*values.FuncInputs) (values.Array, error)
	}
)
