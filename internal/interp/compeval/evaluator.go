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
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/tracer/processor"
	"github.com/gx-org/gx/interp/engine"
)

// CompEval is the evaluator used for compilation evaluation.
type CompEval struct {
	importer ir.Importer
}

// NewHostEvaluator returns a new evaluator for the host.
func NewHostEvaluator(importer ir.Importer) *CompEval {
	return &CompEval{importer: importer}
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
func (ev *CompEval) ArrayOps() engine.ArrayOps {
	return hostArrayOps
}

// Trace register a call to the trace builtin function.
func (ev *CompEval) Trace(ctx ir.Evaluator, call *ir.FuncCallExpr, args []ir.Element) error {
	return errors.Errorf("not implemented")
}
