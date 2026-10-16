// Copyright 2026 Google LLC
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

// Package runner runs GX function, mostly in two different mode.
// First, as a trace mode where GX is executed to build a graph which will then be compiled a run.
// Second, as a run mode where GX is executed like other interpreted language.
package runner

import (
	"fmt"

	"github.com/gx-org/gx/api"
	"github.com/gx-org/gx/api/hostio"
	"github.com/gx-org/gx/api/options"
	"github.com/gx-org/gx/api/values"
	"github.com/gx-org/gx/build/fmterr"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/base/cast"
	"github.com/gx-org/gx/internal/togo"
	"github.com/gx-org/gx/internal/tracer/processor"
	"github.com/gx-org/gx/interp/grapheval"
	"github.com/gx-org/gx/interp"
	"github.com/gx-org/gx/interp/materialise"
)

// Run a function on the device.
func Run(dev *api.Device, fn *ir.FuncDecl, receiver hostio.Value, args []hostio.Value, options []options.PackageOption) (_ []hostio.Value, err error) {
	defer func() {
		if err != nil {
			recvName := ""
			if recv := fn.FType.ReceiverField(); recv != nil {
				recvName = recv.Name.Name + "."
			}
			err = fmt.Errorf("%s.%s%s evaluation error:\n%w", fn.File().Package.Path(), recvName, fn.Name(), err)
		}
		err = fmterr.ToStackTraceError(err)
	}()
	// Create a new graph and evaluator for the interpreter.
	proc := processor.New(values.Factory())
	fc := &togo.FuncInputs{Receiver: receiver, Args: hostio.ToElements(args)}
	proc.ProcessInits(fc)
	builder := dev.Runtime().Backend().Builder(fn.FullyQualifiedName())
	reg, mat := grapheval.New(dev.Runtime().Builder(), proc, builder.Main())
	// Create the interpreter.
	itp, err := interp.New(reg, options)
	if err != nil {
		return nil, err
	}
	fitp, err := itp.ForFile(fn.File())
	if err != nil {
		return nil, err
	}
	// Visit the receiver and arguments values to create elements for the interpreter.
	in, err := mat.FuncInputsToElements(fitp, fn, receiver, args)
	if err != nil {
		return nil, err
	}
	// Add all the arguments to the graph.
	if _, err := materialise.All(mat, proc.ElementArgs()); err != nil {
		return nil, err
	}
	// Interpret the function with the evaluator to build the graph.
	// The evaluation returns a single output element.
	outEls, err := itp.EvalFunc(fn, in)
	if err != nil {
		return nil, err
	}
	outs := make([]hostio.Value, len(outEls))
	for i, el := range outEls {
		var val any = el
		// TODO(degris): create an eagereval to replace grapheval to avoid the following if statement.
		if backendNode, isBackendNode := el.(*grapheval.BackendNode); isBackendNode {
			val = backendNode.Value()
		}
		outs[i], err = cast.To[hostio.Value](val)
		if err != nil {
			return nil, err
		}
	}
	return outs, nil
}
