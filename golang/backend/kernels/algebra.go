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

func castArray[T, U dtype.AlgebraType](dims []int) Unary {
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
	case dtype.Float32:
		return castArray[T, float32](dims), shap, algebraicFactory[float32]{}, nil
	case dtype.Float64:
		return castArray[T, float64](dims), shap, algebraicFactory[float64]{}, nil
	case dtype.Int32:
		return castArray[T, int32](dims), shap, integerFactory[int32]{}, nil
	case dtype.Int64:
		return castArray[T, int64](dims), shap, integerFactory[int64]{}, nil
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
		}
		return f.algebraicFactory.BinaryOp(op, x, y)
	}
	if xAtomic {
		out := &shape.Shape{DType: x.DType, AxisLengths: y.AxisLengths}
		switch op {
		case token.REM:
			return remAtomicToArray[T], out, nil
		}
		return f.algebraicFactory.BinaryOp(op, x, y)
	}
	if yAtomic {
		out := &shape.Shape{DType: x.DType, AxisLengths: x.AxisLengths}
		switch op {
		case token.REM:
			return remArrayToAtomic[T], out, nil
		}
		return f.algebraicFactory.BinaryOp(op, x, y)
	}
	out := &shape.Shape{DType: x.DType, AxisLengths: y.AxisLengths}
	switch op {
	case token.REM:
		return remArrayToArray[T], out, nil
	}
	return f.algebraicFactory.BinaryOp(op, x, y)
}

// ToFloatAtom converts a value into an atom owned by a backend.
func ToFloatAtom[T dtype.Float](val T) *ArrayT[T] {
	return &ArrayT[T]{
		factory: algebraicFactory[T]{},
		shape:   shape.Shape{DType: dtype.Generic[T]()},
		values:  []T{val},
	}
}

// ToIntegerAtom converts a value into an atom owned by a backend.
func ToIntegerAtom[T dtype.IntegerType](val T) *ArrayT[T] {
	return &ArrayT[T]{
		factory: integerFactory[T]{},
		shape:   shape.Shape{DType: dtype.Generic[T]()},
		values:  []T{val},
	}
}

// ToFloatArray converts values and a shape into a native multi-dimensional array owned by a backend.
func ToFloatArray[T dtype.Float](values []T, dims []int) *ArrayT[T] {
	arr := &ArrayT[T]{
		factory: algebraicFactory[T]{},
		shape: shape.Shape{
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

// ToIntegerArray converts values and a shape into a native multi-dimensional array owned by a backend.
func ToIntegerArray[T dtype.IntegerType](values []T, dims []int) *ArrayT[T] {
	arr := &ArrayT[T]{
		factory: integerFactory[T]{},
		shape: shape.Shape{
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
