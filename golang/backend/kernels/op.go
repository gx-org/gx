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
	"github.com/gx-org/backend/dtype"
	"github.com/gx-org/backend/shape"
)

// AtomicToAtomic

func addAtomicToAtomic[T goAlgebra](xVal, yVal Array) (Array, error) {
	x, y := toArray[T](xVal), toArray[T](yVal)
	return ToAlgebraicArray[T]([]T{x.values[0] + y.values[0]}, nil), nil
}

func subAtomicToAtomic[T goAlgebra](xVal, yVal Array) (Array, error) {
	x, y := toArray[T](xVal), toArray[T](yVal)
	return ToAlgebraicArray[T]([]T{x.values[0] - y.values[0]}, nil), nil
}

func mulAtomicToAtomic[T goAlgebra](xVal, yVal Array) (Array, error) {
	x, y := toArray[T](xVal), toArray[T](yVal)
	return ToAlgebraicArray[T]([]T{x.values[0] * y.values[0]}, nil), nil
}

func quoAtomicToAtomic[T goAlgebra](xVal, yVal Array) (Array, error) {
	x, y := toArray[T](xVal), toArray[T](yVal)
	return ToAlgebraicArray[T]([]T{x.values[0] / y.values[0]}, nil), nil
}

func remAtomicToAtomic[T dtype.IntegerType](xVal, yVal Array) (Array, error) {
	x, y := toArray[T](xVal), toArray[T](yVal)
	return ToIntegerArray[T]([]T{x.values[0] % y.values[0]}, nil), nil
}

func shlAtomicToAtomic[T dtype.IntegerType](xVal, yVal Array) (Array, error) {
	x, y := toArray[T](xVal), toArray[T](yVal)
	return ToIntegerArray[T]([]T{x.values[0] << y.values[0]}, nil), nil
}

func shrAtomicToAtomic[T dtype.IntegerType](xVal, yVal Array) (Array, error) {
	x, y := toArray[T](xVal), toArray[T](yVal)
	return ToIntegerArray[T]([]T{x.values[0] >> y.values[0]}, nil), nil
}

func andAtomicToAtomic[T dtype.IntegerType](xVal, yVal Array) (Array, error) {
	x, y := toArray[T](xVal), toArray[T](yVal)
	return ToIntegerArray[T]([]T{x.values[0] & y.values[0]}, nil), nil
}

func orAtomicToAtomic[T dtype.IntegerType](xVal, yVal Array) (Array, error) {
	x, y := toArray[T](xVal), toArray[T](yVal)
	return ToIntegerArray[T]([]T{x.values[0] | y.values[0]}, nil), nil
}

func xorAtomicToAtomic[T dtype.IntegerType](xVal, yVal Array) (Array, error) {
	x, y := toArray[T](xVal), toArray[T](yVal)
	return ToIntegerArray[T]([]T{x.values[0] ^ y.values[0]}, nil), nil
}

func equalAtomicToAtomic[T dtype.AlgebraType](xVal, yVal Array) (Array, error) {
	x, y := toArray[T](xVal), toArray[T](yVal)
	return ToBoolArray([]bool{x.values[0] == y.values[0]}, nil), nil
}

func notEqualAtomicToAtomic[T dtype.AlgebraType](xVal, yVal Array) (Array, error) {
	x, y := toArray[T](xVal), toArray[T](yVal)
	return ToBoolArray([]bool{x.values[0] != y.values[0]}, nil), nil
}

// AtomicToArray

func addAtomicToArray[T goAlgebra](xVal, yVal Array) (Array, error) {
	x, y := toArray[T](xVal).values[0], toArray[T](yVal)
	z := make([]T, y.shape.Size())
	for i, yi := range y.values {
		z[i] = x + yi
	}
	return ToAlgebraicArray[T](z, y.shape.AxisLengths), nil
}

func subAtomicToArray[T goAlgebra](xVal, yVal Array) (Array, error) {
	x, y := toArray[T](xVal).values[0], toArray[T](yVal)
	z := make([]T, y.shape.Size())
	for i, yi := range y.values {
		z[i] = x - yi
	}
	return ToAlgebraicArray[T](z, y.shape.AxisLengths), nil
}

