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

// Package kernels implement Go kernels for GX.
package kernels

import (
	"fmt"
	"go/token"
	"math/big"
	"reflect"
	"unsafe"

	"github.com/pkg/errors"
	"github.com/gomlx/compute/dtypes/bfloat16"
	"github.com/gomlx/compute/dtypes"
	"github.com/gomlx/compute/shapes"
)

type (
	// Array is a value (tuple, atomic, or array) managed by the backend.
	Array interface {
		// newArray returns a new array implementation given a base array.
		newArray(baseArray) Array

		// base returns the base array supporting this array implementation.
		base() baseArray

		// Factory returns the kernels available for the value.
		Factory() Factory

		// Shape returns the shape of the value.
		Shape() shapes.Shape

		// Buffer returns the data of the array as a generic []uint8 buffer.
		Buffer() []byte

		// ToAtom returns the atomic value contained in the array.
		// It returns an error if the value is not atomic, that is if the array
		// contains more than one value.
		ToAtom() (any, error)

		// ToFloatNumber returns the atomic value as a float.
		// It returns an error if the value is not atomic, that is if the array
		// contains more than one value.
		ToFloatNumber() (*big.Float, error)

		// DataString representation of the array without the type.
		DataString() string

		// String representation of the array.
		String() string
	}

	// Handle is a value owned by the Go backend.
	Handle interface {
		KernelValue() Array
	}

	// Unary like - or reshape.
	Unary func(Array) (Array, error)

	// Binary like +, -, *, /.
	Binary func(Array, Array) (Array, error)

	// NAry operator like Concat.
	NAry func([]Array) (Array, error)

	// Factory creates kernels for arrays for all supported types.
	Factory interface {
		Concat(dtypes.DType, int) (NAry, shapes.Shape, error)

		Cast(target dtypes.DType, dims []int) (Unary, shapes.Shape, Factory, error)

		Slice(shapes.Shape, int) (Unary, shapes.Shape, error)

		Reshape(shapes.Shape, []int) (Unary, shapes.Shape, error)

		BroadcastInDim(shapes.Shape, []int) (Unary, shapes.Shape, error)

		UnaryOp(token.Token, shapes.Shape) (Unary, shapes.Shape, error)

		BinaryOp(token.Token, shapes.Shape, shapes.Shape) (Binary, shapes.Shape, error)

		Math() MathFactory
	}
)

func isAtomic(shape shapes.Shape) bool {
	return shape.IsScalar()
}

// ToSlice converts a []byte buffer into a slice of a given Go type.
func ToSlice[T any](data []byte) []T {
	if len(data) == 0 {
		return nil
	}
	var t T
	size := int(unsafe.Sizeof(t))
	if len(data)%size != 0 {
		typeName := reflect.TypeFor[T]().String()
		panic(fmt.Sprintf("data [%d]byte cannot be casted to []%s: %d %% sizeof(%s) != 0", len(data), typeName, len(data), typeName))
	}
	length := len(data) / size
	return unsafe.Slice((*T)(unsafe.Pointer(&data[0])), length)
}

// NewArrayFromRaw returns a new array from raw data.
func NewArrayFromRaw(data []byte, sh shapes.Shape) (Array, error) {
	if int64(len(data)) != sh.ByteSize() {
		return nil, errors.Errorf("buffer size is %d but shape specify a buffer size of %d", len(data), sh.ByteSize())
	}
	switch sh.DType {
	case dtypes.Bool:
		return ToBoolArray(ToSlice[bool](data), sh.Dimensions), nil
	case dtypes.BFloat16:
		return ToBfloat16Array(ToSlice[bfloat16.BFloat16](data), sh.Dimensions), nil
	case dtypes.Float32:
		return ToFloatArray(ToSlice[float32](data), sh.Dimensions), nil
	case dtypes.Float64:
		return ToFloatArray(ToSlice[float64](data), sh.Dimensions), nil
	case dtypes.Uint32:
		return ToIntegerArray(ToSlice[uint32](data), sh.Dimensions), nil
	case dtypes.Uint64:
		return ToIntegerArray(ToSlice[uint64](data), sh.Dimensions), nil
	case dtypes.Int32:
		return ToIntegerArray(ToSlice[int32](data), sh.Dimensions), nil
	case dtypes.Int64:
		return ToIntegerArray(ToSlice[int64](data), sh.Dimensions), nil
	default:
		return nil, errors.Errorf("cannot create an array from raw data: %s not supported", sh.DType.String())
	}
}

// FactoryFor returns a factory given a data type.
func FactoryFor(dt dtypes.DType) (Factory, error) {
	switch dt {
	case dtypes.Bool:
		return boolFactory{}, nil
	case dtypes.BFloat16:
		return bfloat16Factory{}, nil
	case dtypes.Float32:
		return &algebraicFactory[float32]{}, nil
	case dtypes.Float64:
		return &algebraicFactory[float64]{}, nil
	case dtypes.Uint32:
		return &integerFactory[uint32]{}, nil
	case dtypes.Uint64:
		return &integerFactory[uint64]{}, nil
	case dtypes.Int32:
		return &integerFactory[int32]{}, nil
	case dtypes.Int64:
		return &integerFactory[int64]{}, nil
	default:
		return nil, errors.Errorf("no factory for %s", dt.String())
	}
}
