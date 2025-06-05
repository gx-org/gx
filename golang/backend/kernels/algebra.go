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
	"go/token"

	"github.com/pkg/errors"
	"github.com/gx-org/backend/dtype"
	"github.com/gx-org/backend/shape"
	"github.com/gx-org/gx/build/ir"
)

type algebraicFactory[T goAlgebra] struct {
	arrayFactory[T]
	mathFactory[T]
}

var _ Factory = (*algebraicFactory[ir.Int])(nil)

func (f algebraicFactory[T]) Math() MathFactory {
	return f.mathFactory
}

// BinaryOp creates a new kernel for a binary operator.
func (algebraicFactory[T]) BinaryOp(op token.Token, x, y *shape.Shape) (Binary, *shape.Shape, error) {
	xAtomic := isAtomic(x)
	yAtomic := isAtomic(y)
	if xAtomic && yAtomic {
		out := &shape.Shape{DType: x.DType}
		switch op {
		case token.ADD:
			return addAtomicToAtomic[T], out, nil
		case token.SUB:
			return subAtomicToAtomic[T], out, nil
		case token.MUL:
			return mulAtomicToAtomic[T], out, nil
		case token.QUO:
			return quoAtomicToAtomic[T], out, nil
		case token.EQL:
			out.DType = dtype.Bool
			return equalAtomicToAtomic[T], out, nil
		case token.NEQ:
			out.DType = dtype.Bool
			return notEqualAtomicToAtomic[T], out, nil
		default:
			return nil, nil, errors.Errorf("operator %s not supported for %s atomic-atomic", op.String(), x.DType.String())
		}
	}
	if xAtomic {
		out := &shape.Shape{DType: x.DType, AxisLengths: y.AxisLengths}
		switch op {
		case token.ADD:
			return addAtomicToArray[T], out, nil
		case token.SUB:
			return subAtomicToArray[T], out, nil
		case token.MUL:
			return mulAtomicToArray[T], out, nil
		case token.QUO:
			return quoAtomicToArray[T], out, nil
		case token.EQL:
			out.DType = dtype.Bool
			return equalAtomicToArray[T], out, nil
		default:
			return nil, nil, errors.Errorf("operator %s not supported for %s atomic-array", op.String(), x.DType.String())
		}
	}
	if yAtomic {
		out := &shape.Shape{DType: x.DType, AxisLengths: x.AxisLengths}
		switch op {
		case token.ADD:
			return addArrayToAtomic[T], out, nil
		case token.SUB:
			return subArrayToAtomic[T], out, nil
		case token.MUL:
			return mulArrayToAtomic[T], out, nil
		case token.QUO:
			return quoArrayToAtomic[T], out, nil
		case token.EQL:
			out.DType = dtype.Bool
			return equalArrayToAtomic[T], out, nil
		default:
			return nil, nil, errors.Errorf("operator %s not supported for %s array-atomic", op.String(), x.DType.String())
		}
	}
	out := &shape.Shape{DType: x.DType, AxisLengths: y.AxisLengths}
	switch op {
	case token.ADD:
		return addArrayToArray[T], out, nil
	case token.SUB:
		return subArrayToArray[T], out, nil
	case token.MUL:
		return mulArrayToArray[T], out, nil
	case token.QUO:
		return quoArrayToArray[T], out, nil
	case token.EQL:
		out.DType = dtype.Bool
		return equalArrayToArray[T], out, nil
	default:
		return nil, nil, errors.Errorf("operator %s not supported for %s array-array", op.String(), x.DType.String())
	}
}

// UnaryOp creates a new kernel for a unary operator.
func (algebraicFactory[T]) UnaryOp(op token.Token, x *shape.Shape) (Unary, *shape.Shape, error) {
	switch op {
	case token.SUB:
		return subArray[T], x, nil
	default:
		return nil, nil, errors.Errorf("operator %q supported", op.String())
	}
}

func castArray[T dtype.AlgebraType, U goAlgebra](dims []int) Unary {
	return func(x Array) (Array, error) {
		return castArrayWithShape[T, U](x, dims)
	}
}

func (algebraicFactory[T]) Cast(kind dtype.DataType, dims []int) (Unary, *shape.Shape, Factory, error) {
	shap := &shape.Shape{
		DType:       dtype.DataType(kind),
		AxisLengths: dims,
	}
	switch kind {
	case dtype.Bfloat16:
		return castToBfloat16Array[T], shap, bfloat16Factory{}, nil
	case dtype.Float32:
		return castArray[T, float32](dims), shap, algebraicFactory[float32]{}, nil
	case dtype.Float64:
		return castArray[T, float64](dims), shap, algebraicFactory[float64]{}, nil
	case dtype.Int32:
		return castArray[T, int32](dims), shap, integerFactory[int32]{}, nil
	case dtype.Int64:
		return castArray[T, int64](dims), shap, integerFactory[int64]{}, nil
	case dtype.Uint32:
		return castArray[T, uint32](dims), shap, integerFactory[uint32]{}, nil
	case dtype.Uint64:
		return castArray[T, uint64](dims), shap, integerFactory[uint64]{}, nil
	default:
		return nil, nil, nil, errors.Errorf("cast to %v not supported", kind)
	}
}

