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
	"github.com/gomlx/compute/dtypes/gotype"
	"github.com/gomlx/compute/shapes"
)

// AtomicToAtomic

func addAtomicToAtomic[T gotype.NumericNotComplex](xVal, yVal Array) (Array, error) {
	x, y := toArray[T](xVal), toArray[T](yVal)
	return ToAlgebraicArray[T]([]T{x.values[0] + y.values[0]}, nil), nil
}

func subAtomicToAtomic[T gotype.NumericNotComplex](xVal, yVal Array) (Array, error) {
	x, y := toArray[T](xVal), toArray[T](yVal)
	return ToAlgebraicArray[T]([]T{x.values[0] - y.values[0]}, nil), nil
}

func mulAtomicToAtomic[T gotype.NumericNotComplex](xVal, yVal Array) (Array, error) {
	x, y := toArray[T](xVal), toArray[T](yVal)
	return ToAlgebraicArray[T]([]T{x.values[0] * y.values[0]}, nil), nil
}

func quoAtomicToAtomic[T gotype.NumericNotComplex](xVal, yVal Array) (Array, error) {
	x, y := toArray[T](xVal), toArray[T](yVal)
	return ToAlgebraicArray[T]([]T{x.values[0] / y.values[0]}, nil), nil
}

func remAtomicToAtomic[T gotype.Integer](xVal, yVal Array) (Array, error) {
	x, y := toArray[T](xVal), toArray[T](yVal)
	return ToIntegerArray[T]([]T{x.values[0] % y.values[0]}, nil), nil
}

func shlAtomicToAtomic[T gotype.Integer](xVal, yVal Array) (Array, error) {
	x, y := toArray[T](xVal), toArray[T](yVal)
	return ToIntegerArray[T]([]T{x.values[0] << y.values[0]}, nil), nil
}

func shrAtomicToAtomic[T gotype.Integer](xVal, yVal Array) (Array, error) {
	x, y := toArray[T](xVal), toArray[T](yVal)
	return ToIntegerArray[T]([]T{x.values[0] >> y.values[0]}, nil), nil
}

func andAtomicToAtomic[T gotype.Integer](xVal, yVal Array) (Array, error) {
	x, y := toArray[T](xVal), toArray[T](yVal)
	return ToIntegerArray[T]([]T{x.values[0] & y.values[0]}, nil), nil
}

func orAtomicToAtomic[T gotype.Integer](xVal, yVal Array) (Array, error) {
	x, y := toArray[T](xVal), toArray[T](yVal)
	return ToIntegerArray[T]([]T{x.values[0] | y.values[0]}, nil), nil
}

func xorAtomicToAtomic[T gotype.Integer](xVal, yVal Array) (Array, error) {
	x, y := toArray[T](xVal), toArray[T](yVal)
	return ToIntegerArray[T]([]T{x.values[0] ^ y.values[0]}, nil), nil
}

func eqlAtomicToAtomic[T gotype.NumericNotComplex](xVal, yVal Array) (Array, error) {
	x, y := toArray[T](xVal), toArray[T](yVal)
	return ToBoolArray([]bool{x.values[0] == y.values[0]}, nil), nil
}

func neqAtomicToAtomic[T gotype.NumericNotComplex](xVal, yVal Array) (Array, error) {
	x, y := toArray[T](xVal), toArray[T](yVal)
	return ToBoolArray([]bool{x.values[0] != y.values[0]}, nil), nil
}

func lssAtomicToAtomic[T gotype.NumericNotComplex](xVal, yVal Array) (Array, error) {
	x, y := toArray[T](xVal), toArray[T](yVal)
	return ToBoolArray([]bool{x.values[0] < y.values[0]}, nil), nil
}

func gtrAtomicToAtomic[T gotype.NumericNotComplex](xVal, yVal Array) (Array, error) {
	x, y := toArray[T](xVal), toArray[T](yVal)
	return ToBoolArray([]bool{x.values[0] > y.values[0]}, nil), nil
}

func leqAtomicToAtomic[T gotype.NumericNotComplex](xVal, yVal Array) (Array, error) {
	x, y := toArray[T](xVal), toArray[T](yVal)
	return ToBoolArray([]bool{x.values[0] <= y.values[0]}, nil), nil
}

