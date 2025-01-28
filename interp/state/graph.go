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

// Package state provides elements evaluated by the GX interpreter
//
// The GX interpreter runs GX code and builds a computation graph composed of two sets of nodes:
//  1. nodes implemented by a backend (e.g. XLA). For example, this can be
//     a XLA tensor node or an XLA op node.
//  2. generic nodes (i.e. not specific to a backend) keeping references
//     to other nodes. For example, a GX structure composed of fields pointing
//     to tensors is represented by a structure node in the graph that keeps
//     references to the corresponding instances of XLA tensor node
//     (for the XLA backend).
package state

import (
	"go/token"

	"github.com/gx-org/backend/graph"
	"github.com/gx-org/backend/platform"
	"github.com/gx-org/backend/shape"
	"github.com/gx-org/gx/api"
	"github.com/gx-org/gx/api/values"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/interp/elements"
)

type (
	deviceHandler interface {
		deviceHandle() values.Value
	}

	// Initializer is called at the beginning of a run before
	// arguments for the backend are computed.
	Initializer interface {
		Init(*elements.CallInputs) error
	}

	// Argument provides an argument to pass to the backend.
	Argument interface {
		Shape() *shape.Shape

		ToDeviceHandle(*api.Device, *elements.CallInputs) (platform.DeviceHandle, error)
	}
	// Tracer receives traced values from a run.
	Tracer interface {
		Trace(fset *token.FileSet, call *ir.CallExpr, values []values.Value) error
	}

	// State of a collection of nodes.
	State struct {
		backendGraph graph.Graph
		fn           ir.Func
		inits        []Initializer
		args         []Argument

		traces traces
	}

	// Element is a value in the interpreter.
	Element = elements.Element

	// CompiledGraph is a graph compiled for a device.
	CompiledGraph struct {
		device *api.Device
		state  *State
		runner graph.Runner
		out    Element
	}

	// Materialiser is an element that can return an instance of itself composed only of elements from the backend graph.
	Materialiser interface {
		Element
		// Materialise returns the element with all its values from the graph.
		Materialise() (*graph.OutputNode, error)
	}

	emptyRunner struct{}
)

// New interpreter state.
func New(fn ir.Func, backendGraph graph.Graph) *State {
	s := &State{fn: fn, backendGraph: backendGraph}
	s.traces.state = s
	return s
}

// BackendGraph returns the graph maintained by the backend.
func (g *State) BackendGraph() graph.Graph {
	return g.backendGraph
}

// RegisterInit registers an Initializer to the graph.
func (g *State) RegisterInit(init Initializer) {
	g.inits = append(g.inits, init)
}

// RegisterArg an argument for the backend.
// Returns the index of the argument.
func (g *State) RegisterArg(arg Argument) int {
	index := len(g.args)
	g.args = append(g.args, arg)
	return index
}

// ExtractGraphNodes extracts all the graph nodes from an element and its children.
// It first flattens the element, then extracts all the BackendNodes (that is, any elements
// representing a node in the backend graph).
func ExtractGraphNodes(out Element) ([]*graph.OutputNode, error) {
	flatten, err := out.Flatten()
	if err != nil {
		return nil, err
	}
	var graphNodes []*graph.OutputNode
	for _, elt := range flatten {
		node, ok := elt.(*BackendNode)
		if !ok {
			continue
		}
		graphNodes = append(graphNodes, node.nod)
	}
	return graphNodes, nil
}

// Compile a node given a set of parameters and using this node as an output.
// Returns a function that will be run on a device given some inputs.
func (g *State) Compile(dev *api.Device, out Element) (*CompiledGraph, error) {
	paramShapes := make([]*shape.Shape, len(g.args))
	for i, arg := range g.args {
		paramShapes[i] = arg.Shape()
	}
	cg := &CompiledGraph{
		state:  g,
		device: dev,
		out:    out,
	}
	// Flatten out and get all the backend graph nodes from the list.
	graphOutNodes, err := ExtractGraphNodes(out)
	if err != nil {
		return nil, err
	}
	graphAuxNodes, err := ExtractGraphNodes(g.traces.asTuple())
	if err != nil {
		return nil, err
	}
	if len(graphOutNodes) == 0 && len(graphAuxNodes) == 0 {
		// Nothing to run in the graph.
		// We do not need to compile and we leave cg.runner to nil.
		return cg, nil
	}
	cg.runner, err = g.backendGraph.Compile(dev.PlatformDevice(), graphOutNodes, graphAuxNodes, paramShapes)
	return cg, err
}

func (g *CompiledGraph) run(ctx *elements.CallInputs) (out, traces []platform.DeviceHandle, err error) {
	if g.runner == nil {
		// No runner: there is nothing to run in the graph.
		return nil, nil, nil
	}
	handles := make([]platform.Handle, len(g.state.args))
	for i, arg := range g.state.args {
		var err error
		handles[i], err = arg.ToDeviceHandle(g.device, ctx)
		if err != nil {
			return nil, nil, err
		}
	}
	return g.runner.Run(handles)
}

// Run the graph.
func (g *CompiledGraph) Run(receiver values.Value, args []values.Value, tracer Tracer) ([]values.Value, error) {
	fc := &elements.CallInputs{Receiver: receiver, Args: args}
	for _, init := range g.state.inits {
		if err := init.Init(fc); err != nil {
			return nil, err
		}
	}
	out, traced, err := g.run(fc)
	if err != nil {
		return nil, err
	}
	if err := g.state.traces.process(g.device, fc, tracer, traced); err != nil {
		return nil, err
	}
	return g.handlesToValues(fc, out)
}

func (g *CompiledGraph) handlesToValues(ctx *elements.CallInputs, handles []platform.DeviceHandle) ([]values.Value, error) {
	reader := elements.NewUnflattener(g.device, ctx, handles)
	els := []Element{g.out}
	if g.state.fn.FuncType().Results.Len() > 1 {
		els = g.out.(*elements.Tuple).Elements()
	}
	values := make([]values.Value, len(els))
	for i, el := range els {
		val, err := reader.Unflatten(el)
		if err != nil {
			return nil, err
		}
		values[i] = val
	}
	return values, nil
}