type integerFactory[T dtype.IntegerType] struct {
	algebraicFactory[T]
}

// BinaryOp creates a new kernel for a binary operator.
func (f integerFactory[T]) BinaryOp(op token.Token, x, y *shape.Shape) (Binary, *shape.Shape, error) {
	xAtomic := isAtomic(x)
	yAtomic := isAtomic(y)
	if xAtomic && yAtomic {
		out := &shape.Shape{DType: x.DType}
		switch op {
		case token.REM:
			return remAtomicToAtomic[T], out, nil
		case token.SHL:
			return shlAtomicToAtomic[T], out, nil
		case token.SHR:
			return shrAtomicToAtomic[T], out, nil
		case token.AND:
			return andAtomicToAtomic[T], out, nil
		case token.OR:
			return orAtomicToAtomic[T], out, nil
		case token.XOR:
			return xorAtomicToAtomic[T], out, nil
		}
		return f.algebraicFactory.BinaryOp(op, x, y)
	}
	if xAtomic {
		out := &shape.Shape{DType: x.DType, AxisLengths: y.AxisLengths}
		switch op {
		case token.REM:
			return remAtomicToArray[T], out, nil
		case token.SHL:
			return shlAtomicToArray[T], out, nil
		case token.SHR:
			return shrAtomicToArray[T], out, nil
		case token.AND:
			return andAtomicToArray[T], out, nil
		case token.OR:
			return orAtomicToArray[T], out, nil
		case token.XOR:
			return xorAtomicToArray[T], out, nil
		}
		return f.algebraicFactory.BinaryOp(op, x, y)
	}
	if yAtomic {
		out := &shape.Shape{DType: x.DType, AxisLengths: x.AxisLengths}
		switch op {
		case token.REM:
			return remArrayToAtomic[T], out, nil
		case token.SHL:
			return shlArrayToAtomic[T], out, nil
		case token.SHR:
			return shrArrayToAtomic[T], out, nil
		case token.AND:
			return andArrayToAtomic[T], out, nil
		case token.OR:
			return orArrayToAtomic[T], out, nil
		case token.XOR:
			return xorArrayToAtomic[T], out, nil
		}
		return f.algebraicFactory.BinaryOp(op, x, y)
	}
	out := &shape.Shape{DType: x.DType, AxisLengths: y.AxisLengths}
	switch op {
	case token.REM:
		return remArrayToArray[T], out, nil
	case token.SHL:
		return shlArrayToArray[T], out, nil
	case token.SHR:
		return shrArrayToArray[T], out, nil
	case token.AND:
		return andArrayToArray[T], out, nil
	case token.OR:
		return orArrayToArray[T], out, nil
	case token.XOR:
		return xorArrayToArray[T], out, nil
	}
	return f.algebraicFactory.BinaryOp(op, x, y)
}

type bfloat16Factory struct {
	arrayFactory[dtype.Bfloat16T]
}

var _ Factory = (*bfloat16Factory)(nil)

func (bfloat16Factory) Math() MathFactory {
	return nil
}

func (bfloat16Factory) BinaryOp(op token.Token, x, y *shape.Shape) (Binary, *shape.Shape, error) {
	return nil, nil, errors.Errorf("operator %q not supported", op.String())
}

// UnaryOp creates a new kernel for a unary operator.
func (bfloat16Factory) UnaryOp(op token.Token, x *shape.Shape) (Unary, *shape.Shape, error) {
	return nil, nil, errors.Errorf("operator %q not supported", op.String())
}

// Cast an array to another
func (f bfloat16Factory) Cast(kind dtype.DataType, dims []int) (Unary, *shape.Shape, Factory, error) {
	shap := &shape.Shape{
		DType:       dtype.DataType(kind),
		AxisLengths: dims,
	}
	switch kind {
	case dtype.Bfloat16:
		return func(a Array) (Array, error) { return a, nil }, shap, f, nil
	case dtype.Float32:
		return castFromBfloat16Array[float32], shap, algebraicFactory[float32]{}, nil
	case dtype.Float64:
		return castFromBfloat16Array[float64], shap, algebraicFactory[float64]{}, nil
	case dtype.Int32:
		return castFromBfloat16Array[int32], shap, algebraicFactory[int32]{}, nil
	case dtype.Int64:
		return castFromBfloat16Array[int64], shap, algebraicFactory[int64]{}, nil
	default:
		return nil, nil, nil, errors.Errorf("cast to %v not supported", kind)
	}
}

