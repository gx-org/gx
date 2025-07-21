// Copyright 2025 Google LLC
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

// Package processor stores processing that is required before and after
// a compiled function is called.
package processor

import (
	"github.com/gx-org/backend/platform"
	"github.com/gx-org/backend/shape"
	"github.com/gx-org/gx/api/values"
	"github.com/gx-org/gx/build/ir"
)

type (

	// Processor stores initializers and traces to process before and
	// after calling a compiled function.
	Processor struct {
		traces
		inits []Initializer
		args  []Argument
	}

	// Initializer is called at the beginning of a run before
	// arguments for the backend are computed.
	Initializer interface {
		Init(*values.FuncInputs) error
	}

	// Argument provides an argument to pass to the backend.
	Argument interface {
		ir.Element

		Shape() *shape.Shape

		ToDeviceHandle(platform.Device, *values.FuncInputs) (platform.DeviceHandle, error)
	}
)

// RegisterInit registers an Initializer to the graph.
func (p *Processor) RegisterInit(init Initializer) {
	p.inits = append(p.inits, init)
}

// RegisterArg an argument for the backend.
// Returns the index of the argument.
func (p *Processor) RegisterArg(arg Argument) int {
	index := len(p.args)
	p.args = append(p.args, arg)
	return index
}

// Inits returns all the initializers.
func (p *Processor) Inits() []Initializer {
	return p.inits
}

// Args returns the graph arguments.
func (p *Processor) Args() []Argument {
	return p.args
}

// ElementArgs returns the graph arguments as interpreter elements.
func (p *Processor) ElementArgs() []ir.Element {
	els := make([]ir.Element, len(p.args))
	for i, arg := range p.args {
		els[i] = arg
	}
	return els
}

// ProcessInits calls all the initializers callbacks.
func (p *Processor) ProcessInits(fc *values.FuncInputs) error {
	for _, init := range p.inits {
		if err := init.Init(fc); err != nil {
			return err
		}
	}
	return nil
}
