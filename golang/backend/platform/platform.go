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
	"github.com/gx-org/backend/platform"
	"github.com/gx-org/gx/golang/backend/kernels"
)

// Platform implements a native platform in Go.
type Platform struct {
	dev *Device
}

// New returns a Go native platform.
func New() *Platform {
	plat := &Platform{}
	plat.dev = newDevice(plat, 0)
	return plat
}

// Name of the platform.
func (plat *Platform) Name() string {
	return "gonative"
}

// GoDevice returns a Go CPU device.
func (plat *Platform) GoDevice(ordinal int) (*Device, error) {
	return plat.dev, nil
}

// Device returns the CPU device.
func (plat *Platform) Device(ordinal int) (platform.Device, error) {
	return plat.GoDevice(ordinal)
}

// FromValue returns a handle on a value.
func FromValue(dev platform.Device, x kernels.Array) platform.DeviceHandle {
	return &Handle{
		device: dev.(*Device),
		array:  x,
	}
}
