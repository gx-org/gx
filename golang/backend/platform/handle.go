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
	"github.com/pkg/errors"
	"github.com/gx-org/backend/platform"
	"github.com/gx-org/backend/shape"
	"github.com/gx-org/gx/golang/backend/kernels"
)

// Handle to a value stored by the backend.
type Handle struct {
	device *Device
	array  kernels.Array
}

// NewDeviceHandle returns a new device handle given an array and a device.
func NewDeviceHandle(dev *Device, array kernels.Array) *Handle {
	return &Handle{device: dev, array: array}
}

// Platform owning the handle.
func (h *Handle) Platform() platform.Platform {
	return h.device.plat
}

// Device on which the handle is located.
func (h *Handle) Device() platform.Device {
	return h.device
}

// Shape of the data stored by the handle.
func (h *Handle) Shape() *shape.Shape {
	return h.array.Shape()
}

// ToHost fetches the data from the handle and write it to buffer.
func (h *Handle) ToHost(buf platform.HostBuffer) error {
	data := buf.Acquire()
	defer buf.Release()
	copy(data, h.array.Buffer())
	return nil
}

// ToDevice transfers the handle to a device.
func (h *Handle) ToDevice(dev platform.Device) (platform.DeviceHandle, error) {
	goDev, ok := dev.(*Device)
	if ok {
		return ToDevice(goDev, h)
	}
	return nil, errors.Errorf("not implemented")
}

func (h *Handle) toDevice(dev *Device) (*Handle, error) {
	if h.device == dev {
		return h, nil
	}
	return dev.send(h.array.Buffer(), h.Shape())
}

// Array returns the underlying array storing the data for the handle.
func (h *Handle) Array() kernels.Array {
	return h.array
}

// ToDevice sends a generic handle to a device.
func ToDevice(dev *Device, handle platform.Handle) (*Handle, error) {
	switch handleT := handle.(type) {
	case *Handle:
		return handleT.toDevice(dev)
	case platform.HostBuffer:
		return dev.sendFromHost(handleT)
	}
	return nil, errors.Errorf("cross-backend transfers not implemented")
}
