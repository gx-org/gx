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
	"github.com/gx-org/backend/dtype"
	"github.com/gx-org/backend/platform"
	"github.com/gx-org/backend/shape"
	"github.com/gx-org/gx/api/values"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/golang/backend/kernels"
)

// Array managed by GX.
type Array[T dtype.GoDataType] interface {
	Bridger

	Fetch() (*HostArray[T], error)

	Shape() *shape.Shape
}

// DeviceArray is an array stored on a device.
type DeviceArray[T dtype.GoDataType] struct {
	baseBridge[*DeviceArray[T], *values.DeviceArray]
}

var _ Array[int64] = (*DeviceArray[int64])(nil)

// NewDeviceArray returns a new Go array given a device value managed by GX.
func NewDeviceArray[T dtype.GoDataType](val *values.DeviceArray) *DeviceArray[T] {
	array := &DeviceArray[T]{}
	array.baseBridge = newBaseBridge(array, val)
	return array
}

// Fetch the value from the device.
func (array *DeviceArray[T]) Fetch() (*HostArray[T], error) {
	return array.FetchWithAlloc(kernels.Allocator())
}

// Shape of the array.
func (array *DeviceArray[T]) Shape() *shape.Shape {
	return array.value.Shape()
}

// FetchWithAlloc the value from the device given a specific allocator.
func (array *DeviceArray[T]) FetchWithAlloc(alloc platform.Allocator) (*HostArray[T], error) {
	val, err := array.value.ToHostArray(alloc)
	if err != nil {
		return nil, err
	}
	return NewHostArray[T](val), nil
}

func (array *DeviceArray[T]) toDeviceBridger(val *values.DeviceArray) ArrayBridge {
	return NewDeviceArray[T](val)
}

// HostArray is an array stored on a host.
type HostArray[T dtype.GoDataType] struct {
	baseBridge[*HostArray[T], *values.HostArray]
}

var _ Array[int64] = (*HostArray[int64])(nil)

// NewHostArray returns a new Go array given a device value managed by GX.
func NewHostArray[T dtype.GoDataType](val *values.HostArray) *HostArray[T] {
	array := &HostArray[T]{}
	array.baseBridge = newBaseBridge(array, val)
	return array
}

// Fetch the value to the host.
func (array *HostArray[T]) Fetch() (*HostArray[T], error) {
	return array, nil
}

// CopyFlat makes a copy of the data and returns it as a flat representation of an array.
func (array *HostArray[T]) CopyFlat() []T {
	buffer := array.value.Buffer()
	src := buffer.Acquire()
	defer buffer.Release()
	dst := append([]uint8{}, src...)
	return dtype.ToSlice[T](dst)
}

// Shape of the array.
func (array *HostArray[T]) Shape() *shape.Shape {
	return array.value.Shape()
}

func (array *HostArray[T]) toDeviceBridger(val *values.DeviceArray) ArrayBridge {
	return NewDeviceArray[T](val)
}

// ArrayBool returns a new Go host array of bool.
func ArrayBool(vals []bool, dims ...int) *HostArray[bool] {
	if dims == nil {
		dims = []int{len(vals)}
	}
	typ := ir.NewArrayType(nil, ir.TypeFromKind(ir.BoolKind), ir.NewRank(dims))
	array := kernels.ToBoolArray(vals, dims)
	buffer := kernels.NewBuffer(array)
	return NewHostArray[bool](values.NewHostArray(typ, buffer))
}

func inferDims[T dtype.GoDataType](vals []T, dims []int) []int {
	if dims != nil {
		return dims
	}
	return []int{len(vals)}
}

func newArray[T dtype.AlgebraType](dtypeKind ir.Kind, array kernels.Array) *HostArray[T] {
	dims := array.Shape().AxisLengths
	typ := ir.NewArrayType(nil, ir.TypeFromKind(dtypeKind), ir.NewRank(dims))
	buffer := kernels.NewBuffer(array)
	return NewHostArray[T](values.NewHostArray(typ, buffer))
}

// ArrayFloat32 returns a new Go host array of float32.
func ArrayFloat32(vals []float32, dims ...int) *HostArray[float32] {
	dims = inferDims(vals, dims)
	return newArray[float32](ir.Float32Kind, kernels.ToFloatArray(vals, dims))
}

// ArrayFloat64 returns a new Go host array of float64.
func ArrayFloat64(vals []float64, dims ...int) *HostArray[float64] {
	dims = inferDims(vals, dims)
	return newArray[float64](ir.Float64Kind, kernels.ToFloatArray(vals, dims))
}

// ArrayInt32 returns a new Go host array of int32.
func ArrayInt32(vals []int32, dims ...int) *HostArray[int32] {
	dims = inferDims(vals, dims)
	return newArray[int32](ir.Int32Kind, kernels.ToIntegerArray(vals, dims))
}

// ArrayInt64 returns a new Go host array of int64.
func ArrayInt64(vals []int64, dims ...int) *HostArray[int64] {
	dims = inferDims(vals, dims)
	return newArray[int64](ir.Int64Kind, kernels.ToIntegerArray(vals, dims))
}

// ArrayUint32 returns a new Go host array of uint32.
func ArrayUint32(vals []uint32, dims ...int) *HostArray[uint32] {
	dims = inferDims(vals, dims)
	return newArray[uint32](ir.Uint32Kind, kernels.ToIntegerArray(vals, dims))
}

// ArrayUint64 returns a new Go host array of uint64.
func ArrayUint64(vals []uint64, dims ...int) *HostArray[uint64] {
	dims = inferDims(vals, dims)
	return newArray[uint64](ir.Uint64Kind, kernels.ToIntegerArray(vals, dims))
}