func geqAtomicToAtomic[T gotype.NumericNotComplex](xVal, yVal Array) (Array, error) {
	x, y := toArray[T](xVal), toArray[T](yVal)
	return ToBoolArray([]bool{x.values[0] >= y.values[0]}, nil), nil
}

// AtomicToArray

func addAtomicToArray[T gotype.NumericNotComplex](xVal, yVal Array) (Array, error) {
	x, y := toArray[T](xVal).values[0], toArray[T](yVal)
	z := make([]T, y.shape.Size())
	for i, yi := range y.values {
		z[i] = x + yi
	}
	return ToAlgebraicArray[T](z, y.shape.Dimensions), nil
}

func subAtomicToArray[T gotype.NumericNotComplex](xVal, yVal Array) (Array, error) {
	x, y := toArray[T](xVal).values[0], toArray[T](yVal)
	z := make([]T, y.shape.Size())
	for i, yi := range y.values {
		z[i] = x - yi
	}
	return ToAlgebraicArray[T](z, y.shape.Dimensions), nil
}

func mulAtomicToArray[T gotype.NumericNotComplex](xVal, yVal Array) (Array, error) {
	x, y := toArray[T](xVal).values[0], toArray[T](yVal)
	z := make([]T, y.shape.Size())
	for i, yi := range y.values {
		z[i] = x * yi
	}
	return ToAlgebraicArray[T](z, y.shape.Dimensions), nil
}

func quoAtomicToArray[T gotype.NumericNotComplex](xVal, yVal Array) (Array, error) {
	x, y := toArray[T](xVal).values[0], toArray[T](yVal)
	z := make([]T, y.shape.Size())
	for i, yi := range y.values {
		z[i] = x / yi
	}
	return ToAlgebraicArray[T](z, y.shape.Dimensions), nil
}

func remAtomicToArray[T gotype.Integer](xVal, yVal Array) (Array, error) {
	x, y := toArray[T](xVal).values[0], toArray[T](yVal)
	z := make([]T, y.shape.Size())
	for i, yi := range y.values {
		z[i] = x % yi
	}
	return ToIntegerArray[T](z, y.shape.Dimensions), nil
}

func shlAtomicToArray[T gotype.Integer](xVal, yVal Array) (Array, error) {
	x, y := toArray[T](xVal).values[0], toArray[T](yVal)
	z := make([]T, y.shape.Size())
	for i, yi := range y.values {
		z[i] = x << yi
	}
	return ToIntegerArray[T](z, y.shape.Dimensions), nil
}

func shrAtomicToArray[T gotype.Integer](xVal, yVal Array) (Array, error) {
	x, y := toArray[T](xVal).values[0], toArray[T](yVal)
	z := make([]T, y.shape.Size())
	for i, yi := range y.values {
		z[i] = x >> yi
	}
	return ToIntegerArray[T](z, y.shape.Dimensions), nil
}

func andAtomicToArray[T gotype.Integer](xVal, yVal Array) (Array, error) {
	x, y := toArray[T](xVal).values[0], toArray[T](yVal)
	z := make([]T, y.shape.Size())
	for i, yi := range y.values {
		z[i] = x & yi
	}
	return ToIntegerArray[T](z, y.shape.Dimensions), nil
}

func orAtomicToArray[T gotype.Integer](xVal, yVal Array) (Array, error) {
	x, y := toArray[T](xVal).values[0], toArray[T](yVal)
	z := make([]T, y.shape.Size())
	for i, yi := range y.values {
		z[i] = x | yi
	}
	return ToIntegerArray[T](z, y.shape.Dimensions), nil
}

func xorAtomicToArray[T gotype.Integer](xVal, yVal Array) (Array, error) {
	x, y := toArray[T](xVal).values[0], toArray[T](yVal)
	z := make([]T, y.shape.Size())
	for i, yi := range y.values {
		z[i] = x ^ yi
	}
	return ToIntegerArray[T](z, y.shape.Dimensions), nil
}

func equalAtomicToArray[T gotype.NumericNotComplex](xVal, yVal Array) (Array, error) {
	x, y := toArray[T](xVal).values[0], toArray[T](yVal)
	z := make([]bool, y.shape.Size())
	for i, yi := range y.values {
		z[i] = x == yi
	}
	return ToBoolArray(z, y.shape.Dimensions), nil
}

// ArrayToAtomic

func addArrayToAtomic[T gotype.NumericNotComplex](xVal, yVal Array) (Array, error) {
	return addAtomicToArray[T](yVal, xVal)
}

