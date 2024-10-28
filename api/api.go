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

// Package api defines the interface between a backend and a GX package.
package api

import (
	"github.com/gx-org/backend"
	"github.com/gx-org/backend/platform"
	"github.com/gx-org/gx/api/values"
	"github.com/gx-org/gx/build/builder"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/interp"
	"github.com/gx-org/gx/interp/state"
)

// Runtime encapsulates a GX builder and a backend.
// A backend is a platform managing one or more devices
// and a graph builder.
type Runtime struct {
	bck     backend.Backend
	builder *builder.Builder
}

// NewRuntime returns a new GX runtime.
func NewRuntime(bck backend.Backend, bld *builder.Builder) *Runtime {
	return &Runtime{bck: bck, builder: bld}
}

// Backend used by the runtime.
func (rtm *Runtime) Backend() backend.Backend {
	return rtm.bck
}

// Platform used by the runtime.
func (rtm *Runtime) Platform() platform.Platform {
	return rtm.bck.Platform()
}

// Builder returns the builder used to build GX source code into a GX intermediate representation.
func (rtm *Runtime) Builder() *builder.Builder {
	return rtm.builder
}

// Compile a function and returns a runner to run the function on the device.
func (rtm *Runtime) Compile(dev platform.Device, funcDecl *ir.FuncDecl, receiver values.Value, args []values.Value, options []interp.PackageOption) (*state.CompiledGraph, error) {
	itrp, err := interp.New(rtm.bck, options)
	if err != nil {
		return nil, err
	}
	graph, outNode, err := itrp.Eval(funcDecl, receiver, args)
	if err != nil {
		return nil, err
	}
	return graph.Compile(dev, outNode)
}
