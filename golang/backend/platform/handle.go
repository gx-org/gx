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

package platform

import (
	"unsafe"

	"github.com/pkg/errors"
	"github.com/gomlx/compute"
	"github.com/gomlx/compute/dtypes"
	"github.com/gomlx/compute/shapes"
	"github.com/gx-org/gx/golang/backend/kernels"
)

// Handle to a value stored by the backend.
type Handle struct {
	plat   *Platform
	device compute.DeviceNum
	array  kernels.Array
}

var _ compute.Buffer = (*Handle)(nil)

// NewDeviceHandle returns a new device handle given an array and a device.
func NewDeviceHandle(plat *Platform, dev compute.DeviceNum, array kernels.Array) *Handle {
	return &Handle{plat: plat, device: dev, array: array}
}

// Backend returns the backend that owns this buffer.
func (h *Handle) Backend() compute.Backend {
	return h.plat.Backend()
}

// Finalize allows the client to inform the backend that the buffer is no longer needed.
func (h *Handle) Finalize() error {
	return nil
}

// DeviceNum returns the deviceNum for the buffer.
func (h *Handle) DeviceNum() (compute.DeviceNum, error) {
	return h.device, nil
}

// Shape returns the shape for the buffer.
func (h *Handle) Shape() (shapes.Shape, error) {
	return h.array.Shape(), nil
}

// ToFlatData transfers the flat values of the buffer to the Go flat array.
func (h *Handle) ToFlatData(flat any) error {
	dst := dtypes.UnsafeByteSliceFromAny(flat)
	src := h.array.Buffer()
	if len(dst) != len(src) {
		return errors.Errorf("cannot copy buffer of %d bytes into flat slice of %d bytes", len(src), len(dst))
	}
	copy(dst, src)
	return nil
}

// Data returns a slice pointing to the buffer storage memory directly.
func (h *Handle) Data() (flat any, err error) {
	buf := h.array.Buffer()
	sh := h.array.Shape()
	return dtypes.UnsafeAnySliceFromBytes(unsafe.Pointer(unsafe.SliceData(buf)), sh.DType, sh.Size()), nil
}

// CopyToDevice copies the buffer to another device on the same backend.
func (h *Handle) CopyToDevice(dev compute.DeviceNum) (compute.Buffer, error) {
	return h.toDevice(dev)
}

func (h *Handle) toDevice(dev compute.DeviceNum) (*Handle, error) {
	if h.device == dev {
		return h, nil
	}
	return h.plat.send(dev, h.array.Buffer(), h.array.Shape())
}

// Array returns the underlying array storing the data for the handle.
func (h *Handle) Array() kernels.Array {
	return h.array
}

// ToDevice sends a generic buffer to a device.
func ToDevice(plat *Platform, dev compute.DeviceNum, handle compute.Buffer) (*Handle, error) {
	if handleT, ok := handle.(*Handle); ok && handleT.plat == plat {
		return handleT.toDevice(dev)
	}
	sh, err := handle.Shape()
	if err != nil {
		return nil, err
	}
	raw := make([]byte, sh.ByteSize())
	flat := dtypes.UnsafeAnySliceFromBytes(unsafe.Pointer(unsafe.SliceData(raw)), sh.DType, sh.Size())
	if err := handle.ToFlatData(flat); err != nil {
		return nil, err
	}
	return plat.send(dev, raw, sh)
}
