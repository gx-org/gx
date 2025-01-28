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
	"github.com/gx-org/gx/build/builder"
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

// Device returns a new device on the platform.
func (rtm *Runtime) Device(ord int) (*Device, error) {
	dev, err := rtm.bck.Platform().Device(ord)
	if err != nil {
		return nil, err
	}
	return &Device{rtm: rtm, dev: dev}, nil
}

// Builder returns the builder used to build GX source code into a GX intermediate representation.
func (rtm *Runtime) Builder() *builder.Builder {
	return rtm.builder
}

type (
	// PackageOption is an option specific to a package.
	PackageOption interface {
		Package() string
	}

	// Device created by a runtime.
	Device struct {
		rtm *Runtime
		dev platform.Device
	}
)

// Runtime returns the device's parent runtime.
func (dev *Device) Runtime() *Runtime {
	return dev.rtm
}

// PlatformDevice returns the device used on the platform.
func (dev *Device) PlatformDevice() platform.Device {
	return dev.dev
}
