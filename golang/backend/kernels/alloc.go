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
	"github.com/gomlx/compute/dtypes/bfloat16"
	"github.com/gomlx/compute/dtypes"
	"github.com/gomlx/compute/dtypes/gotype"
	"github.com/gomlx/compute/shapes"
)

type (
	// HostBuffer is a handle to a buffer of data located locally on the platform,
	// typically in CPU memory, and shared between a platform and its users.
	//
	// The data needs to be acquired before being written or read. All access to the data
	// is blocked until that is released.
	HostBuffer interface {
		// Shape of the underlying array.
		Shape() shapes.Shape
		// Acquire locks the buffer and returns it.
		// The buffer can be read or written by the caller. All other access is locked.
		// Returns nil if the handle has been freed.
		Acquire() []byte
		// Release the buffer. The caller of that function should not read or write data
		// from the buffer.
		Release()
		// Free the memory occupied by the buffer. The handle is invalid after calling this function.
		Free()
	}

	// Allocator allocates memory on the host.
	Allocator interface {
		Allocate(shapes.Shape) (HostBuffer, error)
	}

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
	_ Allocator  = (*allocator)(nil)
	_ HostBuffer = (*Buffer)(nil)
	_ Handle     = (*Buffer)(nil)
)

var goAllocator = &allocator{}

// GoAllocator returns an allocator allocating memory using Go.
func GoAllocator() Allocator {
	return goAllocator
}

// DefaultAllocator returns the default allocator allocating memory using Go.
func DefaultAllocator() Allocator {
	return goAllocator
}

// Allocate Go memory given a shape.
func (allocator) Allocate(sh shapes.Shape) (HostBuffer, error) {
	array, err := Zero(sh)
	if err != nil {
		return nil, err
	}
	return &Buffer{array: array}, nil
}

// HostTransfer transfers data from a source host buffer to another.
func HostTransfer(dstB, srcB HostBuffer) error {
	src := srcB.Acquire()
	defer srcB.Release()
	dst := dstB.Acquire()
	defer dstB.Release()
	if len(src) != len(dst) {
		return errors.Errorf("cannot transfer data from a buffer of size %d to a buffer of size %d", len(src), len(dst))
	}
	copy(dst, src)
	return nil
}

// NewBuffer returns a Go backend array as a generic buffer to use for GX values.
func NewBuffer(a Array) *Buffer {
	return &Buffer{array: a}
}

// Shape of the underlying array.
func (buf *Buffer) Shape() shapes.Shape {
	return buf.array.Shape()
}

// KernelValue returns the underlying Go backend array storing the data for the buffer.
func (buf *Buffer) KernelValue() Array {
	return buf.array
}

// ToHost fetches the data from the handle and write it to buffer.
func (buf *Buffer) ToHost(dst []byte) error {
	src := buf.Acquire()
	defer buf.Release()

	if len(src) != len(dst) {
		return errors.Errorf("cannot copy source with length %d (shape: %s) to destination of length %d", len(src), buf.Shape(), len(dst))
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
func ToBuffer[T gotype.Supported](vals []T, sh shapes.Shape) *Buffer {
	var array Array
	switch sh.DType {
	case dtypes.Bool:
		array = ToBoolArray(any(vals).([]bool), sh.Dimensions)
	case dtypes.BFloat16:
		array = ToBfloat16Array(any(vals).([]bfloat16.BFloat16), sh.Dimensions)
	case dtypes.Float32:
		array = ToFloatArray[float32](any(vals).([]float32), sh.Dimensions)
	case dtypes.Float64:
		array = ToFloatArray[float64](any(vals).([]float64), sh.Dimensions)
	case dtypes.Int32:
		array = ToIntegerArray[int32](any(vals).([]int32), sh.Dimensions)
	case dtypes.Int64:
		array = ToIntegerArray[int64](any(vals).([]int64), sh.Dimensions)
	case dtypes.Uint32:
		array = ToIntegerArray[uint32](any(vals).([]uint32), sh.Dimensions)
	case dtypes.Uint64:
		array = ToIntegerArray[uint64](any(vals).([]uint64), sh.Dimensions)
	default:
		panic(fmt.Sprintf("cannot convert data type %s to a %s %s", sh.DType.String(), reflect.TypeFor[[]T]().String(), reflect.TypeFor[*Buffer]()))
	}
	return NewBuffer(array)
}
