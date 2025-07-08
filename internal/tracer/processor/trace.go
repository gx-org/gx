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

package processor

import (
	"github.com/gx-org/backend/platform"
	"github.com/gx-org/gx/api/trace"
	"github.com/gx-org/gx/api/values"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/interp/elements"
)

type traceProcessor struct {
	call   elements.CallAt
	traced []elements.Element
}

func (t *traceProcessor) parse(tracer trace.Callback, parser *elements.Unflattener) error {
	vals := make([]values.Value, len(t.traced))
	for i, tr := range t.traced {
		var err error
		vals[i], err = parser.Unflatten(tr)
		if err != nil {
			return err
		}
	}
	return tracer.Trace(t.call.FSet(), t.call.Node(), vals)
}

type traces struct {
	traces  []*traceProcessor
	flatten []elements.Element
}

// Trace a set of elements.
func (ts *traces) Trace(call elements.CallAt, fn *elements.Func, irFunc *ir.FuncBuiltin, args []elements.Element, ctx *values.FuncInputs) error {
	ts.traces = append(ts.traces, &traceProcessor{
		call:   call,
		traced: args,
	})
	for _, arg := range args {
		flatten, err := arg.Flatten()
		if err != nil {
			return err
		}
		ts.flatten = append(ts.flatten, flatten...)
	}
	return nil
}

// RegisterTrace registers a call to the trace builtin.
func (ts *traces) RegisterTrace(call elements.CallAt, fn elements.Func, irFunc *ir.FuncBuiltin, args []elements.Element, ctx *values.FuncInputs) error {
	ts.traces = append(ts.traces, &traceProcessor{
		call:   call,
		traced: args,
	})
	for _, arg := range args {
		flatten, err := arg.Flatten()
		if err != nil {
			return err
		}
		ts.flatten = append(ts.flatten, flatten...)
	}
	return nil
}

// ProcessTraces processes the graph outputs related to traces.
func (ts *traces) ProcessTraces(dev platform.Device, in *values.FuncInputs, tracer trace.Callback, aux []platform.DeviceHandle) error {
	if tracer == nil || len(ts.traces) == 0 {
		return nil
	}
	parser := elements.NewUnflattener(dev, in, aux)
	for _, trace := range ts.traces {
		if err := trace.parse(tracer, parser); err != nil {
			return err
		}
	}
	return nil
}

// Traces returns a tuple of all the trace nodes.
func (ts *traces) Traces() []elements.Element {
	return ts.flatten
}
