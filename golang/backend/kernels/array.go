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
	"github.com/gomlx/compute/dtypes/bfloat16"
	"github.com/gomlx/compute/dtypes"
	"github.com/gomlx/compute/dtypes/gotype"
	"github.com/gomlx/compute/shapes"
	"github.com/gx-org/gx/fmt/fmtarray"
)

type (
	arrayFactory[T gotype.Supported] struct{}

	baseArray interface {
		array()
	}

	// arrayT is a multi-dimensional array stored by the host or the device.
	arrayT[T gotype.Supported] struct {
		shape   shapes.Shape
		values  []T
		factory Factory
	}
)

var _ baseArray = (*arrayT[int32])(nil)

func toArray[T gotype.Supported](a Array) *arrayT[T] {
	return a.base().(*arrayT[T])
}

func (a *arrayT[T]) array() {}

// Shape of the array.
func (a *arrayT[T]) Shape() shapes.Shape {
	return a.shape
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

// DataString representation of the data in the array.
func (a *arrayT[T]) DataString() string {
	return fmtarray.SDataPrint[T](a.Flat(), a.shape.Dimensions)
}

// String representation of the array.
func (a *arrayT[T]) String() string {
	return fmtarray.Sprint[T](a.Flat(), a.shape.Dimensions)
}

// Buffer returns the data of the array as a generic []byte buffer.
func (a *arrayT[T]) Buffer() []byte {
	if len(a.values) == 0 {
		return nil
	}
	ptr := unsafe.Pointer(&(a.values[0]))
	return unsafe.Slice((*byte)(ptr), int(a.shape.ByteSize()))
}

// Factory available for arrays.
func (a *arrayT[T]) Factory() Factory {
	return a.factory
}

func (a *arrayT[T]) toAtom() (val T, err error) {
	if !a.shape.IsScalar() {
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
	shape := shapes.Make(a.shape.DType, dims...)
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
		shape:   shapes.Make(dtypes.FromGenericsType[T](), dims...),
		values:  append([]T{}, a.values...),
	}
}

func (f arrayFactory[T]) Slice(x shapes.Shape, index int) (Unary, shapes.Shape, error) {
	shapeOut := shapes.Make(x.DType, x.Dimensions[1:]...)
	stride := shapeOut.Size()
	return func(a Array) (Array, error) {
		aT := toArray[T](a)
		return a.newArray(&arrayT[T]{
			factory: aT.factory,
			shape:   shapeOut,
			values:  aT.values[index*stride : (index+1)*stride],
		}), nil
	}, shapeOut, nil
}

func (f arrayFactory[T]) Reshape(x shapes.Shape, axisLengths []int) (Unary, shapes.Shape, error) {
	shapeOut := shapes.Make(dtypes.FromGenericsType[T](), axisLengths...)
	return func(a Array) (Array, error) {
		aT := toArray[T](a)
		return a.newArray(&arrayT[T]{
			factory: aT.factory,
			shape:   shapeOut,
			values:  aT.values,
		}), nil
	}, shapeOut, nil
}

func broadcast[T gotype.Supported](vals []T, target int) ([]T, error) {
	if len(vals) != 1 {
		return nil, errors.Errorf("cannot broadcast more than one value")
	}
	out := make([]T, target)
	for i := range out {
		out[i] = vals[0]
	}
	return out, nil
}

func (f arrayFactory[T]) BroadcastInDim(targetShape shapes.Shape, broadcastAxes []int) (Unary, shapes.Shape, error) {
	return func(a Array) (Array, error) {
		aT := toArray[T](a)
		vals, err := broadcast(aT.values, targetShape.Size())
		if err != nil {
			return nil, err
		}
		return a.newArray(&arrayT[T]{
			factory: aT.factory,
			shape:   targetShape,
			values:  vals,
		}), nil
	}, targetShape, nil
}

func (arrayFactory[T]) Concat(dt dtypes.DType, n int) (NAry, shapes.Shape, error) {
	return concat[T], shapes.Make(dt, n), nil
}

type (
	algebraArray[T gotype.NumericNotComplex] struct {
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

type nonAlgebraArray[T bool] struct {
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
	*arrayT[bfloat16.BFloat16]
}

var _ Array = (*bfloat16Array)(nil)

func (a *bfloat16Array) newArray(b baseArray) Array {
	return &bfloat16Array{arrayT: b.(*arrayT[bfloat16.BFloat16])}
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
