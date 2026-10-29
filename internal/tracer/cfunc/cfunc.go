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
	"github.com/gomlx/compute"
	"github.com/gx-org/gx/api"
	"github.com/gx-org/gx/api/hostio"
	"github.com/gx-org/gx/api/trace"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/interp/flatten"
	"github.com/gx-org/gx/internal/togo"
	"github.com/gx-org/gx/internal/tracer/processor"
	"github.com/gx-org/gx/interp/materialise"
)

// CompiledFunc is a graph compiled for a device.
type CompiledFunc struct {
	fn      ir.Func
	process *processor.Processor
	device  *api.Device
	runner  compute.Executable
	outs    []ir.Element
	numOut  int
}

func extractGraphNodes(ao materialise.Materialiser, els []ir.Element) ([]compute.Value, error) {
	flatten, err := flatten.Flatten(els...)
	if err != nil {
		return nil, err
	}
	var nodes []compute.Value
	for _, el := range flatten {
		mat, ok := el.(materialise.ElementMaterialiser)
		if !ok {
			continue
		}
		node, err := mat.Materialise(ao)
		if err != nil {
			return nil, err
		}
		nodes = append(nodes, node.Value())
	}
	return nodes, nil
}

// Compile a function that will be run on a device given some inputs.
func Compile(dev *api.Device, fn ir.Func, p *processor.Processor, ao materialise.Materialiser, bld compute.Builder, outs []ir.Element) (*CompiledFunc, error) {
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
	cg.numOut = len(graphOutNodes)
	graphAuxNodes, err := extractGraphNodes(ao, p.Traces())
	if err != nil {
		return nil, err
	}
	if len(graphOutNodes) == 0 && len(graphAuxNodes) == 0 {
		// Nothing to run in the graph.
		// We do not need to compile and we leave cg.runner to nil.
		return cg, nil
	}
	if err := bld.DeviceAssignment(dev.PlatformDevice()); err != nil {
		return nil, err
	}
	allOutNodes := append(append([]compute.Value{}, graphOutNodes...), graphAuxNodes...)
	if err := bld.Main().Return(allOutNodes, nil); err != nil {
		return nil, err
	}
	cg.runner, err = bld.Compile()
	return cg, err
}

func (g *CompiledFunc) run(ctx *togo.FuncInputs) (out, traces []compute.Buffer, err error) {
	if g.runner == nil {
		// No runner: there is nothing to run in the graph.
		return nil, nil, nil
	}
	args := g.process.Args()
	handles := make([]compute.Buffer, len(args))
	for i, arg := range args {
		var err error
		handles[i], err = arg.ToDeviceHandle(g.device.Backend(), g.device.PlatformDevice(), ctx)
		if err != nil {
			return nil, nil, err
		}
	}
	results, err := g.runner.Execute(handles, nil, g.device.PlatformDevice())
	if err != nil {
		return nil, nil, err
	}
	return results[:g.numOut], results[g.numOut:], nil
}

// Run the graph.
func (g *CompiledFunc) Run(receiver hostio.Value, args []hostio.Value, tracer trace.Callback) ([]hostio.Value, error) {
	fc := &togo.FuncInputs{Receiver: receiver, Args: hostio.ToElements(args)}
	if err := g.process.ProcessInits(fc); err != nil {
		return nil, err
	}
	out, traced, err := g.run(fc)
	if err != nil {
		return nil, err
	}
	defer func() {
		for _, b := range traced {
			_ = b.Finalize()
		}
	}()
	if err := g.process.ProcessTraces(g.device.PlatformDevice(), fc, tracer, traced); err != nil {
		return nil, err
	}
	return g.handlesToValues(fc, out)
}

func (g *CompiledFunc) handlesToValues(in *togo.FuncInputs, handles []compute.Buffer) ([]hostio.Value, error) {
	reader := flatten.NewParser(g.device.PlatformDevice(), in, handles)
	values := make([]hostio.Value, len(g.outs))
	for i, el := range g.outs {
		val, err := reader.Unflatten(el)
		if err != nil {
			return nil, err
		}
		values[i] = val
	}
	return values, nil
}
