// Copyright 2026 Google LLC
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

package hostio

import (
	"fmt"
	"math/big"
	"strings"
	"unsafe"

	"github.com/pkg/errors"
	"github.com/gomlx/compute"
	"github.com/gomlx/compute/dtypes"
	"github.com/gomlx/compute/dtypes/gotype"
	"github.com/gomlx/compute/shapes"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/golang/backend/kernels"
	"github.com/gx-org/gx/internal/togo"
)

// Array is an array value (also includes atomic).
type Array interface {
	Value
	togo.CompileValue

	// Type returns the GX type of the array.
	Type() ir.Type

	// Shape of the array.
	Shape() shapes.Shape

	// ToDevice transfers the array to a device.
	// It is a no-op if the data is already on the device.
	ToDevice(bck compute.Backend, dev compute.DeviceNum) (*DeviceArray, error)

	// ToHostArray transfers the array on the host if it is not already.
	// Use the Go allocator.
	ToHostArray(alloc kernels.Allocator) (*HostArray, error)
}

type baseArray struct {
	typ       ir.Type
	arrayType ir.ArrayType
}

func newBaseArray(typ ir.Type, sh shapes.Shape) (*baseArray, error) {
	arrayType, err := ir.ToArrayTypeGivenShape(nil, typ, sh)
	if err != nil {
		return nil, err
	}
	return &baseArray{typ: typ, arrayType: arrayType}, nil
}

// DeviceArray managed by GX where the data is on a device.
type DeviceArray struct {
	*baseArray
	handle compute.Buffer
}

var _ Array = (*DeviceArray)(nil)

// NewDeviceArray returns a new array managed by GX.
func NewDeviceArray(typ ir.Type, handle compute.Buffer) (*DeviceArray, error) {
	sh, err := handle.Shape()
	if err != nil {
		return nil, err
	}
	base, err := newBaseArray(typ, sh)
	if err != nil {
		return nil, err
	}
	return &DeviceArray{
		baseArray: base,
		handle:    handle,
	}, nil
}

// ToHost transfers the value to the host.
func (a *DeviceArray) ToHost(alloc kernels.Allocator) (Value, error) {
	return a.ToHostArray(alloc)
}

// Finalize releases the device buffer.
func (a *DeviceArray) Finalize() error {
	return a.handle.Finalize()
}

// GoValue of the underlying element.
func (a *DeviceArray) GoValue() (any, error) {
	hVal, err := a.ToHostArray(kernels.DefaultAllocator())
	if err != nil {
		return nil, err
	}
	return hVal.GoValue()
}

// ToHostArray transfers the array to the host using the Go allocator.
func (a *DeviceArray) ToHostArray(alloc kernels.Allocator) (*HostArray, error) {
	sh := a.Shape()
	hostBuffer, err := alloc.Allocate(sh)
	if err != nil {
		return nil, err
	}
	data := hostBuffer.Acquire()
	defer hostBuffer.Release()
	flat := dtypes.UnsafeAnySliceFromBytes(unsafe.Pointer(unsafe.SliceData(data)), sh.DType, sh.Size())
	if err := a.handle.ToFlatData(flat); err != nil {
		return nil, err
	}
	return NewHostArray(a.typ, hostBuffer)
}

// Type of the array.
func (a *DeviceArray) Type() ir.Type {
	return a.typ
}

// Shape of the array.
func (a *DeviceArray) Shape() shapes.Shape {
	sh, err := a.handle.Shape()
	if err != nil {
		panic(err)
	}
	return sh
}

// Buffer returns the buffer pointing to the data on the device.
func (a *DeviceArray) Buffer() compute.Buffer {
	return a.handle
}

// ToDevice transfers the data to a device.
// It is a no-op if the data is already on the device.
func (a *DeviceArray) ToDevice(bck compute.Backend, dev compute.DeviceNum) (*DeviceArray, error) {
	devNum, err := a.handle.DeviceNum()
	if err != nil {
		return nil, err
	}
	if a.handle.Backend() == bck {
		if devNum == dev {
			return a, nil
		}
		handle, err := a.handle.CopyToDevice(dev)
		if err != nil {
			return nil, err
		}
		return NewDeviceArray(a.typ, handle)
	}
	sh := a.Shape()
	raw := make([]byte, sh.ByteSize())
	flat := dtypes.UnsafeAnySliceFromBytes(unsafe.Pointer(unsafe.SliceData(raw)), sh.DType, sh.Size())
	if err := a.handle.ToFlatData(flat); err != nil {
		return nil, err
	}
	handle, err := bck.BufferFromFlatData(dev, flat, sh)
	if err != nil {
		return nil, err
	}
	return NewDeviceArray(a.typ, handle)
}