func mulAtomicToArray[T goAlgebra](xVal, yVal Array) (Array, error) {
	x, y := toArray[T](xVal).values[0], toArray[T](yVal)
	z := make([]T, y.shape.Size())
	for i, yi := range y.values {
		z[i] = x * yi
	}
	return ToAlgebraicArray[T](z, y.shape.AxisLengths), nil
}

func quoAtomicToArray[T goAlgebra](xVal, yVal Array) (Array, error) {
	x, y := toArray[T](xVal).values[0], toArray[T](yVal)
	z := make([]T, y.shape.Size())
	for i, yi := range y.values {
		z[i] = x / yi
	}
	return ToAlgebraicArray[T](z, y.shape.AxisLengths), nil
}

func remAtomicToArray[T dtype.IntegerType](xVal, yVal Array) (Array, error) {
	x, y := toArray[T](xVal).values[0], toArray[T](yVal)
	z := make([]T, y.shape.Size())
	for i, yi := range y.values {
		z[i] = x % yi
	}
	return ToIntegerArray[T](z, y.shape.AxisLengths), nil
}

func shlAtomicToArray[T dtype.IntegerType](xVal, yVal Array) (Array, error) {
	x, y := toArray[T](xVal).values[0], toArray[T](yVal)
	z := make([]T, y.shape.Size())
	for i, yi := range y.values {
		z[i] = x << yi
	}
	return ToIntegerArray[T](z, y.shape.AxisLengths), nil
}

func shrAtomicToArray[T dtype.IntegerType](xVal, yVal Array) (Array, error) {
	x, y := toArray[T](xVal).values[0], toArray[T](yVal)
	z := make([]T, y.shape.Size())
	for i, yi := range y.values {
		z[i] = x >> yi
	}
	return ToIntegerArray[T](z, y.shape.AxisLengths), nil
}

func andAtomicToArray[T dtype.IntegerType](xVal, yVal Array) (Array, error) {
	x, y := toArray[T](xVal).values[0], toArray[T](yVal)
	z := make([]T, y.shape.Size())
	for i, yi := range y.values {
		z[i] = x & yi
	}
	return ToIntegerArray[T](z, y.shape.AxisLengths), nil
}

func orAtomicToArray[T dtype.IntegerType](xVal, yVal Array) (Array, error) {
	x, y := toArray[T](xVal).values[0], toArray[T](yVal)
	z := make([]T, y.shape.Size())
	for i, yi := range y.values {
		z[i] = x | yi
	}
	return ToIntegerArray[T](z, y.shape.AxisLengths), nil
}

func xorAtomicToArray[T dtype.IntegerType](xVal, yVal Array) (Array, error) {
	x, y := toArray[T](xVal).values[0], toArray[T](yVal)
	z := make([]T, y.shape.Size())
	for i, yi := range y.values {
		z[i] = x ^ yi
	}
	return ToIntegerArray[T](z, y.shape.AxisLengths), nil
}

func equalAtomicToArray[T dtype.AlgebraType](xVal, yVal Array) (Array, error) {
	x, y := toArray[T](xVal).values[0], toArray[T](yVal)
	z := make([]bool, y.shape.Size())
	for i, yi := range y.values {
		z[i] = x == yi
	}
	return ToBoolArray(z, y.shape.AxisLengths), nil
}

// ArrayToAtomic

func addArrayToAtomic[T goAlgebra](xVal, yVal Array) (Array, error) {
	return addAtomicToArray[T](yVal, xVal)
}

func subArrayToAtomic[T goAlgebra](xVal, yVal Array) (Array, error) {
	x, y := toArray[T](xVal), toArray[T](yVal).values[0]
	z := make([]T, x.shape.Size())
	for i, xi := range x.values {
		z[i] = xi - y
	}
	return ToAlgebraicArray[T](z, x.shape.AxisLengths), nil
}

func mulArrayToAtomic[T goAlgebra](xVal, yVal Array) (Array, error) {
	return mulAtomicToArray[T](yVal, xVal)
}

