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

func toAlgebraicArray[T dtype.AlgebraType](shape *shape.Shape, values []T) *ArrayT[T] {
	return &ArrayT[T]{
		factory: algebraicFactory[T]{},
		shape:   shape,
		values:  values,
	}
}

func toBoolArray(sh *shape.Shape, values []bool) *ArrayT[bool] {
	return &ArrayT[bool]{
		factory: boolFactory{},
		shape: &shape.Shape{
			DType:       dtype.Bool,
			AxisLengths: sh.AxisLengths,
		},
		values: values,
	}
}

// AtomicToAtomic

func addAtomicToAtomic[T dtype.AlgebraType](xVal, yVal Array) (Array, error) {
	x, y := xVal.(*ArrayT[T]), yVal.(*ArrayT[T])
	return toAlgebraicArray[T](x.shape, []T{x.values[0] + y.values[0]}), nil
}

func subAtomicToAtomic[T dtype.AlgebraType](xVal, yVal Array) (Array, error) {
	x, y := xVal.(*ArrayT[T]), yVal.(*ArrayT[T])
	return toAlgebraicArray[T](x.shape, []T{x.values[0] - y.values[0]}), nil
}

func mulAtomicToAtomic[T dtype.AlgebraType](xVal, yVal Array) (Array, error) {
	x, y := xVal.(*ArrayT[T]), yVal.(*ArrayT[T])
	return toAlgebraicArray[T](x.shape, []T{x.values[0] * y.values[0]}), nil
}

func quoAtomicToAtomic[T dtype.AlgebraType](xVal, yVal Array) (Array, error) {
	x, y := xVal.(*ArrayT[T]), yVal.(*ArrayT[T])
	return toAlgebraicArray[T](x.shape, []T{x.values[0] / y.values[0]}), nil
}

func equalAtomicToAtomic[T dtype.AlgebraType](xVal, yVal Array) (Array, error) {
	x, y := xVal.(*ArrayT[T]), yVal.(*ArrayT[T])
	return toBoolArray(&shape.Shape{
		DType: dtype.Bool,
	}, []bool{x.values[0] == y.values[0]}), nil
}

// AtomicToArray

func addAtomicToArray[T dtype.AlgebraType](xVal, yVal Array) (Array, error) {
	x, y := xVal.(*ArrayT[T]).values[0], yVal.(*ArrayT[T])
	z := make([]T, y.shape.Size())
	for i, yi := range y.values {
		z[i] = x + yi
	}
	return toAlgebraicArray[T](y.shape, z), nil
}

func subAtomicToArray[T dtype.AlgebraType](xVal, yVal Array) (Array, error) {
	x, y := xVal.(*ArrayT[T]).values[0], yVal.(*ArrayT[T])
	z := make([]T, y.shape.Size())
	for i, yi := range y.values {
		z[i] = x - yi
	}
	return toAlgebraicArray[T](y.shape, z), nil
}

func mulAtomicToArray[T dtype.AlgebraType](xVal, yVal Array) (Array, error) {
	x, y := xVal.(*ArrayT[T]).values[0], yVal.(*ArrayT[T])
	z := make([]T, y.shape.Size())
	for i, yi := range y.values {
		z[i] = x * yi
	}
	return toAlgebraicArray[T](y.shape, z), nil
}

func quoAtomicToArray[T dtype.AlgebraType](xVal, yVal Array) (Array, error) {
	x, y := xVal.(*ArrayT[T]).values[0], yVal.(*ArrayT[T])
	z := make([]T, y.shape.Size())
	for i, yi := range y.values {
		z[i] = x / yi
	}
	return toAlgebraicArray[T](y.shape, z), nil
}

func equalAtomicToArray[T dtype.AlgebraType](xVal, yVal Array) (Array, error) {
	x, y := xVal.(*ArrayT[T]).values[0], yVal.(*ArrayT[T])
	z := make([]bool, y.shape.Size())
	for i, yi := range y.values {
		z[i] = x == yi
	}
	return toBoolArray(y.shape, z), nil
}

// ArrayToAtomic

func addArrayToAtomic[T dtype.AlgebraType](xVal, yVal Array) (Array, error) {
	return addAtomicToArray[T](yVal, xVal)
}

