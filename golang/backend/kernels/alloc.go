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

package kernels

import (
	"fmt"
	"reflect"
	"sync"

	"github.com/pkg/errors"
	"github.com/gx-org/backend/dtype"
	"github.com/gx-org/backend/platform"
	"github.com/gx-org/backend/shape"
)

type (
	// allocator is a Go memory allocator where the memory is fully managed by the Go runtime.
	// This is the default allocator when using GX from Go.
	allocator struct{}

	// Buffer managed by Go.
	Buffer struct {
		mut   sync.Mutex
		array Array
	}
)

var (
	_ platform.Allocator = (*allocator)(nil)
	_ Handle             = (*Buffer)(nil)
)

var goAllocator = &allocator{}

// Allocator returns an allocator allocating memory using Go.
func Allocator() platform.Allocator {
	return goAllocator
}

// Allocate Go memory given a shape.
func (allocator) Allocate(sh *shape.Shape) (platform.HostBuffer, error) {
	array, err := Zero(sh)
	if err != nil {
		return nil, err
	}
	return &Buffer{array: array}, nil
}

// NewBuffer returns a Go backend array as a generic buffer to use for GX values.
func NewBuffer(a Array) *Buffer {
	return &Buffer{array: a}
}

// Shape of the underlying array.
func (buf *Buffer) Shape() *shape.Shape {
	return buf.array.Shape()
}

// KernelValue returns the underlying Go backend array storing the data for the buffer.
func (buf *Buffer) KernelValue() Array {
	return buf.array
}

// ToDevice transfers the handle to a device.
func (buf *Buffer) ToDevice(dev platform.Device) (platform.DeviceHandle, error) {
	data := buf.Acquire()
	defer buf.Release()
	return dev.Send(data, buf.array.Shape())
}

// ToHost fetches the data from the handle and write it to buffer.
func (buf *Buffer) ToHost(target platform.HostBuffer) error {
	src := buf.Acquire()
	defer buf.Release()

	dst := target.Acquire()
	defer target.Release()

	if len(src) != len(dst) {
		return errors.Errorf("cannot copy source with length %d (shape: %s) to destination of length %d (shape: %s)", len(src), buf.Shape(), len(dst), target.Shape())
	}
	copy(dst, src)
	return nil
}

// Acquire locks the buffer and returns it.
// The buffer can be read or written by the caller. All other access is locked.
// Returns nil if the handle has been freed.
func (buf *Buffer) Acquire() []byte {
	buf.mut.Lock()
	return buf.array.Buffer()
}

// Release the buffer. The caller of that function should not read or write data
// from the buffer.
func (buf *Buffer) Release() {
	buf.mut.Unlock()
}

// Free the memory occupied by the buffer. The handle is invalid after calling this function.
func (buf *Buffer) Free() {
	buf.array = nil
}

// ToBuffer converts Go values into a GX array.
func ToBuffer[T dtype.GoDataType](vals []T, sh *shape.Shape) *Buffer {
	var array Array
	switch sh.DType {
	case dtype.Bool:
		array = ToBoolArray(any(vals).([]bool), sh.AxisLengths)
	case dtype.Float32:
		array = toAlgebraicArray[float32](sh, any(vals).([]float32))
	case dtype.Float64:
		array = toAlgebraicArray[float64](sh, any(vals).([]float64))
	case dtype.Int32:
		array = toAlgebraicArray[int32](sh, any(vals).([]int32))
	case dtype.Int64:
		array = toAlgebraicArray[int64](sh, any(vals).([]int64))
	case dtype.Uint32:
		array = toAlgebraicArray[uint32](sh, any(vals).([]uint32))
	case dtype.Uint64:
		array = toAlgebraicArray[uint64](sh, any(vals).([]uint64))
	default:
		panic(fmt.Sprintf("cannot convert data type %s to a %s %s", sh.DType.String(), reflect.TypeFor[[]T]().String(), reflect.TypeFor[*Buffer]()))
	}
	return NewBuffer(array)
}
