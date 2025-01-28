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
	"github.com/gx-org/backend/dtype"
	"github.com/gx-org/backend/platform"
	"github.com/gx-org/backend/shape"
	"github.com/gx-org/gx/golang/backend/kernels"
)

// Device is a device for the Go backend, that is a CPU.
type Device struct {
	plat *Platform
	ord  int
}

func newDevice(plat *Platform, ord int) *Device {
	return &Device{plat: plat}
}

// Platform owning the device.
func (dev *Device) Platform() platform.Platform {
	return dev.plat
}

// Ordinal of the device on the platform.
func (dev *Device) Ordinal() int {
	return dev.ord
}

// Send raw data to the device. Return a handle from this package.
func (dev *Device) send(data []byte, sh *shape.Shape) (*Handle, error) {
	if sh.DType == dtype.Invalid {
		return nil, errors.Errorf("GX %s data type not supported", sh.DType.String())
	}
	array, err := kernels.NewArrayFromRaw(data, sh)
	if err != nil {
		return nil, err
	}
	return NewDeviceHandle(dev, array), nil
}

func (dev *Device) sendFromHost(handle platform.HostBuffer) (*Handle, error) {
	data := handle.Acquire()
	defer handle.Release()
	return dev.send(data, handle.Shape())
}

// Send raw data to the device.
func (dev *Device) Send(buf []byte, sh *shape.Shape) (platform.DeviceHandle, error) {
	return dev.send(buf, sh)
}
