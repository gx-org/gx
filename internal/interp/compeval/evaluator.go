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

package compeval

import (
	"github.com/pkg/errors"
	"github.com/gx-org/gx/api/values"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/golang/backend/kernels"
	"github.com/gx-org/gx/internal/interp/compeval/cpevelements"
	"github.com/gx-org/gx/internal/tracer/processor"
	"github.com/gx-org/gx/interp/elements"
	"github.com/gx-org/gx/interp/evaluator"
	"github.com/gx-org/gx/interp/fun"
)

type (
	// NewRunFunc is a function to create a function that will be executed by the interpreter.
	NewRunFunc func(fn ir.Func, recv *fun.Receiver) fun.Func

	// CompEval is the evaluator used for compilation evaluation.
	CompEval struct {
		importer   ir.Importer
		newRunFunc NewRunFunc
	}
)

var _ fun.Evaluator = (*CompEval)(nil)

// NewHostEvaluator returns a new evaluator for the host.
func NewHostEvaluator(importer ir.Importer, newRunFunc NewRunFunc) *CompEval {
	return &CompEval{importer: importer, newRunFunc: newRunFunc}
}

// NewFunc creates a new function given its definition and a receiver.
func (ev *CompEval) NewFunc(fn ir.Func, recv *fun.Receiver) fun.Func {
	switch fnT := fn.(type) {
	case *ir.Annotator:
		return cpevelements.NewAnnotator(fnT, recv)
	case *ir.Macro:
		return cpevelements.NewMacro(fnT, recv)
	}
	return cpevelements.NewFunc(fn, recv)
}

// NewFuncLit creates a new function literal.
func (ev *CompEval) NewFuncLit(env *fun.CallEnv, fn *ir.FuncLit) (fun.Func, error) {
	return cpevelements.NewFunc(fn, nil), nil
}

// NewRunFunc returns a function that will be run by the interpreter.
func (ev *CompEval) NewRunFunc(fn ir.Func, recv *fun.Receiver) fun.Func {
	return ev.newRunFunc(fn, recv)
}

// Processor returns the processor used to process inits and traces for compiled function.
func (ev *CompEval) Processor() *processor.Processor {
	return nil
}

// Importer returns the importer used by the evaluator.
func (ev *CompEval) Importer() ir.Importer {
	return ev.importer
}

// ArrayOps returns the implementation used for array operations.
func (ev *CompEval) ArrayOps() evaluator.ArrayOps {
	return hostArrayOps
}

// ElementFromAtom returns an element from a GX value.
func (ev *CompEval) ElementFromAtom(file *ir.File, expr ir.Expr, val values.Array) (evaluator.NumericalElement, error) {
	hostValue, err := val.ToHostArray(kernels.Allocator())
	if err != nil {
		return nil, err
	}
	return cpevelements.NewAtom(elements.NewExprAt(file, expr), hostValue)
}

// ElementFromStorage returns an element from an atomic GX value and its storage.
func (ev *CompEval) ElementFromStorage(file *ir.File, storage ir.StorageWithValue, val ir.Element) ir.Element {
	return cpevelements.NewStoredValue(file, storage, val)
}

// Trace register a call to the trace builtin function.
func (ev *CompEval) Trace(ctx ir.Evaluator, call *ir.FuncCallExpr, args []ir.Element) error {
	return errors.Errorf("not implemented")
}
