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

package eagereval

import (
	"fmt"

	"github.com/gx-org/gx/api"
	"github.com/gx-org/gx/api/hostio"
	"github.com/gx-org/gx/api/options"
	"github.com/gx-org/gx/build/fmterr"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/base/cast"
	"github.com/gx-org/gx/interp"
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
	bck, err := cast.To[EagerBackend](dev.Backend())
	if err != nil {
		return nil, err
	}
	// Create an evaluator for the interpreter executing operations eagerly on the backend.
	builder := dev.Runtime().Backend().Builder(fn.FullyQualifiedName())
	reg, ops := newRegistry(bck, dev.Runtime().Builder(), builder.Main())
	// Create the interpreter.
	itp, err := interp.New(reg, options)
	if err != nil {
		return nil, err
	}
	fitp, err := itp.ForFile(fn.File())
	if err != nil {
		return nil, err
	}
	// Convert the receiver and arguments values into elements for the interpreter.
	in, err := ops.funcInputsToElements(fitp, fn, receiver, args)
	if err != nil {
		return nil, err
	}
	// Interpret the function: the backend computes all the operations eagerly.
	outs, err := itp.EvalFunc(fn, in)
	if err != nil {
		return nil, err
	}
	return ops.Values(outs)
}