func castToBfloat16Array[T dtype.AlgebraType](xVal Array) (Array, error) {
	x := toArray[T](xVal)
	z := make([]dtype.Bfloat16T, x.shape.Size())
	for i, xi := range x.values {
		z[i] = dtype.BFloat16FromFloat32((float32)(xi))
	}
	return ToBfloat16Array(z, x.shape.AxisLengths), nil
}

func castFromBfloat16Array[U goAlgebra](xVal Array) (Array, error) {
	x := toArray[dtype.Bfloat16T](xVal)
	z := make([]U, x.shape.Size())
	for i, xi := range x.values {
		z[i] = U(xi.Float32())
	}
	return ToAlgebraicArray[U](z, x.shape.AxisLengths), nil
}

// ToBfloat16Atom converts a value into an atom owned by a backend.
func ToBfloat16Atom(val dtype.Bfloat16T) Array {
	return ToBfloat16Array([]dtype.Bfloat16T{val}, nil)
}

// ToFloatAtom converts a value into an atom owned by a backend.
func ToFloatAtom[T dtype.Float](val T) Array {
	return ToFloatArray([]T{val}, nil)
}

// ToIntegerAtom converts a value into an atom owned by a backend.
func ToIntegerAtom[T dtype.IntegerType](val T) Array {
	return ToIntegerArray([]T{val}, nil)
}

// ToBfloat16Array converts values and a shape into a native multi-dimensional array owned by a backend.
func ToBfloat16Array(values []dtype.Bfloat16T, dims []int) Array {
	arr := &bfloat16Array{&arrayT[dtype.Bfloat16T]{
		factory: bfloat16Factory{},
		shape: shape.Shape{
			DType:       dtype.Bfloat16,
			AxisLengths: dims,
		},
		values: values,
	}}
	if len(values) != arr.shape.Size() {
		panic(fmt.Sprintf("mismatch between the number of values (=%d) and the number of elements (=%d) in shape %s", len(values), arr.shape.Size(), arr.shape.String()))
	}
	return arr
}

// ToAlgebraicArray converts values and a shape into a native multi-dimensional array owned by a backend.
func ToAlgebraicArray[T dtype.Float | dtype.IntegerType](values []T, dims []int) Array {
	arr := &algebraArray[T]{&arrayT[T]{
		factory: algebraicFactory[T]{},
		shape: shape.Shape{
			DType:       dtype.Generic[T](),
			AxisLengths: dims,
		},
		values: values,
	}}
	if len(values) != arr.shape.Size() {
		panic(fmt.Sprintf("mismatch between the number of values (=%d) and the number of elements (=%d) in shape %s", len(values), arr.shape.Size(), arr.shape.String()))
	}
	return arr
}

// ToFloatArray converts values and a shape into a native multi-dimensional array owned by a backend.
func ToFloatArray[T dtype.Float](values []T, dims []int) Array {
	return ToAlgebraicArray(values, dims)
}

// ToIntegerArray converts values and a shape into a native multi-dimensional array owned by a backend.
func ToIntegerArray[T dtype.IntegerType](values []T, dims []int) Array {
	arr := &algebraArray[T]{&arrayT[T]{
		factory: integerFactory[T]{},
		shape: shape.Shape{
			DType:       dtype.Generic[T](),
			AxisLengths: dims,
		},
		values: values,
	}}
	if len(values) != arr.shape.Size() {
		panic(fmt.Sprintf("mismatch between the number of values (=%d) and the number of elements (=%d) in shape %s", len(values), arr.shape.Size(), arr.shape.String()))
	}
	return arr
}

// Zero returns an array of zeros given a shape.
func Zero(sh *shape.Shape) (Array, error) {
	switch sh.DType {
	case dtype.Bool:
		return ToBoolArray(make([]bool, sh.Size()), sh.AxisLengths), nil
	case dtype.Bfloat16:
		return ToBfloat16Array(make([]dtype.Bfloat16T, sh.Size()), sh.AxisLengths), nil
	case dtype.Float32:
		return ToFloatArray(make([]float32, sh.Size()), sh.AxisLengths), nil
	case dtype.Float64:
		return ToFloatArray(make([]float64, sh.Size()), sh.AxisLengths), nil
	case dtype.Int32:
		return ToIntegerArray(make([]int32, sh.Size()), sh.AxisLengths), nil
	case dtype.Int64:
		return ToIntegerArray(make([]int64, sh.Size()), sh.AxisLengths), nil
	case dtype.Uint32:
		return ToIntegerArray(make([]uint32, sh.Size()), sh.AxisLengths), nil
	case dtype.Uint64:
		return ToIntegerArray(make([]uint64, sh.Size()), sh.AxisLengths), nil
	default:
		return nil, errors.Errorf("cannot an array of data type %s: not supported", sh.DType)
	}

}
