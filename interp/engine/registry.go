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
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/tracer/processor"
)

// Registry registers element necessary for the GX interpreter.
type Registry struct {
	// Importer imports packages.
	Importer ir.Importer

	// CompEvalEngine returns the engine used for compeval functions.
	CompEvalEngine Engine

	// HostEngine returns core operations for the code running on the host.
	HostEngine Engine

	// Processor registers inputs and outputs.
	Processor *processor.Processor

	// Runners to evaluate functions.
	Runners Runners

	// Trace register a call to the trace builtin function.
	Trace func(ctx ir.Evaluator, call *ir.FuncCallExpr, args []ir.Element) error
}

// Engine returns the engine matching a given function nature.
func (reg *Registry) Engine(nature ir.FuncNature) Engine {
	if nature.CompEval {
		return reg.CompEvalEngine
	}
	return reg.HostEngine
}
