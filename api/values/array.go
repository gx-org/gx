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

package values

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/pkg/errors"
	"github.com/gx-org/backend/dtypes"
	"github.com/gx-org/backend/platform"
	"github.com/gx-org/backend/shape"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/golang/backend/kernels"
)

// Array is an array value (also includes atomic).
type Array interface {
	Value

	// Handle to the data.
	Handle() platform.Handle

	// Type returns the GX type of the array.
	Type() ir.Type

	// Shape of the array.
	Shape() *shape.Shape

	// ToDevice transfers the array to a device.
	// It is a no-op if the data is already on the device.
	ToDevice(dev platform.Device) (*DeviceArray, error)

	// ToHostArray transfers the array on the host if it is not already.
	// Use the Go allocator.
	ToHostArray(alloc platform.Allocator) (*HostArray, error)
}

type baseArray struct {
	typ       ir.Type
	arrayType ir.ArrayType
}

func newBaseArray(typ ir.Type, sh *shape.Shape) (*baseArray, error) {
	arrayType, err := ir.ToArrayTypeGivenShape(nil, typ, sh)
	if err != nil {
		return nil, err
	}
	return &baseArray{typ: typ, arrayType: arrayType}, nil
}

// DeviceArray managed by GX where the data is on a device.
type DeviceArray struct {
	*baseArray
	handle platform.DeviceHandle
}

var _ Array = (*DeviceArray)(nil)

// NewDeviceArray returns a new array managed by GX.
func NewDeviceArray(typ ir.Type, handle platform.DeviceHandle) (*DeviceArray, error) {
	base, err := newBaseArray(typ, handle.Shape())
	if err != nil {
		return nil, err
	}
	return &DeviceArray{
		baseArray: base,
		handle:    handle,
	}, nil
}

func (*DeviceArray) value() {}

// ToHost transfers the value to the host.
func (a *DeviceArray) ToHost(alloc platform.Allocator) (Value, error) {
	return a.ToHostArray(alloc)
}

// ToHostArray transfers the array to the host using the Go allocator.
func (a *DeviceArray) ToHostArray(alloc platform.Allocator) (*HostArray, error) {
	hostBuffer, err := alloc.Allocate(a.Shape())
	if err != nil {
		return nil, err
	}
	if err := a.handle.ToHost(hostBuffer); err != nil {
		return nil, err
	}
	return NewHostArray(a.typ, hostBuffer)
}

// Type of the array.
func (a *DeviceArray) Type() ir.Type {
	return a.typ
}

// Shape of the array.
func (a *DeviceArray) Shape() *shape.Shape {
	return a.handle.Shape()
}

// Handle to the data.
func (a *DeviceArray) Handle() platform.Handle {
	return a.handle
}

// DeviceHandle returns the handle pointing to the data on the device.
func (a *DeviceArray) DeviceHandle() platform.DeviceHandle {
	return a.handle
}

// ToDevice transfers the data to a device.
// It is a no-op if the data is already on the device.
func (a *DeviceArray) ToDevice(dev platform.Device) (*DeviceArray, error) {
	if a.handle.Device() == dev {
		return a, nil
	}
	handle, err := a.handle.ToDevice(dev)
	if err != nil {
		return nil, err
	}
	return NewDeviceArray(a.typ, handle)
}

// SourceString returns the GX source code of the implementation.
func (a *DeviceArray) SourceString(from *ir.File) string {
	host, err := a.ToHost(kernels.Allocator())
	var hostS string
	if err != nil {
		hostS = err.Error()
	} else {
		hostS = host.SourceString(from)
	}
	return fmt.Sprintf("DeviceArray{DeviceID: %d}: %s", a.handle.Device().Ordinal(), hostS)
}

// HostArray managed by GX where the data is on a device.
type HostArray struct {
	*baseArray
	buffer platform.HostBuffer
}

var _ Array = (*HostArray)(nil)

// NewHostArray returns a new array managed by GX.
func NewHostArray(typ ir.Type, handle platform.HostBuffer) (*HostArray, error) {
	base, err := newBaseArray(typ, handle.Shape())
	if err != nil {
		return nil, err
	}
	return &HostArray{baseArray: base, buffer: handle}, nil
}

func (*HostArray) value() {}

// Type of the array.
func (a *HostArray) Type() ir.Type {
	return a.typ
}

// Shape of the array.
func (a *HostArray) Shape() *shape.Shape {
	return a.buffer.Shape()
}

// Handle to the data.
func (a *HostArray) Handle() platform.Handle {
	return a.buffer
}

// ToHost returns receiver. The allocator is ignored.
func (a *HostArray) ToHost(platform.Allocator) (Value, error) {
	return a, nil
}

// ToHostArray returns the receiver.
func (a *HostArray) ToHostArray(platform.Allocator) (*HostArray, error) {
	return a, nil
}

// Buffer returns the buffer holding the array data.
func (a *HostArray) Buffer() platform.HostBuffer {
	return a.buffer
}

// ToDevice transfers the data to a device.
// It is a no-op if the data is already on the device.
func (a *HostArray) ToDevice(dev platform.Device) (*DeviceArray, error) {
	data := a.buffer.Acquire()
	defer a.buffer.Release()
	handle, err := dev.Send(data, a.buffer.Shape())
	if err != nil {
		return nil, err
	}
	return NewDeviceArray(a.typ, handle)
}

// ToAtom returns the value as an atomic value.
// An error is returned if the array contains more than one value.
func (a *HostArray) ToAtom() (any, error) {
	data := a.Buffer().Acquire()
	defer a.Buffer().Release()
	array, err := kernels.NewArrayFromRaw(data, a.Shape())
	if err != nil {
		return nil, err
	}
	return array.ToAtom()
}

// ToFloatNumber returns the value as a float number.
// An error is returned if the array contains more than one value.
func (a *HostArray) ToFloatNumber() (*big.Float, error) {
	data := a.Buffer().Acquire()
	defer a.Buffer().Release()
	array, err := kernels.NewArrayFromRaw(data, a.Shape())
	if err != nil {
		return nil, err
	}
	return array.ToFloatNumber()
}

// SourceString returns the GX source code of the implementation.
func (a *HostArray) SourceString(from *ir.File) string {
	data := a.Buffer().Acquire()
	defer a.Buffer().Release()
	array, err := kernels.NewArrayFromRaw(data, a.Shape())
	if err != nil {
		return fmt.Sprintf("\nError parsing raw data:\n%+v\n", err)
	}
	namedType, isNamed := a.typ.(*ir.NamedType)
	if isNamed {
		return fmt.Sprintf("%s%s", namedType.ReferString(from), array.DataString())
	}
	shape := a.buffer.Shape().AxisLengths
	dims := make([]string, len(shape))
	for i, axLen := range shape {
		dims[i] = fmt.Sprintf("[%d]", axLen)
	}
	return fmt.Sprintf("%s%s%s", strings.Join(dims, ""), a.arrayType.DataType().ReferString(from), array.DataString())
}

// String representation of the array and its type.
func (a *HostArray) String() string {
	return a.SourceString(nil)
}

// ToAtom converts an array on the host into a Go atom value.
func ToAtom[T dtypes.Supported](a *HostArray) (T, error) {
	data := a.Buffer().Acquire()
	defer a.Buffer().Release()
	slice := dtypes.ToSlice[T](data)
	if len(slice) != 1 {
		var zero T
		return zero, errors.Errorf("array (length=%d) is not an atom", len(slice))
	}
	return slice[0], nil
}
