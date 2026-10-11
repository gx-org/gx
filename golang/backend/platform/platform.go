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

// Package platform implements a GX platform for the native Go backend.
package platform

import (
	"fmt"
	"slices"

	"github.com/gomlx/compute"
	"github.com/gomlx/compute/dtypes"
	"github.com/gomlx/compute/shapes"
	"github.com/gx-org/gx/golang/backend/kernels"
)

// Platform implements a native platform in Go.
type Platform struct {
	bck       compute.Backend
	finalized bool
}

// New returns a Go native platform.
func New(bck compute.Backend) *Platform {
	return &Platform{bck: bck}
}

// Backend returns the backend owning the platform.
func (plat *Platform) Backend() compute.Backend {
	return plat.bck
}

// Name of the backend.
func (plat *Platform) Name() string {
	return "gonative"
}

// String returns the same as Name.
func (plat *Platform) String() string {
	return plat.Name()
}

// Description is a longer description of the Backend that can be used to pretty-print.
func (plat *Platform) Description() string {
	return "Go Native Backend"
}

// NumDevices returns the number of devices available for this Backend.
func (plat *Platform) NumDevices() int {
	return 1
}

// DeviceDescription returns a description of the device at the given deviceNum.
func (plat *Platform) DeviceDescription(deviceNum compute.DeviceNum) string {
	return fmt.Sprintf("Go CPU device #%d", deviceNum)
}

// Capabilities returns information about what is supported by this backend.
func (plat *Platform) Capabilities() compute.Capabilities {
	return compute.Capabilities{}
}

// BufferFromFlatData transfers data from Go given as a flat slice to the deviceNum, and returns the corresponding Buffer.
func (plat *Platform) BufferFromFlatData(deviceNum compute.DeviceNum, flat any, shape shapes.Shape) (compute.Buffer, error) {
	data := slices.Clone(dtypes.UnsafeByteSliceFromAny(flat))
	return plat.send(deviceNum, data, shape)
}

// HasSharedBuffers returns whether the backend supports shared buffers.
func (plat *Platform) HasSharedBuffers() bool {
	return true
}

// NewSharedBuffer returns a shared buffer that can be both used as input for execution of computations and directly read or mutated by the clients.
func (plat *Platform) NewSharedBuffer(deviceNum compute.DeviceNum, shape shapes.Shape) (buffer compute.Buffer, flat any, err error) {
	arr, err := kernels.Zero(shape)
	if err != nil {
		return nil, nil, err
	}
	h := NewDeviceHandle(plat, deviceNum, arr)
	flat, err = h.Data()
	if err != nil {
		return nil, nil, err
	}
	return h, flat, nil
}

// Finalize releases all the associated resources immediately, and makes the backend invalid.
func (plat *Platform) Finalize() {
	plat.finalized = true
}

// IsFinalized returns true if the backend is in an invalid state.
func (plat *Platform) IsFinalized() bool {
	return plat == nil || plat.finalized
}

// FromValue returns a handle on a value.
func FromValue(plat *Platform, dev compute.DeviceNum, x kernels.Array) compute.Buffer {
	return NewDeviceHandle(plat, dev, x)
}
