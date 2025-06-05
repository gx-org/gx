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
	"math/big"
	"unsafe"

	"github.com/pkg/errors"
	"github.com/gx-org/backend/dtype"
	"github.com/gx-org/backend/shape"
	"github.com/gx-org/gx/fmt/fmtarray"
)

type (
	arrayFactory[T dtype.GoDataType] struct{}

	baseArray interface {
		array()
	}

	// arrayT is a multi-dimensional array stored by the host or the device.
	arrayT[T dtype.GoDataType] struct {
		shape   shape.Shape
		values  []T
		factory Factory
	}
)

var _ baseArray = (*arrayT[int32])(nil)

func toArray[T dtype.GoDataType](a Array) *arrayT[T] {
	return a.base().(*arrayT[T])
}

func (a *arrayT[T]) array() {}

// Shape of the array.
func (a *arrayT[T]) Shape() *shape.Shape {
	return &a.shape
}

// Flat values of the array.
func (a *arrayT[T]) Flat() []T {
	if a.values == nil {
		size := a.shape.Size()
		if size > 0 {
			a.values = make([]T, size)
		}
	}
	return a.values
}

// String representation of the array.
func (a *arrayT[T]) String() string {
	return fmtarray.Sprint[T](a.Flat(), a.Shape().AxisLengths)
}

// Buffer returns the data of the array as a generic []byte buffer.
func (a *arrayT[T]) Buffer() []byte {
	ptr := unsafe.Pointer(&(a.values[0]))
	return unsafe.Slice((*byte)(ptr), a.shape.Size()*dtype.Sizeof(a.shape.DType))
}

// Factory available for arrays.
func (a *arrayT[T]) Factory() Factory {
	return a.factory
}

func (a *arrayT[T]) toAtom() (val T, err error) {
	if !a.shape.IsAtomic() {
		err = errors.Errorf("%s not atomic", a.shape.String())
	}
	val = a.values[0]
	return
}

// ToAtom returns the atomic value contained in the array.
// It returns an error if the value is not atomic, that is if the array
// contains more than one value.
func (a *arrayT[T]) ToAtom() (any, error) {
	return a.toAtom()
}

// ScalarToArray fills an array with a scalar value.
func (a *arrayT[T]) ScalarToArray(dims []int) *arrayT[T] {
	shape := shape.Shape{
		DType:       a.shape.DType,
		AxisLengths: dims,
	}
	values := make([]T, shape.Size())
	for i := range values {
		values[i] = a.values[0]
	}
	return &arrayT[T]{
		shape:   shape,
		values:  values,
		factory: a.factory,
	}
}

func (a *arrayT[T]) reshapeArray(dims []int) arrayT[T] {
	return arrayT[T]{
		factory: a.factory,
		shape: shape.Shape{
			DType:       dtype.Generic[T](),
			AxisLengths: dims,
		},
		values: append([]T{}, a.values...),
	}
}

func (f arrayFactory[T]) Slice(x *shape.Shape, index int) (Unary, *shape.Shape, error) {
	shapeOut := shape.Shape{
		DType:       x.DType,
		AxisLengths: x.AxisLengths[1:],
	}
	stride := shapeOut.Size()
	return func(a Array) (Array, error) {
		aT := toArray[T](a)
		return a.newArray(&arrayT[T]{
			factory: aT.factory,
			shape:   shapeOut,
			values:  aT.values[index*stride : (index+1)*stride],
		}), nil
	}, &shapeOut, nil
}

// Reshape scalar kernel.
func (f arrayFactory[T]) Reshape(x *shape.Shape, axisLengths []int) (Unary, *shape.Shape, error) {
	shapeOut := shape.Shape{
		DType:       dtype.Generic[T](),
		AxisLengths: axisLengths,
	}
	return func(a Array) (Array, error) {
		aT := toArray[T](a)
		return a.newArray(&arrayT[T]{
			factory: aT.factory,
			shape:   shapeOut,
			values:  aT.values,
		}), nil
	}, &shapeOut, nil
}

func (arrayFactory[T]) Concat(dt dtype.DataType, n int) (NAry, *shape.Shape, error) {
	return concat[T], &shape.Shape{
		DType:       dt,
		AxisLengths: []int{n},
	}, nil
}

type (
	goAlgebra interface {
		dtype.Float | dtype.IntegerType
	}

	algebraArray[T goAlgebra] struct {
		*arrayT[T]
	}
)

var _ Array = (*algebraArray[int32])(nil)

func (a *algebraArray[T]) newArray(b baseArray) Array {
	return &algebraArray[T]{arrayT: b.(*arrayT[T])}
}

func (a *algebraArray[T]) base() baseArray {
	return a.arrayT
}

func (a *algebraArray[T]) ToFloatNumber() (*big.Float, error) {
	val, err := a.toAtom()
	if err != nil {
		return nil, err
	}
	return big.NewFloat(float64(val)), nil
}

type nonAlgebraArray[T dtype.NonAlgebraType] struct {
	*arrayT[T]
}

var _ Array = (*nonAlgebraArray[bool])(nil)

func (a *nonAlgebraArray[T]) newArray(b baseArray) Array {
	return &nonAlgebraArray[T]{arrayT: b.(*arrayT[T])}
}

func (a *nonAlgebraArray[T]) base() baseArray {
	return a.arrayT
}

func (a *nonAlgebraArray[T]) ToFloatNumber() (*big.Float, error) {
	return nil, errors.Errorf("cannot convert non-algebra type %s to big.Float", a.shape.DType)
}

type bfloat16Array struct {
	*arrayT[dtype.Bfloat16T]
}

var _ Array = (*bfloat16Array)(nil)

func (a *bfloat16Array) newArray(b baseArray) Array {
	return &bfloat16Array{arrayT: b.(*arrayT[dtype.Bfloat16T])}
}

func (a *bfloat16Array) base() baseArray {
	return a.arrayT
}

func (a *bfloat16Array) ToFloatNumber() (*big.Float, error) {
	val, err := a.toAtom()
	if err != nil {
		return nil, err
	}
	return big.NewFloat(float64(val.Float32())), nil
}
