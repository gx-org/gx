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

package types

import (
	"fmt"
	"unsafe"

	"github.com/gx-org/backend/dtype"
	"github.com/gx-org/backend/platform"
	"github.com/gx-org/gx/api"
	"github.com/gx-org/gx/api/values"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/golang/backend/kernels"
)

// Atom managed by GX.
type Atom[T dtype.GoDataType] interface {
	Bridger

	// Fetch the value from the device to the host and returns a handle to the host value.
	Fetch() (*HostAtom[T], error)

	// FetchValue fetches the atom value from the device.
	FetchValue() (T, error)
}

// DeviceAtom is an array stored on a device.
type DeviceAtom[T dtype.GoDataType] struct {
	baseBridge[*DeviceAtom[T], *values.DeviceArray]
}

var _ Atom[int64] = (*DeviceAtom[int64])(nil)

// NewDeviceAtom returns a new Go array given a device value managed by GX.
func NewDeviceAtom[T dtype.GoDataType](val *values.DeviceArray) *DeviceAtom[T] {
	atomic := &DeviceAtom[T]{}
	atomic.baseBridge = newBaseBridge(atomic, val)
	shapeGot := val.Shape()
	dtypeWant := dtype.Generic[T]()
	if shapeGot.DType != dtypeWant || !shapeGot.IsAtomic() {
		panic(fmt.Sprintf("assigning GX device value of shape %s to a Go binding atom of type %s", shapeGot.String(), dtypeWant.String()))
	}
	return atomic
}

// Fetch the value from the device.
func (atom *DeviceAtom[T]) Fetch() (*HostAtom[T], error) {
	return atom.FetchWithAlloc(kernels.Allocator())
}

// FetchWithAlloc the value from the device given a specific allocator.
func (atom *DeviceAtom[T]) FetchWithAlloc(alloc platform.Allocator) (*HostAtom[T], error) {
	val, err := atom.value.ToHostArray(alloc)
	if err != nil {
		return nil, err
	}
	return NewHostAtom[T](val), nil
}

// FetchValue returns the atom value from the device.
func (atom *DeviceAtom[T]) FetchValue() (val T, err error) {
	var host *HostAtom[T]
	host, err = atom.Fetch()
	if err != nil {
		return
	}
	val = host.Value()
	return
}

func (atom *DeviceAtom[T]) toDeviceBridger(val *values.DeviceArray) ArrayBridge {
	return NewDeviceAtom[T](val)
}

func (atom *DeviceAtom[T]) String() string {
	return atom.GXValue().String()
}

// HostAtom is an array stored on a host.
type HostAtom[T dtype.GoDataType] struct {
	baseBridge[*HostAtom[T], *values.HostArray]
}

var _ Atom[int64] = (*HostAtom[int64])(nil)

// NewHostAtom returns a new Go array given a device value managed by GX.
func NewHostAtom[T dtype.GoDataType](val *values.HostArray) *HostAtom[T] {
	atomic := &HostAtom[T]{}
	atomic.baseBridge = newBaseBridge(atomic, val)
	return atomic
}

// Fetch the value to the host.
func (atom *HostAtom[T]) Fetch() (*HostAtom[T], error) {
	return atom, nil
}

// Value of the atom.
func (atom *HostAtom[T]) Value() T {
	buf := atom.value.Buffer()
	data := buf.Acquire()
	defer buf.Release()
	return *((*T)(unsafe.Pointer(&data[0])))
}

// FetchValue returns the atom value from the device.
func (atom *HostAtom[T]) FetchValue() (val T, err error) {
	return atom.Value(), nil
}

// SendTo sends the value to a device.
func (atom *HostAtom[T]) SendTo(dev *api.Device) (*DeviceAtom[T], error) {
	devArray, err := atom.value.ToDevice(dev.PlatformDevice())
	if err != nil {
		return nil, err
	}
	return NewDeviceAtom[T](devArray), nil
}

func (atom *HostAtom[T]) toDeviceBridger(val *values.DeviceArray) ArrayBridge {
	return NewDeviceAtom[T](val)
}

func (atom *HostAtom[T]) String() string {
	return atom.GXValue().String()
}

func newAtom[T dtype.GoDataType](dt ir.Kind, array kernels.Array) *HostAtom[T] {
	typ := ir.TypeFromKind(dt)
	buffer := kernels.NewBuffer(array)
	hostArray, err := values.NewHostArray(typ, buffer)
	if err != nil {
		// Should never happen.
		// The only possible error is when an array is created with type
		// not matching its shape. We construct both in these functions.
		panic(err)
	}
	return NewHostAtom[T](hostArray)
}

// Bool returns a new Go host atom of bool.
func Bool(val bool) *HostAtom[bool] {
	return newAtom[bool](ir.BoolKind, kernels.ToBoolAtom(val))
}

// Float32 returns a new Go host atom of float32.
func Float32(val float32) *HostAtom[float32] {
	return newAtom[float32](ir.Float32Kind, kernels.ToFloatAtom(val))
}

// Float64 returns a new Go host atom of float64.
func Float64(val float64) *HostAtom[float64] {
	return newAtom[float64](ir.Float64Kind, kernels.ToFloatAtom(val))
}

// DefaultInt returns a new Go host atom of int32.
func DefaultInt(val ir.Int) *HostAtom[ir.Int] {
	return newAtom[ir.Int](ir.DefaultIntKind, kernels.ToIntegerAtom(val))
}

// Int32 returns a new Go host atom of int32.
func Int32(val int32) *HostAtom[int32] {
	return newAtom[int32](ir.Int32Kind, kernels.ToIntegerAtom(val))
}

// Int64 returns a new Go host atom of int64.
func Int64(val int64) *HostAtom[int64] {
	return newAtom[int64](ir.Int64Kind, kernels.ToIntegerAtom(val))
}

// Uint32 returns a new Go host atom of uint32.
func Uint32(val uint32) *HostAtom[uint32] {
	return newAtom[uint32](ir.Uint32Kind, kernels.ToIntegerAtom(val))
}

// Uint64 returns a new Go host atom of uint64.
func Uint64(val uint64) *HostAtom[uint64] {
	return newAtom[uint64](ir.Uint64Kind, kernels.ToIntegerAtom(val))
}

// AtomFromHost returns an atom from a value stored on the host.
func AtomFromHost[T dtype.GoDataType](hostValue *values.HostArray) T {
	return NewHostAtom[T](hostValue).Value()
}
