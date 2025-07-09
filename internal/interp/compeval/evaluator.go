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
	"github.com/gx-org/gx/interp/context"
	"github.com/gx-org/gx/interp/elements"
	"github.com/gx-org/gx/interp/evaluator"
)

// CompEval is the evaluator used for compilation evaluation.
type CompEval struct {
	importer ir.Importer
}

var _ context.Evaluator = (*CompEval)(nil)

// NewHostEvaluator returns a new evaluator for the host.
func NewHostEvaluator(importer ir.Importer) *CompEval {
	return &CompEval{importer: importer}
}

// NewSub returns a new evaluator given a new array operator implementations.
func (ev *CompEval) NewSub(elements.ArrayOps) evaluator.Evaluator {
	return ev
}

// NewFunc creates a new function given its definition and a receiver.
func (ev *CompEval) NewFunc(ctx *context.Core, fn ir.Func, recv *elements.Receiver) elements.Func {
	if macro, ok := fn.(*ir.Macro); ok {
		return cpevelements.NewMacro(macro, recv)
	}
	return cpevelements.NewFunc(fn, recv)
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
func (ev *CompEval) ArrayOps() elements.ArrayOps {
	return hostArrayOps
}

// ElementFromAtom returns an element from a GX value.
func (ev *CompEval) ElementFromAtom(ctx ir.Evaluator, src ir.AssignableExpr, val values.Array) (elements.NumericalElement, error) {
	hostValue, err := val.ToHostArray(kernels.Allocator())
	if err != nil {
		return nil, err
	}
	return cpevelements.NewAtom(elements.NewExprAt(ctx.File(), src), hostValue)
}

// CallFuncLit calls a function literal.
func (ev *CompEval) CallFuncLit(ctx *context.Context, ref *ir.FuncLit, args []ir.Element) ([]ir.Element, error) {
	return nil, errors.Errorf("not implemented")
}

// Trace register a call to the trace builtin function.
func (ev *CompEval) Trace(ctx ir.Evaluator, call *ir.CallExpr, args []ir.Element) error {
	return errors.Errorf("not implemented")
}