func subArrayToAtomic[T gotype.NumericNotComplex](xVal, yVal Array) (Array, error) {
	x, y := toArray[T](xVal), toArray[T](yVal).values[0]
	z := make([]T, x.shape.Size())
	for i, xi := range x.values {
		z[i] = xi - y
	}
	return ToAlgebraicArray[T](z, x.shape.Dimensions), nil
}

func mulArrayToAtomic[T gotype.NumericNotComplex](xVal, yVal Array) (Array, error) {
	return mulAtomicToArray[T](yVal, xVal)
}

func quoArrayToAtomic[T gotype.NumericNotComplex](xVal, yVal Array) (Array, error) {
	x, y := toArray[T](xVal), toArray[T](yVal).values[0]
	z := make([]T, x.shape.Size())
	for i, xi := range x.values {
		z[i] = xi / y
	}
	return ToAlgebraicArray[T](z, x.shape.Dimensions), nil
}

func remArrayToAtomic[T gotype.Integer](xVal, yVal Array) (Array, error) {
	x, y := toArray[T](xVal), toArray[T](yVal).values[0]
	z := make([]T, x.shape.Size())
	for i, xi := range x.values {
		z[i] = xi % y
	}
	return ToIntegerArray[T](z, x.shape.Dimensions), nil
}

func shlArrayToAtomic[T gotype.Integer](xVal, yVal Array) (Array, error) {
	x, y := toArray[T](xVal), toArray[T](yVal).values[0]
	z := make([]T, x.shape.Size())
	for i, xi := range x.values {
		z[i] = xi << y
	}
	return ToIntegerArray[T](z, x.shape.Dimensions), nil
}

func shrArrayToAtomic[T gotype.Integer](xVal, yVal Array) (Array, error) {
	x, y := toArray[T](xVal), toArray[T](yVal).values[0]
	z := make([]T, x.shape.Size())
	for i, xi := range x.values {
		z[i] = xi >> y
	}
	return ToIntegerArray[T](z, x.shape.Dimensions), nil
}

func andArrayToAtomic[T gotype.Integer](xVal, yVal Array) (Array, error) {
	x, y := toArray[T](xVal), toArray[T](yVal).values[0]
	z := make([]T, x.shape.Size())
	for i, xi := range x.values {
		z[i] = xi & y
	}
	return ToIntegerArray[T](z, x.shape.Dimensions), nil
}

func orArrayToAtomic[T gotype.Integer](xVal, yVal Array) (Array, error) {
	x, y := toArray[T](xVal), toArray[T](yVal).values[0]
	z := make([]T, x.shape.Size())
	for i, xi := range x.values {
		z[i] = xi | y
	}
	return ToIntegerArray[T](z, x.shape.Dimensions), nil
}

func xorArrayToAtomic[T gotype.Integer](xVal, yVal Array) (Array, error) {
	x, y := toArray[T](xVal), toArray[T](yVal).values[0]
	z := make([]T, x.shape.Size())
	for i, xi := range x.values {
		z[i] = xi ^ y
	}
	return ToIntegerArray[T](z, x.shape.Dimensions), nil
}

func equalArrayToAtomic[T gotype.NumericNotComplex](xVal, yVal Array) (Array, error) {
	x, y := toArray[T](xVal), toArray[T](yVal).values[0]
	z := make([]bool, x.shape.Size())
	for i, xi := range x.values {
		z[i] = xi == y
	}
	return ToBoolArray(z, x.shape.Dimensions), nil
}

// ArrayToArray

func subArrayToArray[T gotype.NumericNotComplex](xVal, yVal Array) (Array, error) {
	x, y := toArray[T](xVal), toArray[T](yVal)
	z := make([]T, y.shape.Size())
	for i, xi := range x.values {
		z[i] = xi - y.values[i]
	}
	return ToAlgebraicArray[T](z, x.shape.Dimensions), nil
}

func addArrayToArray[T gotype.NumericNotComplex](xVal, yVal Array) (Array, error) {
	x, y := toArray[T](xVal), toArray[T](yVal)
	z := make([]T, y.shape.Size())
	for i, xi := range x.values {
		z[i] = xi + y.values[i]
	}
	return ToAlgebraicArray[T](z, x.shape.Dimensions), nil
}