func quoArrayToAtomic[T goAlgebra](xVal, yVal Array) (Array, error) {
	x, y := toArray[T](xVal), toArray[T](yVal).values[0]
	z := make([]T, x.shape.Size())
	for i, xi := range x.values {
		z[i] = xi / y
	}
	return ToAlgebraicArray[T](z, x.shape.AxisLengths), nil
}

func remArrayToAtomic[T dtype.IntegerType](xVal, yVal Array) (Array, error) {
	x, y := toArray[T](xVal), toArray[T](yVal).values[0]
	z := make([]T, x.shape.Size())
	for i, xi := range x.values {
		z[i] = xi % y
	}
	return ToIntegerArray[T](z, x.shape.AxisLengths), nil
}

func shlArrayToAtomic[T dtype.IntegerType](xVal, yVal Array) (Array, error) {
	x, y := toArray[T](xVal), toArray[T](yVal).values[0]
	z := make([]T, x.shape.Size())
	for i, xi := range x.values {
		z[i] = xi << y
	}
	return ToIntegerArray[T](z, x.shape.AxisLengths), nil
}

func shrArrayToAtomic[T dtype.IntegerType](xVal, yVal Array) (Array, error) {
	x, y := toArray[T](xVal), toArray[T](yVal).values[0]
	z := make([]T, x.shape.Size())
	for i, xi := range x.values {
		z[i] = xi >> y
	}
	return ToIntegerArray[T](z, x.shape.AxisLengths), nil
}

func andArrayToAtomic[T dtype.IntegerType](xVal, yVal Array) (Array, error) {
	x, y := toArray[T](xVal), toArray[T](yVal).values[0]
	z := make([]T, x.shape.Size())
	for i, xi := range x.values {
		z[i] = xi & y
	}
	return ToIntegerArray[T](z, x.shape.AxisLengths), nil
}

func orArrayToAtomic[T dtype.IntegerType](xVal, yVal Array) (Array, error) {
	x, y := toArray[T](xVal), toArray[T](yVal).values[0]
	z := make([]T, x.shape.Size())
	for i, xi := range x.values {
		z[i] = xi | y
	}
	return ToIntegerArray[T](z, x.shape.AxisLengths), nil
}

func xorArrayToAtomic[T dtype.IntegerType](xVal, yVal Array) (Array, error) {
	x, y := toArray[T](xVal), toArray[T](yVal).values[0]
	z := make([]T, x.shape.Size())
	for i, xi := range x.values {
		z[i] = xi ^ y
	}
	return ToIntegerArray[T](z, x.shape.AxisLengths), nil
}

func equalArrayToAtomic[T dtype.AlgebraType](xVal, yVal Array) (Array, error) {
	x, y := toArray[T](xVal), toArray[T](yVal).values[0]
	z := make([]bool, x.shape.Size())
	for i, xi := range x.values {
		z[i] = xi == y
	}
	return ToBoolArray(z, x.shape.AxisLengths), nil
}

// ArrayToArray

func subArrayToArray[T goAlgebra](xVal, yVal Array) (Array, error) {
	x, y := toArray[T](xVal), toArray[T](yVal)
	z := make([]T, y.shape.Size())
	for i, xi := range x.values {
		z[i] = xi - y.values[i]
	}
	return ToAlgebraicArray[T](z, x.shape.AxisLengths), nil
}

func addArrayToArray[T goAlgebra](xVal, yVal Array) (Array, error) {
	x, y := toArray[T](xVal), toArray[T](yVal)
	z := make([]T, y.shape.Size())
	for i, xi := range x.values {
		z[i] = xi + y.values[i]
	}
	return ToAlgebraicArray[T](z, x.shape.AxisLengths), nil
}

func mulArrayToArray[T goAlgebra](xVal, yVal Array) (Array, error) {
	x, y := toArray[T](xVal), toArray[T](yVal)
	z := make([]T, y.shape.Size())
	for i, xi := range x.values {
		z[i] = xi * y.values[i]
	}
	return ToAlgebraicArray[T](z, x.shape.AxisLengths), nil
}

