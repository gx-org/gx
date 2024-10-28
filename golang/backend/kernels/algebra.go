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

type algebraicFactory[T dtype.AlgebraType] struct {
	arrayFactory[T]
}

var _ Factory = (*algebraicFactory[ir.Int])(nil)

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
		default:
			return nil, nil, errors.Errorf("operator %s not supported for atomic-atomic", op.String())
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
			return nil, nil, errors.Errorf("operator %s not supported for atomic-array", op.String())
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
			return nil, nil, errors.Errorf("operator %s not supported for array-atomic", op.String())
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
		return nil, nil, errors.Errorf("operator %s not supported for array-array", op.String())
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

func (algebraicFactory[T]) Cast(kind dtype.DataType, dims []int) (Unary, *shape.Shape, Factory, error) {
	shap := &shape.Shape{
		DType:       dtype.DataType(kind),
		AxisLengths: dims,
	}
	switch kind {
	case dtype.Float32:
		return castArray[T, float32], shap, algebraicFactory[float32]{}, nil
	case dtype.Float64:
		return castArray[T, float64], shap, algebraicFactory[float64]{}, nil
	case dtype.Int32:
		return castArray[T, int32], shap, algebraicFactory[int32]{}, nil
	case dtype.Int64:
		return castArray[T, int64], shap, algebraicFactory[int64]{}, nil
	default:
		return nil, nil, nil, errors.Errorf("cast to %v not supported", kind)
	}
}

// ToAlgebraicAtom converts a value into an atom owned by a backend.
func ToAlgebraicAtom[T dtype.AlgebraType](val T) *ArrayT[T] {
	return &ArrayT[T]{
		factory: algebraicFactory[T]{},
		shape:   &shape.Shape{DType: dtype.Generic[T]()},
		values:  []T{val},
	}
}

// ToAlgebraicArray converts values and a shape into a native multi-dimensional array owned by a backend.
func ToAlgebraicArray[T dtype.AlgebraType](values []T, dims []int) *ArrayT[T] {
	arr := &ArrayT[T]{
		factory: algebraicFactory[T]{},
		shape: &shape.Shape{
			DType:       dtype.Generic[T](),
			AxisLengths: dims,
		},
		values: values,
	}
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
	case dtype.Float32:
		return ToAlgebraicArray(make([]float32, sh.Size()), sh.AxisLengths), nil
	case dtype.Float64:
		return ToAlgebraicArray(make([]float64, sh.Size()), sh.AxisLengths), nil
	case dtype.Int32:
		return ToAlgebraicArray(make([]int32, sh.Size()), sh.AxisLengths), nil
	case dtype.Int64:
		return ToAlgebraicArray(make([]int64, sh.Size()), sh.AxisLengths), nil
	case dtype.Uint32:
		return ToAlgebraicArray(make([]uint32, sh.Size()), sh.AxisLengths), nil
	case dtype.Uint64:
		return ToAlgebraicArray(make([]uint64, sh.Size()), sh.AxisLengths), nil
	default:
		return nil, errors.Errorf("cannot an array of data type %s: not supported", sh.DType)
	}

}
