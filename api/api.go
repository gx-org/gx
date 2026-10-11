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
	"unsafe"

	"github.com/pkg/errors"
	"github.com/gomlx/compute"
	"github.com/gomlx/compute/dtypes"
	"github.com/gomlx/compute/shapes"
	"github.com/gx-org/gx/build/builder"
)

// Runtime encapsulates a GX builder and a backend.
// A backend is a platform managing one or more devices
// and a graph builder.
type Runtime struct {
	bck     compute.Backend
	builder *builder.Builder
}

// NewRuntime returns a new GX runtime.
func NewRuntime(bck compute.Backend, bld *builder.Builder) *Runtime {
	return &Runtime{bck: bck, builder: bld}
}

// Backend used by the runtime.
func (rtm *Runtime) Backend() compute.Backend {
	return rtm.bck
}

// Device returns a new device on the platform.
func (rtm *Runtime) Device(ord int) (*Device, error) {
	return &Device{rtm: rtm, dev: compute.DeviceNum(ord)}, nil
}

// Builder returns the builder used to build GX source code into a GX intermediate representation.
func (rtm *Runtime) Builder() *builder.Builder {
	return rtm.builder
}

// Device created by a runtime.
type Device struct {
	rtm *Runtime
	dev compute.DeviceNum
}

// Runtime returns the device's parent runtime.
func (dev *Device) Runtime() *Runtime {
	return dev.rtm
}

// PlatformDevice returns the device used on the backend.
func (dev *Device) PlatformDevice() compute.DeviceNum {
	return dev.dev
}

// Backend returns the backend of the runtime.
func (dev *Device) Backend() compute.Backend {
	return dev.rtm.bck
}

// Send sends a buffer of data to the device.
func (dev *Device) Send(buf []byte, sh shapes.Shape) (compute.Buffer, error) {
	if sh.DType == dtypes.InvalidDType {
		return nil, errors.Errorf("GX invalid data type not supported")
	}
	if int64(len(buf)) != sh.ByteSize() {
		return nil, errors.Errorf("buffer size is %d but shape specify a buffer size of %d", len(buf), sh.ByteSize())
	}
	flat := dtypes.UnsafeAnySliceFromBytes(unsafe.Pointer(unsafe.SliceData(buf)), sh.DType, sh.Size())
	return dev.Backend().BufferFromFlatData(dev.dev, flat, sh)
}