func quoArrayToArray[T goAlgebra](xVal, yVal Array) (Array, error) {
	x, y := toArray[T](xVal), toArray[T](yVal)
	z := make([]T, y.shape.Size())
	for i, xi := range x.values {
		z[i] = xi / y.values[i]
	}
	return ToAlgebraicArray[T](z, x.shape.AxisLengths), nil
}

func remArrayToArray[T dtype.IntegerType](xVal, yVal Array) (Array, error) {
	x, y := toArray[T](xVal), toArray[T](yVal)
	z := make([]T, y.shape.Size())
	for i, xi := range x.values {
		z[i] = xi % y.values[i]
	}
	return ToIntegerArray[T](z, x.shape.AxisLengths), nil
}

func shlArrayToArray[T dtype.IntegerType](xVal, yVal Array) (Array, error) {
	x, y := toArray[T](xVal), toArray[T](yVal)
	z := make([]T, y.shape.Size())
	for i, xi := range x.values {
		z[i] = xi << y.values[i]
	}
	return ToIntegerArray[T](z, x.shape.AxisLengths), nil
}

func shrArrayToArray[T dtype.IntegerType](xVal, yVal Array) (Array, error) {
	x, y := toArray[T](xVal), toArray[T](yVal)
	z := make([]T, y.shape.Size())
	for i, xi := range x.values {
		z[i] = xi >> y.values[i]
	}
	return ToIntegerArray[T](z, x.shape.AxisLengths), nil
}

func andArrayToArray[T dtype.IntegerType](xVal, yVal Array) (Array, error) {
	x, y := toArray[T](xVal), toArray[T](yVal)
	z := make([]T, y.shape.Size())
	for i, xi := range x.values {
		z[i] = xi & y.values[i]
	}
	return ToIntegerArray[T](z, x.shape.AxisLengths), nil
}

func orArrayToArray[T dtype.IntegerType](xVal, yVal Array) (Array, error) {
	x, y := toArray[T](xVal), toArray[T](yVal)
	z := make([]T, y.shape.Size())
	for i, xi := range x.values {
		z[i] = xi | y.values[i]
	}
	return ToIntegerArray[T](z, x.shape.AxisLengths), nil
}

func xorArrayToArray[T dtype.IntegerType](xVal, yVal Array) (Array, error) {
	x, y := toArray[T](xVal), toArray[T](yVal)
	z := make([]T, y.shape.Size())
	for i, xi := range x.values {
		z[i] = xi ^ y.values[i]
	}
	return ToIntegerArray[T](z, x.shape.AxisLengths), nil
}

func equalArrayToArray[T dtype.AlgebraType](xVal, yVal Array) (Array, error) {
	x, y := toArray[T](xVal), toArray[T](yVal)
	z := make([]bool, y.shape.Size())
	for i, xi := range x.values {
		z[i] = xi == y.values[i]
	}
	return ToBoolArray(z, x.shape.AxisLengths), nil
}

// Unary Operators

func subArray[T goAlgebra](xVal Array) (Array, error) {
	x := toArray[T](xVal)
	z := make([]T, x.shape.Size())
	for i, xi := range x.values {
		z[i] = -xi
	}
	return ToAlgebraicArray[T](z, x.shape.AxisLengths), nil
}

func castArrayWithShape[T dtype.AlgebraType, U goAlgebra](xVal Array, dims []int) (Array, error) {
	x := toArray[T](xVal)
	z := make([]U, x.shape.Size())
	for i, xi := range x.values {
		z[i] = U(xi)
	}
	return ToAlgebraicArray[U](z, dims), nil
}

// N-ary Operators

func concat[T dtype.GoDataType](xVals []Array) (Array, error) {
	xVal0 := xVals[0]
	z := make([]T, len(xVals))
	for i, x := range xVals {
		z[i] = toArray[T](x).values[0]
	}
	return xVal0.newArray(&arrayT[T]{
		factory: xVal0.Factory(),
		shape: shape.Shape{
			DType:       xVal0.Shape().DType,
			AxisLengths: []int{len(xVals)},
		},
		values: z,
	}), nil
}
