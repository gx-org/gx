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

// Package types defines types used in the Go bindings.
package types

import (
	"fmt"

	"github.com/gx-org/backend/dtype"
	"github.com/gx-org/backend/shape"
	"github.com/gx-org/gx/api"
	"github.com/gx-org/gx/api/values"
	"github.com/gx-org/gx/build/ir"
)

type (
	// Bridger is able to return a Go-GX value bridge.
	Bridger interface {
		Bridge() Bridge
	}

	// Bridge is implemented by all types used by the Go bindings.
	// It provides the bridge between the Go instance and the GX value.
	Bridge interface {
		values.Valuer

		// Bridger returns the Go object owning the GX value.
		Bridger() Bridger
	}

	// ArrayBridge is a bridge value representing an array (or an atomic value).
	ArrayBridge interface {
		Bridge

		// Shape returns the shape of the array.
		Shape() *shape.Shape

		// ToDevice transfers the array to a device.
		ToDevice(*api.Device) (ArrayBridge, error)
	}

	// Factory creates instances given a type.
	// Numerical types (atoms or arrays) are not supported.
	Factory interface {
		New(ir.Type) (Bridge, error)
	}

	// StructBridge is a bridge for a structure.
	StructBridge interface {
		Bridge

		// StructValue returns the GX value of the structure.
		StructValue() *values.Struct

		// NewFromField returns a new instance of the type defined by the field of the structure.
		// An error is returned if the field does not belong to the structure.
		NewFromField(field *ir.Field) (Bridge, error)

		// SetField sets a field with a value.
		// An error is returns if the value cannot be casted to the field type.
		SetField(field *ir.Field, val Bridge) error
	}

	arrayBridger interface {
		Bridger

		toDeviceBridger(*values.DeviceArray) ArrayBridge
	}

	baseBridge[B arrayBridger, V values.Array] struct {
		owner B
		value V
	}
)

var _ ArrayBridge = (*baseBridge[arrayBridger, values.Array])(nil)

func newBaseBridge[B arrayBridger, V values.Array](owner B, value V) baseBridge[B, V] {
	return baseBridge[B, V]{owner: owner, value: value}
}

// GXValue returns the GX value storing the array data.
func (bb *baseBridge[B, V]) GXValue() values.Value {
	return bb.value
}

// Bridge between the Go value and the GX value.
func (bb *baseBridge[B, V]) Bridge() Bridge {
	return bb
}

// Shape of the array.
func (bb *baseBridge[B, V]) Shape() *shape.Shape {
	return bb.value.Shape()
}

// ToDevice transfers the array to a device.
func (bb *baseBridge[B, V]) ToDevice(dev *api.Device) (ArrayBridge, error) {
	value, err := bb.value.ToDevice(dev.PlatformDevice())
	if err != nil {
		return nil, err
	}
	return bb.owner.toDeviceBridger(value), nil
}

// Bridger returns the Go value of the object.
func (bb *baseBridge[B, V]) Bridger() Bridger {
	return bb.owner
}

// NewAtom returns a new Go atom bridge given a GX value of the same data type.
func NewAtom[T dtype.GoDataType](arr values.Array) Atom[T] {
	hostValue, ok := arr.(*values.HostArray)
	if ok {
		return NewHostAtom[T](hostValue)
	}
	deviceValue, ok := arr.(*values.DeviceArray)
	if ok {
		return NewDeviceAtom[T](deviceValue)
	}
	panic(fmt.Sprintf("cannot cast %T to a host or device GX array value", arr))
}

// NewArray returns a new Go array bridge given a GX value of the same data type.
func NewArray[T dtype.GoDataType](arr values.Array) Array[T] {
	hostValue, ok := arr.(*values.HostArray)
	if ok {
		return NewHostArray[T](hostValue)
	}
	deviceValue, ok := arr.(*values.DeviceArray)
	if ok {
		return NewDeviceArray[T](deviceValue)
	}
	panic(fmt.Sprintf("cannot cast %T to a host or device GX array value", arr))
}