func subArrayToAtomic[T dtype.AlgebraType](xVal, yVal Array) (Array, error) {
	x, y := xVal.(*ArrayT[T]), yVal.(*ArrayT[T]).values[0]
	z := make([]T, x.shape.Size())
	for i, xi := range x.values {
		z[i] = xi - y
	}
	return toAlgebraicArray[T](x.shape, z), nil
}

func mulArrayToAtomic[T dtype.AlgebraType](xVal, yVal Array) (Array, error) {
	return mulAtomicToArray[T](yVal, xVal)
}

func quoArrayToAtomic[T dtype.AlgebraType](xVal, yVal Array) (Array, error) {
	x, y := xVal.(*ArrayT[T]), yVal.(*ArrayT[T]).values[0]
	z := make([]T, x.shape.Size())
	for i, xi := range x.values {
		z[i] = xi / y
	}
	return toAlgebraicArray[T](x.shape, z), nil
}

func equalArrayToAtomic[T dtype.AlgebraType](xVal, yVal Array) (Array, error) {
	x, y := xVal.(*ArrayT[T]), yVal.(*ArrayT[T]).values[0]
	z := make([]bool, x.shape.Size())
	for i, xi := range x.values {
		z[i] = xi == y
	}
	return toBoolArray(x.shape, z), nil
}

// ArrayToArray

func subArrayToArray[T dtype.AlgebraType](xVal, yVal Array) (Array, error) {
	x, y := xVal.(*ArrayT[T]), yVal.(*ArrayT[T])
	z := make([]T, y.shape.Size())
	for i, xi := range x.values {
		z[i] = xi - y.values[i]
	}
	return toAlgebraicArray[T](x.shape, z), nil
}

func addArrayToArray[T dtype.AlgebraType](xVal, yVal Array) (Array, error) {
	x, y := xVal.(*ArrayT[T]), yVal.(*ArrayT[T])
	z := make([]T, y.shape.Size())
	for i, xi := range x.values {
		z[i] = xi + y.values[i]
	}
	return toAlgebraicArray[T](x.shape, z), nil
}

func mulArrayToArray[T dtype.AlgebraType](xVal, yVal Array) (Array, error) {
	x, y := xVal.(*ArrayT[T]), yVal.(*ArrayT[T])
	z := make([]T, y.shape.Size())
	for i, xi := range x.values {
		z[i] = xi * y.values[i]
	}
	return toAlgebraicArray[T](x.shape, z), nil
}

func quoArrayToArray[T dtype.AlgebraType](xVal, yVal Array) (Array, error) {
	x, y := xVal.(*ArrayT[T]), yVal.(*ArrayT[T])
	z := make([]T, y.shape.Size())
	for i, xi := range x.values {
		z[i] = xi / y.values[i]
	}
	return toAlgebraicArray[T](x.shape, z), nil
}

func equalArrayToArray[T dtype.AlgebraType](xVal, yVal Array) (Array, error) {
	x, y := xVal.(*ArrayT[T]), yVal.(*ArrayT[T])
	z := make([]bool, y.shape.Size())
	for i, xi := range x.values {
		z[i] = xi == y.values[i]
	}
	return toBoolArray(x.shape, z), nil
}

// Unary Operators

func subArray[T dtype.AlgebraType](xVal Array) (Array, error) {
	x := xVal.(*ArrayT[T])
	z := make([]T, x.shape.Size())
	for i, xi := range x.values {
		z[i] = -xi
	}
	return toAlgebraicArray[T](x.shape, z), nil
}

func castArray[T dtype.AlgebraType, U dtype.AlgebraType](xVal Array) (Array, error) {
	x := xVal.(*ArrayT[T])
	z := make([]U, x.shape.Size())
	for i, xi := range x.values {
		z[i] = U(xi)
	}
	return toAlgebraicArray[U](x.shape, z), nil
}

// N-ary Operators

func concat[T dtype.GoDataType](xVals []Array) (Array, error) {
	xVal0 := xVals[0]
	z := make([]T, len(xVals))
	for i, x := range xVals {
		z[i] = x.(*ArrayT[T]).values[0]
	}
	return &ArrayT[T]{
		factory: xVal0.Factory(),
		shape: &shape.Shape{
			DType:       xVal0.Shape().DType,
			AxisLengths: []int{len(xVals)},
		},
		values: z,
	}, nil
}
