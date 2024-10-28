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
	"github.com/pkg/errors"
	"github.com/gx-org/backend/graph"
	"github.com/gx-org/backend/platform"
	"github.com/gx-org/gx/api/values"
	"github.com/gx-org/gx/build/fmterr"
	"github.com/gx-org/gx/build/ir"
)

type (
	valueFetcher func(aux []platform.DeviceHandle) (values.Value, error)

	traceFetcher struct {
		call     CallAt
		fetchers []valueFetcher
	}

	nodeTracer struct {
		cGraph *CompiledGraph
		nodes  []*graph.OutputNode
		traces []*traceFetcher
	}
)

// Trace a set of elements.
func (s *State) Trace(call CallAt, fn *Func, irFunc *ir.FuncBuiltin, args []Element, ctx Context) error {
	fetchers := make([]valueFetcher, len(args))
	for i, arg := range args {
		var f valueFetcher
		switch argT := arg.(type) {
		case backendElement:
			node, sh, err := NodeFromElement(argT)
			if err != nil {
				return err
			}
			f = s.tracer.fetcher(arg, &graph.OutputNode{
				Node:  node,
				Shape: sh,
			})
		case ElementWithConstant:
			f = func([]platform.DeviceHandle) (values.Value, error) { return ctx.Args()[i], nil }
		default:
			return fmterr.Position(call.FSet(), call.ExprT().Src, errors.Errorf("cannot trace element %d:%T", i, argT))
		}
		fetchers[i] = f
	}
	s.tracer.traces = append(s.tracer.traces, &traceFetcher{
		call:     call,
		fetchers: fetchers,
	})
	return nil
}

func (t *nodeTracer) fetcher(el Element, node *graph.OutputNode) valueFetcher {
	nodeID := len(t.nodes)
	t.nodes = append(t.nodes, node)
	return func(handles []platform.DeviceHandle) (values.Value, error) {
		return (&handleParser{handles: handles[nodeID : nodeID+1]}).parse(el)
	}
}

func (t *nodeTracer) process(tracer Tracer, aux []platform.DeviceHandle) error {
	if tracer == nil || len(t.traces) == 0 {
		return nil
	}
	for _, trace := range t.traces {
		values, err := trace.fetchAll(aux)
		if err != nil {
			return err
		}
		if err := tracer.Trace(trace.call.FSet(), trace.call.ExprT(), values); err != nil {
			return err
		}
	}
	return nil
}

func (f *traceFetcher) fetchAll(aux []platform.DeviceHandle) ([]values.Value, error) {
	vals := make([]values.Value, len(f.fetchers))
	for i, fetcher := range f.fetchers {
		var err error
		vals[i], err = fetcher(aux)
		if err != nil {
			return nil, err
		}
	}
	return vals, nil
}
