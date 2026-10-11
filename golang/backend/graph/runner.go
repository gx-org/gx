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

package graph

import (
	"github.com/pkg/errors"
	"github.com/gomlx/compute"
	"github.com/gomlx/compute/shapes"
	"github.com/gx-org/gx/golang/backend/kernels"
	goplatform "github.com/gx-org/gx/golang/backend/platform"
)

type (
	nodeRunner struct {
		graph  *Graph
		device compute.DeviceNum
		output []execNode
	}

	execNode interface {
		shape() shapes.Shape
		kernelFactory() kernels.Factory
		exec(*executor) (kernels.Array, error)
	}

	executor struct {
		runner *nodeRunner
		args   []compute.Buffer
	}
)

var _ compute.Executable = (*nodeRunner)(nil)

func toExecNodes(outs []compute.Value) ([]execNode, error) {
	execs := make([]execNode, len(outs))
	for i, out := range outs {
		var ok bool
		execs[i], ok = out.(execNode)
		if !ok {
			return nil, errors.Errorf("node of type %T not supported by the backend", out)
		}
	}
	return execs, nil
}

// Compile the graph for a given device.
// The graph is not supposed to be modified once it has been compiled.
func (g *Graph) Compile(dev compute.DeviceNum) (compute.Executable, error) {
	if !g.returned {
		return nil, errors.Errorf("Return() was not called for function %q before Compile()", g.funcName)
	}
	nr := &nodeRunner{
		graph:  g,
		device: dev,
	}
	var err error
	nr.output, err = toExecNodes(g.outputs)
	if err != nil {
		return nil, err
	}
	return nr, nil
}

func (r *nodeRunner) Finalize() {}

func (r *nodeRunner) Inputs() (names []string, inputShapes []shapes.Shape) {
	names = make([]string, len(r.graph.args))
	inputShapes = make([]shapes.Shape, len(r.graph.args))
	for i, arg := range r.graph.args {
		names[i] = arg.name
		inputShapes[i] = arg.shape()
	}
	return names, inputShapes
}

func (r *nodeRunner) Outputs() (outputShapes []shapes.Shape) {
	outputShapes = make([]shapes.Shape, len(r.output))
	for i, out := range r.output {
		outputShapes[i] = out.shape()
	}
	return outputShapes
}

func (r *nodeRunner) runAll(exec *executor, nodes []execNode) ([]compute.Buffer, error) {
	handles := make([]compute.Buffer, len(nodes))
	for i, node := range nodes {
		kVal, err := node.exec(exec)
		if err != nil {
			return nil, err
		}
		handles[i] = goplatform.FromValue(r.graph.plat, r.device, kVal)
	}
	return handles, nil
}

func (r *nodeRunner) Execute(inputs []compute.Buffer, donate []bool, defaultDevice compute.DeviceNum) ([]compute.Buffer, error) {
	exec := &executor{
		runner: r,
		args:   inputs,
	}
	return r.runAll(exec, r.output)
}
