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

// Package tracer implements a context and evaluator for the interpreter
// which builds a graph by tracing operators being executed when GX code is interpreted.
package tracer

import (
	"fmt"

	"github.com/gx-org/backend/ops"
	"github.com/gx-org/gx/api"
	"github.com/gx-org/gx/api/options"
	"github.com/gx-org/gx/api/trace"
	"github.com/gx-org/gx/api/values"
	"github.com/gx-org/gx/build/fmterr"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/tracer/cfunc"
	"github.com/gx-org/gx/internal/tracer/processor"
	"github.com/gx-org/gx/interp/grapheval"
	"github.com/gx-org/gx/interp"
)

type (
	// tracer interprets GX code to build a backend graph.
	// The graph is then compiled to return a compiled function.
	tracer struct {
		graph ops.Graph
	}

	// CompiledFunc is a function compiled for a given receiver and argument shapes
	// which is ready to run.
	CompiledFunc interface {
		// Run the function for the given GX values.
		Run(receiver values.Value, args []values.Value, tracer trace.Callback) ([]values.Value, error)
	}
)

// Trace a function and returns a runner to run the function on the device.
func Trace(dev *api.Device, fn *ir.FuncDecl, receiver values.Value, args []values.Value, options []options.PackageOption) (_ CompiledFunc, err error) {
	defer func() {
		if err != nil {
			recvName := ""
			if recv := fn.FType.ReceiverField(); recv != nil {
				recvName = recv.Name.Name + "."
			}
			err = fmt.Errorf("%s.%s%s evaluation error:\n%w", fn.File().Package.FullName(), recvName, fn.Name(), err)
		}
		err = fmterr.ToStackTraceError(err)
	}()
	// Create a new evaluator for the interpreter.
	proc := &processor.Processor{}
	graph := dev.Runtime().Backend().NewOps(fn.FullyQualifiedName())
	tr := &tracer{
		graph: graph,
	}
	ev := grapheval.New(dev.Runtime().Builder(), proc, tr.graph)
	itp, err := interp.New(ev, options)
	if err != nil {
		return nil, err
	}
	// Transform the receiver and arguments values into elements for the interpreter.
	in, err := ev.FuncInputsToElements(itp, fn.File(), fn.FuncType(), receiver, args)
	if err != nil {
		return nil, err
	}

	// Interpret the function with the evaluator to build the graph.
	// The evaluation returns a single output element.
	outs, err := itp.EvalFunc(fn, in)
	if err != nil {
		return nil, err
	}
	// Compile the resulting graph given the output element.
	return cfunc.Compile(dev, fn, proc, ev.Materialiser(), outs)
}
