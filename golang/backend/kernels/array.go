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
	"unsafe"

	"github.com/pkg/errors"
	"github.com/gx-org/backend/dtype"
	"github.com/gx-org/backend/shape"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/fmt/fmtarray"
)

type (
	arrayFactory[T dtype.GoDataType] struct{}

	// ArrayT is a multi-dimensional array stored by the host or the device.
	ArrayT[T dtype.GoDataType] struct {
		shape   shape.Shape
		values  []T
		factory Factory
	}
)

var _ Array = (*ArrayT[ir.Int])(nil)

func (a *ArrayT[T]) arrayT() {}

// Shape of the array.
func (a *ArrayT[T]) Shape() *shape.Shape {
	return &a.shape
}

// Flat values of the array.
func (a *ArrayT[T]) Flat() []T {
	if a.values == nil {
		size := a.shape.Size()
		if size > 0 {
			a.values = make([]T, size)
		}
	}
	return a.values
}

// String representation of the array.
func (a *ArrayT[T]) String() string {
	return fmtarray.Sprint[T](a.Flat(), a.Shape().AxisLengths)
}

// Buffer returns the data of the array as a generic []byte buffer.
func (a *ArrayT[T]) Buffer() []byte {
	ptr := unsafe.Pointer(&(a.values[0]))
	return unsafe.Slice((*byte)(ptr), a.shape.Size()*dtype.Sizeof(a.shape.DType))
}

// Factory available for arrays.
func (a *ArrayT[T]) Factory() Factory {
	return a.factory
}

// ToAtom returns the atomic value contained in the array.
// It returns an error if the value is not atomic, that is if the array
// contains more than one value.
func (a *ArrayT[T]) ToAtom() (any, error) {
	if !a.shape.IsAtomic() {
		return nil, errors.Errorf("%s not atomic", a.shape.String())
	}
	return a.values[0], nil
}

// ScalarToArray fills an array with a scalar value.
func (a *ArrayT[T]) ScalarToArray(dims []int) *ArrayT[T] {
	shape := shape.Shape{
		DType:       a.shape.DType,
		AxisLengths: dims,
	}
	values := make([]T, shape.Size())
	for i := range values {
		values[i] = a.values[0]
	}
	return &ArrayT[T]{
		shape:   shape,
		values:  values,
		factory: a.factory,
	}
}

func (a *ArrayT[T]) reshapeArray(dims []int) (Array, error) {
	return &ArrayT[T]{
		factory: a.factory,
		shape: shape.Shape{
			DType:       dtype.Generic[T](),
			AxisLengths: dims,
		},
		values: append([]T{}, a.values...),
	}, nil
}

// Reshape scalar kernel.
func (f arrayFactory[T]) Reshape(x *shape.Shape, axisLengths []int) (Unary, *shape.Shape, error) {
	shapeOut := shape.Shape{
		DType:       dtype.Generic[T](),
		AxisLengths: axisLengths,
	}
	return func(a Array) (Array, error) {
		aT := a.(*ArrayT[T])
		return &ArrayT[T]{
			factory: aT.factory,
			shape:   shapeOut,
			values:  aT.values,
		}, nil
	}, &shapeOut, nil
}

func (arrayFactory[T]) Concat(dt dtype.DataType, n int) (NAry, *shape.Shape, error) {
	return concat[T], &shape.Shape{
		DType:       dt,
		AxisLengths: []int{n},
	}, nil
}
