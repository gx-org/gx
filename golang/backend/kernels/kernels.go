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
	"go/token"
	"math/big"

	"github.com/pkg/errors"
	"github.com/gx-org/backend/dtype"
	"github.com/gx-org/backend/shape"
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
		Shape() *shape.Shape

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
		Concat(dtype.DataType, int) (NAry, *shape.Shape, error)

		Cast(target dtype.DataType, dims []int) (Unary, *shape.Shape, Factory, error)

		Slice(*shape.Shape, int) (Unary, *shape.Shape, error)

		Reshape(*shape.Shape, []int) (Unary, *shape.Shape, error)

		BroadcastInDim(*shape.Shape, []int) (Unary, *shape.Shape, error)

		UnaryOp(token.Token, *shape.Shape) (Unary, *shape.Shape, error)

		BinaryOp(token.Token, *shape.Shape, *shape.Shape) (Binary, *shape.Shape, error)

		Math() MathFactory
	}
)

func isAtomic(shape *shape.Shape) bool {
	return len(shape.AxisLengths) == 0
}

// NewArrayFromRaw returns a new array from raw data.
func NewArrayFromRaw(data []byte, sh *shape.Shape) (Array, error) {
	if len(data) != sh.ByteSize() {
		return nil, errors.Errorf("buffer size is %d but shape specify a buffer size of %d", len(data), sh.ByteSize())
	}
	switch sh.DType {
	case dtype.Bool:
		return ToBoolArray(dtype.ToSlice[bool](data), sh.AxisLengths), nil
	case dtype.Bfloat16:
		return ToBfloat16Array(dtype.ToSlice[dtype.Bfloat16T](data), sh.AxisLengths), nil
	case dtype.Float32:
		return ToFloatArray(dtype.ToSlice[float32](data), sh.AxisLengths), nil
	case dtype.Float64:
		return ToFloatArray(dtype.ToSlice[float64](data), sh.AxisLengths), nil
	case dtype.Uint32:
		return ToIntegerArray(dtype.ToSlice[uint32](data), sh.AxisLengths), nil
	case dtype.Uint64:
		return ToIntegerArray(dtype.ToSlice[uint64](data), sh.AxisLengths), nil
	case dtype.Int32:
		return ToIntegerArray(dtype.ToSlice[int32](data), sh.AxisLengths), nil
	case dtype.Int64:
		return ToIntegerArray(dtype.ToSlice[int64](data), sh.AxisLengths), nil
	default:
		return nil, errors.Errorf("cannot create an array from raw data: %s not supported", sh.DType.String())
	}
}

// FactoryFor returns a factory given a data type.
func FactoryFor(dt dtype.DataType) (Factory, error) {
	switch dt {
	case dtype.Bool:
		return boolFactory{}, nil
	case dtype.Bfloat16:
		return bfloat16Factory{}, nil
	case dtype.Float32:
		return algebraicFactory[float32]{}, nil
	case dtype.Float64:
		return algebraicFactory[float64]{}, nil
	case dtype.Uint32:
		return integerFactory[uint32]{}, nil
	case dtype.Uint64:
		return integerFactory[uint64]{}, nil
	case dtype.Int32:
		return integerFactory[int32]{}, nil
	case dtype.Int64:
		return integerFactory[int64]{}, nil
	default:
		return nil, errors.Errorf("no factory for %s", dt.String())
	}
}
