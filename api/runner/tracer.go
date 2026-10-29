// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package runner

import (
	"fmt"

	"github.com/gx-org/gx/api"
	"github.com/gx-org/gx/api/hostio"
	"github.com/gx-org/gx/api/options"
	"github.com/gx-org/gx/api/trace"
	"github.com/gx-org/gx/build/fmterr"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/golang/backend/kernels"
	"github.com/gx-org/gx/internal/togo"
	"github.com/gx-org/gx/internal/tracer/cfunc"
	"github.com/gx-org/gx/internal/tracer/processor"
	"github.com/gx-org/gx/interp/grapheval"
	"github.com/gx-org/gx/interp"
	"github.com/gx-org/gx/interp/materialise"
)

// CompiledFunc is a function compiled for a given receiver and argument shapes
// which is ready to run.
type CompiledFunc interface {
	// Run the function for the given GX values.
	Run(receiver hostio.Value, args []hostio.Value, tracer trace.Callback) ([]hostio.Value, error)
}

// EagerBackend is an interface implemented by backend only supporting eager mode.
type EagerBackend interface {
	Eager()
}

// Trace a function and returns a runner to run the function on the device.
func Trace(dev *api.Device, fn *ir.FuncDecl, receiver hostio.Value, args []hostio.Value, options []options.PackageOption) (_ CompiledFunc, err error) {
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
	proc := processor.New()
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
	outs, err := itp.EvalFunc(fn, in)
	if err != nil {
		return nil, err
	}
	// Compile the resulting graph given the output element.
	return cfunc.Compile(dev, fn, proc, mat, builder, outs)
}

// TraceAndRun traces fn, runs it on dev, transfers its outputs to the host,
// and releases the device buffers of the outputs.
func TraceAndRun(dev *api.Device, fn *ir.FuncDecl, recv hostio.Value, args []hostio.Value, options []options.PackageOption, tracer trace.Callback) ([]hostio.Value, error) {
	compiled, err := Trace(dev, fn, recv, args, options)
	if err != nil {
		return nil, err
	}
	outs, err := compiled.Run(recv, args, tracer)
	if err != nil {
		return nil, err
	}
	defer hostio.Finalize(outs...)
	return hostio.ToHost(kernels.DefaultAllocator(), outs)
}
