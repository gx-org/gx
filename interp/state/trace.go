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

package state

import (
	"github.com/gx-org/backend/platform"
	"github.com/gx-org/gx/api"
	"github.com/gx-org/gx/api/values"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/interp/elements"
)

type trace struct {
	call   elements.CallAt
	traced []Element
}

func (t *trace) parse(tracer Tracer, parser *handleParser) error {
	vals := make([]values.Value, len(t.traced))
	for i, tr := range t.traced {
		var err error
		vals[i], err = parser.parse(tr)
		if err != nil {
			return err
		}
	}
	return tracer.Trace(t.call.FSet(), t.call.Node(), vals)
}

type traces struct {
	state   *State
	traces  []*trace
	flatten []Element
}

// Trace a set of elements.
func (s *State) Trace(call elements.CallAt, fn *Func, irFunc *ir.FuncBuiltin, args []Element, ctx Context) error {
	s.traces.traces = append(s.traces.traces, &trace{
		call:   call,
		traced: args,
	})
	for _, arg := range args {
		flatten, err := arg.Flatten()
		if err != nil {
			return err
		}
		s.traces.flatten = append(s.traces.flatten, flatten...)
	}
	return nil
}

func (ts *traces) process(dev *api.Device, ctx Context, tracer Tracer, aux []platform.DeviceHandle) error {
	if tracer == nil || len(ts.traces) == 0 {
		return nil
	}
	parser := newHandleParser(dev, ctx, aux)
	for _, trace := range ts.traces {
		if err := trace.parse(tracer, parser); err != nil {
			return err
		}
	}
	return nil
}

func (ts *traces) asTuple() *elements.Tuple {
	return elements.NewTuple(nil, nil, ts.flatten)
}
