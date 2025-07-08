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
	"github.com/gx-org/gx/api/values"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/tracer/processor"
	"github.com/gx-org/gx/interp/elements"
)

type (
	// Context in which an operator is being executed.
	Context interface {
		ir.Evaluator

		// NewFileContext returns a context for a given file.
		NewFileContext(*ir.File) (Context, error)

		// Evaluator returns the evaluator used by the interpreter.
		Evaluator() Evaluator

		// Sub returns a new context with additional variables being defined.
		Sub(map[string]ir.Element) (Context, error)

		// EvalFunctionToElement evaluates a function literal into an interpreter element.
		EvalFunctionToElement(eval Evaluator, fn ir.Func, args []elements.Element) ([]elements.Element, error)
	}

	// Importer imports packages given their path.
	Importer interface {
		// Import a package given its path.
		Import(pkgPath string) (*ir.Package, error)
	}

	// Evaluator implements GX operators.
	Evaluator interface {
		// NewFunc creates a new function given its definition and a receiver.
		NewFunc(fn ir.Func, recv *elements.Receiver) elements.Func

		// Importer returns an importer to import package.
		Importer() Importer

		// ArrayOps returns the operator implementations for arrays.
		ArrayOps() elements.ArrayOps

		// Processor returns the used by the array evaluator.
		Processor() *processor.Processor

		// ElementFromAtom returns an element from an atomic GX value.
		ElementFromAtom(src elements.ExprAt, val values.Array) (elements.NumericalElement, error)

		// CallFuncLit calls a function literal.
		CallFuncLit(ctx Context, ref *ir.FuncLit, args []elements.Element) ([]elements.Element, error)

		// Trace register a call to the trace builtin function.
		Trace(call elements.CallAt, fn elements.Func, irFunc *ir.FuncBuiltin, args []elements.Element, fc *values.FuncInputs) error
	}
)