func mulArrayToArray[T gotype.NumericNotComplex](xVal, yVal Array) (Array, error) {
	x, y := toArray[T](xVal), toArray[T](yVal)
	z := make([]T, y.shape.Size())
	for i, xi := range x.values {
		z[i] = xi * y.values[i]
	}
	return ToAlgebraicArray[T](z, x.shape.Dimensions), nil
}

func quoArrayToArray[T gotype.NumericNotComplex](xVal, yVal Array) (Array, error) {
	x, y := toArray[T](xVal), toArray[T](yVal)
	z := make([]T, y.shape.Size())
	for i, xi := range x.values {
		z[i] = xi / y.values[i]
	}
	return ToAlgebraicArray[T](z, x.shape.Dimensions), nil
}

func remArrayToArray[T gotype.Integer](xVal, yVal Array) (Array, error) {
	x, y := toArray[T](xVal), toArray[T](yVal)
	z := make([]T, y.shape.Size())
	for i, xi := range x.values {
		z[i] = xi % y.values[i]
	}
	return ToIntegerArray[T](z, x.shape.Dimensions), nil
}

func shlArrayToArray[T gotype.Integer](xVal, yVal Array) (Array, error) {
	x, y := toArray[T](xVal), toArray[T](yVal)
	z := make([]T, y.shape.Size())
	for i, xi := range x.values {
		z[i] = xi << y.values[i]
	}
	return ToIntegerArray[T](z, x.shape.Dimensions), nil
}

func shrArrayToArray[T gotype.Integer](xVal, yVal Array) (Array, error) {
	x, y := toArray[T](xVal), toArray[T](yVal)
	z := make([]T, y.shape.Size())
	for i, xi := range x.values {
		z[i] = xi >> y.values[i]
	}
	return ToIntegerArray[T](z, x.shape.Dimensions), nil
}

func andArrayToArray[T gotype.Integer](xVal, yVal Array) (Array, error) {
	x, y := toArray[T](xVal), toArray[T](yVal)
	z := make([]T, y.shape.Size())
	for i, xi := range x.values {
		z[i] = xi & y.values[i]
	}
	return ToIntegerArray[T](z, x.shape.Dimensions), nil
}

func orArrayToArray[T gotype.Integer](xVal, yVal Array) (Array, error) {
	x, y := toArray[T](xVal), toArray[T](yVal)
	z := make([]T, y.shape.Size())
	for i, xi := range x.values {
		z[i] = xi | y.values[i]
	}
	return ToIntegerArray[T](z, x.shape.Dimensions), nil
}

func xorArrayToArray[T gotype.Integer](xVal, yVal Array) (Array, error) {
	x, y := toArray[T](xVal), toArray[T](yVal)
	z := make([]T, y.shape.Size())
	for i, xi := range x.values {
		z[i] = xi ^ y.values[i]
	}
	return ToIntegerArray[T](z, x.shape.Dimensions), nil
}

func equalArrayToArray[T gotype.NumericNotComplex](xVal, yVal Array) (Array, error) {
	x, y := toArray[T](xVal), toArray[T](yVal)
	z := make([]bool, y.shape.Size())
	for i, xi := range x.values {
		z[i] = xi == y.values[i]
	}
	return ToBoolArray(z, x.shape.Dimensions), nil
}

// Unary Operators

func subArray[T gotype.NumericNotComplex](xVal Array) (Array, error) {
	x := toArray[T](xVal)
	z := make([]T, x.shape.Size())
	for i, xi := range x.values {
		z[i] = -xi
	}
	return ToAlgebraicArray[T](z, x.shape.Dimensions), nil
}

func castArrayWithShape[T gotype.NumericNotComplex, U gotype.NumericNotComplex](xVal Array, dims []int) (Array, error) {
	x := toArray[T](xVal)
	z := make([]U, x.shape.Size())
	for i, xi := range x.values {
		z[i] = U(xi)
	}
	return ToAlgebraicArray[U](z, dims), nil
}

// N-ary Operators

func concat[T gotype.Supported](xVals []Array) (Array, error) {
	xVal0 := xVals[0]
	z := make([]T, len(xVals))
	for i, x := range xVals {
		z[i] = toArray[T](x).values[0]
	}
	return xVal0.newArray(&arrayT[T]{
		factory: xVal0.Factory(),
		shape:   shapes.Make(xVal0.Shape().DType, len(xVals)),
		values:  z,
	}), nil
}