// SourceString returns the GX source code of the implementation.
func (a *DeviceArray) SourceString(from *ir.File) string {
	host, err := a.ToHost(kernels.DefaultAllocator())
	var hostS string
	if err != nil {
		hostS = err.Error()
	} else {
		hostS = host.SourceString(from)
	}
	devNum, _ := a.handle.DeviceNum()
	return fmt.Sprintf("DeviceArray{DeviceID: %d}: %s", devNum, hostS)
}

// HostArray managed by GX where the data is on a device.
type HostArray struct {
	*baseArray
	buffer kernels.HostBuffer
}

var _ Array = (*HostArray)(nil)

// NewHostArray returns a new array managed by GX.
func NewHostArray(typ ir.Type, buffer kernels.HostBuffer) (*HostArray, error) {
	base, err := newBaseArray(typ, buffer.Shape())
	if err != nil {
		return nil, err
	}
	return &HostArray{baseArray: base, buffer: buffer}, nil
}

// Type of the array.
func (a *HostArray) Type() ir.Type {
	return a.typ
}

// Shape of the array.
func (a *HostArray) Shape() shapes.Shape {
	return a.buffer.Shape()
}

// ToHost returns receiver. The allocator is ignored.
func (a *HostArray) ToHost(kernels.Allocator) (Value, error) {
	return a, nil
}

// Finalize is a no-op for arrays on the host.
func (a *HostArray) Finalize() error {
	return nil
}

// ToHostArray returns the receiver.
func (a *HostArray) ToHostArray(kernels.Allocator) (*HostArray, error) {
	return a, nil
}

// Buffer returns the buffer holding the array data.
func (a *HostArray) Buffer() kernels.HostBuffer {
	return a.buffer
}

// ToDevice transfers the data to a device.
// It is a no-op if the data is already on the device.
func (a *HostArray) ToDevice(bck compute.Backend, dev compute.DeviceNum) (*DeviceArray, error) {
	data := a.buffer.Acquire()
	defer a.buffer.Release()
	sh := a.buffer.Shape()
	flat := dtypes.UnsafeAnySliceFromBytes(unsafe.Pointer(unsafe.SliceData(data)), sh.DType, sh.Size())
	handle, err := bck.BufferFromFlatData(dev, flat, sh)
	if err != nil {
		return nil, err
	}
	return NewDeviceArray(a.typ, handle)
}

// ToAtom returns the value as an atomic value.
// An error is returned if the array contains more than one value.
func (a *HostArray) ToAtom() (any, error) {
	array, err := a.goValue()
	if err != nil {
		return nil, err
	}
	return array.ToAtom()
}

func (a *HostArray) goValue() (kernels.Array, error) {
	data := a.Buffer().Acquire()
	defer a.Buffer().Release()
	return kernels.NewArrayFromRaw(data, a.Shape())
}

// GoValue of the underlying element.
func (a *HostArray) GoValue() (any, error) {
	return a.goValue()
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

// ShortString returns the values of the array as a string without the type.
func (a *HostArray) ShortString(from *ir.File) string {
	return a.sourceString(from, true)
}

// SourceString returns the GX source code of the implementation.
func (a *HostArray) SourceString(from *ir.File) string {
	return a.sourceString(from, false)
}

func (a *HostArray) sourceString(from *ir.File, short bool) string {
	data := a.Buffer().Acquire()
	defer a.Buffer().Release()
	array, err := kernels.NewArrayFromRaw(data, a.Shape())
	if err != nil {
		return fmt.Sprintf("\nError parsing raw data:\n%+v\n", err)
	}
	if short {
		return array.DataString()
	}
	shape := a.buffer.Shape().Dimensions
	dims := make([]string, len(shape))
	for i, axLen := range shape {
		dims[i] = fmt.Sprintf("[%d]", axLen)
	}
	dataS := array.DataString()
	if len(shape) == 0 {
		dataS = fmt.Sprintf("(%s)", dataS)
	}
	namedType, isNamed := a.typ.(*ir.NamedType)
	if isNamed {
		return fmt.Sprintf("%s%s", namedType.ReferString(from), dataS)
	}
	return fmt.Sprintf("%s%s%s", strings.Join(dims, ""), a.arrayType.DataType().ReferString(from), dataS)
}

// String representation of the array and its type.
func (a *HostArray) String() string {
	return a.SourceString(nil)
}

// ToAtom converts an array on the host into a Go atom value.
func ToAtom[T gotype.Supported](a *HostArray) (T, error) {
	data := a.Buffer().Acquire()
	defer a.Buffer().Release()
	slice := kernels.ToSlice[T](data)
	if len(slice) != 1 {
		var zero T
		return zero, errors.Errorf("array (length=%d) is not an atom", len(slice))
	}
	return slice[0], nil
}
