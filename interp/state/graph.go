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
	"github.com/gx-org/gx/api/values"
	"github.com/gx-org/gx/build/ir"
)

type (
	deviceHandler interface {
		deviceHandle() values.Value
	}

	// Initializer is called at the beginning of a run before
	// arguments for the backend are computed.
	Initializer interface {
		Init(Context) error
	}

	// Argument provides an argument to pass to the backend.
	Argument interface {
		Shape() *shape.Shape

		ToDeviceHandle(platform.Device, Context) (platform.DeviceHandle, error)
	}
	// Tracer receives traced values from a run.
	Tracer interface {
		Trace(fset *token.FileSet, call *ir.CallExpr, values []values.Value) error
	}

	// State of a collection of nodes.
	State struct {
		backendGraph graph.Graph
		fn           *ir.FuncDecl
		inits        []Initializer
		args         []Argument

		tracer nodeTracer
	}

	// CompiledGraph is a graph compiled for a device.
	CompiledGraph struct {
		device platform.Device
		state  *State
		runner graph.Runner
		out    Element
	}

	// Element in the state.
	Element interface {
		State() *State
	}

	// backendElement is an element that can be converted into one or more graph nodes.
	backendElement interface {
		Element

		nodes() ([]*graph.OutputNode, error)
	}

	// Copyable is an interface implemented by nodes that need to be copied when passed to a function.
	Copyable interface {
		Copy() Element
	}

	// FieldSelector selects a field given its index.
	FieldSelector interface {
		SelectField(ExprAt, int) (Element, error)
	}

	// Slicer is a state element that can be sliced.
	Slicer interface {
		Slice(ExprAt, int) (Element, error)
	}

	// ArraySlicer is a state element with an array that can be sliced.
	ArraySlicer interface {
		NumericalElement
		Slice(ExprAt, int) (Element, error)
	}

	// MethodSelector selects a method given its index.
	MethodSelector interface {
		SelectMethod(ir.Func) (*Func, error)
	}

	emptyRunner struct{}
)

// New XLA computation graph.
func New(fn *ir.FuncDecl, backendGraph graph.Graph) *State {
	return &State{fn: fn, backendGraph: backendGraph}
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

// Compile a node given a set of parameters and using this node as an output.
// Returns a function that will be run on a device given some inputs.
func (g *State) Compile(dev platform.Device, out Element) (*CompiledGraph, error) {
	paramShapes := make([]*shape.Shape, len(g.args))
	for i, arg := range g.args {
		paramShapes[i] = arg.Shape()
	}
	outNodes, err := OutputsFromElement(out)
	if err != nil {
		return nil, err
	}
	cg := &CompiledGraph{
		state:  g,
		device: dev,
		out:    out,
	}
	if len(outNodes) == 0 {
		// Nothing to run in the graph.
		// We do not need to compile and we leave cg.runner to nil.
		return cg, nil
	}
	cg.runner, err = g.backendGraph.Compile(dev, outNodes, g.tracer.nodes, paramShapes)
	return cg, err
}

type functionCall struct {
	receiver values.Value
	args     []values.Value
}

func (fc functionCall) Receiver() values.Value {
	return fc.receiver
}

func (fc functionCall) Args() []values.Value {
	return fc.args
}

// Run the graph.
func (g *CompiledGraph) Run(receiver values.Value, args []values.Value, tracer Tracer) ([]values.Value, error) {
	fc := functionCall{receiver: receiver, args: args}
	for _, init := range g.state.inits {
		if err := init.Init(fc); err != nil {
			return nil, err
		}
	}
	if g.runner == nil {
		// No runner: there is nothing to run in the graph.
		return g.handlesToValues(nil)
	}
	handles := make([]platform.Handle, len(g.state.args))
	for i, arg := range g.state.args {
		var err error
		handles[i], err = arg.ToDeviceHandle(g.device, fc)
		if err != nil {
			return nil, err
		}
	}
	out, traced, err := g.runner.Run(handles)
	if err != nil {
		return nil, err
	}
	if err := g.state.tracer.process(tracer, traced); err != nil {
		return nil, err
	}
	return g.handlesToValues(out)
}

func (g *CompiledGraph) handlesToValues(out []platform.DeviceHandle) ([]values.Value, error) {
	reader := &handleParser{handles: out}
	elements := []Element{g.out}
	if g.state.fn.FType.Results.Len() > 1 {
		elements = g.out.(*Tuple).Unpack()
	}
	values := make([]values.Value, len(elements))
	for i, el := range elements {
		val, err := reader.parse(el)
		if err != nil {
			return nil, err
		}
		values[i] = val
	}
	return values, nil
}
