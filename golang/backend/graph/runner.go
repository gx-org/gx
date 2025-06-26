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
	"github.com/gx-org/backend/ops"
	"github.com/gx-org/backend/platform"
	"github.com/gx-org/backend/shape"
	"github.com/gx-org/gx/golang/backend/kernels"
	goplatform "github.com/gx-org/gx/golang/backend/platform"
)

type (
	nodeRunner struct {
		graph  *Graph
		device *goplatform.Device
		output []execNode
		traced []execNode
	}

	execNode interface {
		ops.Node
		shape() *shape.Shape
		kernelFactory() kernels.Factory
		exec(*executor) (kernels.Array, error)
	}

	executor struct {
		runner *nodeRunner
		args   []platform.Handle
	}
)

func toExecNodes(outs []*ops.OutputNode) ([]execNode, error) {
	execs := make([]execNode, len(outs))
	for i, out := range outs {
		var ok bool
		execs[i], ok = out.Node.(execNode)
		if !ok {
			return nil, errors.Errorf("node of type %T not supported by the backend", out)
		}
	}
	return execs, nil
}

// Compile the graph for a given device.
// The graph is not supposed to be modified once it has been compiled.
func (g *Graph) Compile(dev platform.Device, output, traced []*ops.OutputNode, params []*shape.Shape) (ops.Runner, error) {
	nr := &nodeRunner{
		graph:  g,
		device: dev.(*goplatform.Device),
	}
	var err error
	nr.output, err = toExecNodes(output)
	if err != nil {
		return nil, err
	}
	nr.traced, err = toExecNodes(traced)
	if err != nil {
		return nil, err
	}
	return nr, nil
}

func (r *nodeRunner) runAll(exec *executor, nodes []execNode) ([]platform.DeviceHandle, error) {
	handles := make([]platform.DeviceHandle, len(nodes))
	for i, node := range nodes {
		kVal, err := node.exec(exec)
		if err != nil {
			return nil, err
		}
		handles[i] = goplatform.FromValue(r.device, kVal)
	}
	return handles, nil
}

func (r *nodeRunner) Run(args []platform.Handle) (out, traced []platform.DeviceHandle, err error) {
	exec := &executor{
		runner: r,
		args:   args,
	}
	out, err = r.runAll(exec, r.output)
	if err != nil {
		return nil, nil, err
	}
	traced, err = r.runAll(exec, r.traced)
	if err != nil {
		return nil, nil, err
	}
	return
}
