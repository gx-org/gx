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

// Package cfunc is a function that has been compiled by a backend.
package cfunc

import (
	"github.com/gx-org/backend/ops"
	"github.com/gx-org/backend/platform"
	"github.com/gx-org/backend/shape"
	"github.com/gx-org/gx/api"
	"github.com/gx-org/gx/api/trace"
	"github.com/gx-org/gx/api/values"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/tracer/processor"
	"github.com/gx-org/gx/interp/elements"
)

// CompiledFunc is a graph compiled for a device.
type CompiledFunc struct {
	fn      ir.Func
	process *processor.Processor
	device  *api.Device
	runner  ops.Runner
	outs    []elements.Element
}

func extractGraphNodes(ao elements.ArrayOps, els []elements.Element) ([]*ops.OutputNode, error) {
	flatten, err := elements.Flatten(els...)
	if err != nil {
		return nil, err
	}
	var nodes []*ops.OutputNode
	for _, el := range flatten {
		mat, ok := el.(elements.Materialiser)
		if !ok {
			continue
		}
		node, err := mat.Materialise(ao)
		if err != nil {
			return nil, err
		}
		nodes = append(nodes, node.OutNode())
	}
	return nodes, nil
}

// Compile a function that will be run on a device given some inputs.
func Compile(dev *api.Device, fn ir.Func, p *processor.Processor, ao elements.ArrayOps, outs []elements.Element) (*CompiledFunc, error) {
	args := p.Args()
	paramShapes := make([]*shape.Shape, len(args))
	for i, arg := range args {
		paramShapes[i] = arg.Shape()
	}
	cg := &CompiledFunc{
		fn:      fn,
		process: p,
		device:  dev,
		outs:    outs,
	}
	// Flatten out and get all the backend graph nodes from the list.
	graphOutNodes, err := extractGraphNodes(ao, outs)
	if err != nil {
		return nil, err
	}
	graphAuxNodes, err := extractGraphNodes(ao, p.Traces())
	if err != nil {
		return nil, err
	}
	if len(graphOutNodes) == 0 && len(graphAuxNodes) == 0 {
		// Nothing to run in the graph.
		// We do not need to compile and we leave cg.runner to nil.
		return cg, nil
	}
	cg.runner, err = ao.Graph().Compile(dev.PlatformDevice(), graphOutNodes, graphAuxNodes, paramShapes)
	return cg, err
}

func (g *CompiledFunc) run(ctx *values.FuncInputs) (out, traces []platform.DeviceHandle, err error) {
	if g.runner == nil {
		// No runner: there is nothing to run in the graph.
		return nil, nil, nil
	}
	args := g.process.Args()
	handles := make([]platform.Handle, len(args))
	for i, arg := range args {
		var err error
		handles[i], err = arg.ToDeviceHandle(g.device.PlatformDevice(), ctx)
		if err != nil {
			return nil, nil, err
		}
	}
	return g.runner.Run(handles)
}

// Run the graph.
func (g *CompiledFunc) Run(receiver values.Value, args []values.Value, tracer trace.Callback) ([]values.Value, error) {
	fc := &values.FuncInputs{Receiver: receiver, Args: args}
	if err := g.process.ProcessInits(fc); err != nil {
		return nil, err
	}
	out, traced, err := g.run(fc)
	if err != nil {
		return nil, err
	}
	if err := g.process.ProcessTraces(g.device.PlatformDevice(), fc, tracer, traced); err != nil {
		return nil, err
	}
	return g.handlesToValues(fc, out)
}

func (g *CompiledFunc) handlesToValues(in *values.FuncInputs, handles []platform.DeviceHandle) ([]values.Value, error) {
	reader := elements.NewUnflattener(g.device.PlatformDevice(), in, handles)
	values := make([]values.Value, len(g.outs))
	for i, el := range g.outs {
		val, err := reader.Unflatten(el)
		if err != nil {
			return nil, err
		}
		values[i] = val
	}
	return values, nil
}
