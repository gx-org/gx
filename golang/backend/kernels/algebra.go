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
	"github.com/gomlx/compute/dtypes/bfloat16"
	"github.com/gomlx/compute/dtypes"
	"github.com/gomlx/compute/dtypes/gotype"
	"github.com/gomlx/compute/shapes"
	"github.com/gx-org/gx/build/ir"
)

type algebraicFactory[T gotype.NumericNotComplex] struct {
	arrayFactory[T]
	mathFactory[T]
}

var _ Factory = (*algebraicFactory[ir.Int])(nil)

func (f algebraicFactory[T]) Math() MathFactory {
	return f.mathFactory
}

// BinaryOp creates a new kernel for a binary operator.
func (algebraicFactory[T]) BinaryOp(op token.Token, x, y shapes.Shape) (Binary, shapes.Shape, error) {
	xAtomic := isAtomic(x)
	yAtomic := isAtomic(y)
	if xAtomic && yAtomic {
		out := shapes.Make(x.DType)
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
			return eqlAtomicToAtomic[T], shapes.Make(dtypes.Bool), nil
		case token.NEQ:
			return neqAtomicToAtomic[T], shapes.Make(dtypes.Bool), nil
		case token.LSS:
			return lssAtomicToAtomic[T], shapes.Make(dtypes.Bool), nil
		case token.GTR:
			return gtrAtomicToAtomic[T], shapes.Make(dtypes.Bool), nil
		case token.LEQ:
			return leqAtomicToAtomic[T], shapes.Make(dtypes.Bool), nil
		case token.GEQ:
			return geqAtomicToAtomic[T], shapes.Make(dtypes.Bool), nil
		default:
			return nil, shapes.Shape{}, errors.Errorf("operator %s not supported for %s atomic-atomic", op.String(), x.DType.String())
		}
	}
	if xAtomic {
		out := shapes.Make(x.DType, y.Dimensions...)
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
			return equalAtomicToArray[T], shapes.Make(dtypes.Bool, y.Dimensions...), nil
		default:
			return nil, shapes.Shape{}, errors.Errorf("operator %s not supported for %s atomic-array", op.String(), x.DType.String())
		}
	}
	if yAtomic {
		out := shapes.Make(x.DType, x.Dimensions...)
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
			return equalArrayToAtomic[T], shapes.Make(dtypes.Bool, x.Dimensions...), nil
		default:
			return nil, shapes.Shape{}, errors.Errorf("operator %s not supported for %s array-atomic", op.String(), x.DType.String())
		}
	}
	out := shapes.Make(x.DType, y.Dimensions...)
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
		return equalArrayToArray[T], shapes.Make(dtypes.Bool, y.Dimensions...), nil
	default:
		return nil, shapes.Shape{}, errors.Errorf("operator %s not supported for %s array-array", op.String(), x.DType.String())
	}
}

// UnaryOp creates a new kernel for a unary operator.
func (algebraicFactory[T]) UnaryOp(op token.Token, x shapes.Shape) (Unary, shapes.Shape, error) {
	switch op {
	case token.SUB:
		return subArray[T], x, nil
	default:
		return nil, shapes.Shape{}, errors.Errorf("operator %q supported", op.String())
	}
}

func castArray[T gotype.NumericNotComplex, U gotype.NumericNotComplex](dims []int) Unary {
	return func(x Array) (Array, error) {
		return castArrayWithShape[T, U](x, dims)
	}
}

func (algebraicFactory[T]) Cast(kind dtypes.DType, dims []int) (Unary, shapes.Shape, Factory, error) {
	shap := shapes.Make(kind, dims...)
	switch kind {
	case dtypes.BFloat16:
		return castToBfloat16Array[T], shap, bfloat16Factory{}, nil
	case dtypes.Float32:
		return castArray[T, float32](dims), shap, algebraicFactory[float32]{}, nil
	case dtypes.Float64:
		return castArray[T, float64](dims), shap, algebraicFactory[float64]{}, nil
	case dtypes.Int32:
		return castArray[T, int32](dims), shap, integerFactory[int32]{}, nil
	case dtypes.Int64:
		return castArray[T, int64](dims), shap, integerFactory[int64]{}, nil
	case dtypes.Uint32:
		return castArray[T, uint32](dims), shap, integerFactory[uint32]{}, nil
	case dtypes.Uint64:
		return castArray[T, uint64](dims), shap, integerFactory[uint64]{}, nil
	default:
		return nil, shapes.Shape{}, nil, errors.Errorf("cast to %v not supported", kind)
	}
}

type integerFactory[T gotype.Integer] struct {
	algebraicFactory[T]
}

// BinaryOp creates a new kernel for a binary operator.
func (f integerFactory[T]) BinaryOp(op token.Token, x, y shapes.Shape) (Binary, shapes.Shape, error) {
	xAtomic := isAtomic(x)
	yAtomic := isAtomic(y)
	if xAtomic && yAtomic {
		out := shapes.Make(x.DType)
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
		out := shapes.Make(x.DType, y.Dimensions...)
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
		out := shapes.Make(x.DType, x.Dimensions...)
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
	out := shapes.Make(x.DType, y.Dimensions...)
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
	arrayFactory[bfloat16.BFloat16]
}

var _ Factory = (*bfloat16Factory)(nil)

func (bfloat16Factory) Math() MathFactory {
	return nil
}

func (bfloat16Factory) BinaryOp(op token.Token, x, y shapes.Shape) (Binary, shapes.Shape, error) {
	return nil, shapes.Shape{}, errors.Errorf("operator %q not supported", op.String())
}

// UnaryOp creates a new kernel for a unary operator.
func (bfloat16Factory) UnaryOp(op token.Token, x shapes.Shape) (Unary, shapes.Shape, error) {
	return nil, shapes.Shape{}, errors.Errorf("operator %q not supported", op.String())
}

// Cast an array to another
func (f bfloat16Factory) Cast(kind dtypes.DType, dims []int) (Unary, shapes.Shape, Factory, error) {
	shap := shapes.Make(kind, dims...)
	switch kind {
	case dtypes.BFloat16:
		return func(a Array) (Array, error) { return a, nil }, shap, f, nil
	case dtypes.Float32:
		return castFromBfloat16Array[float32], shap, algebraicFactory[float32]{}, nil
	case dtypes.Float64:
		return castFromBfloat16Array[float64], shap, algebraicFactory[float64]{}, nil
	case dtypes.Int32:
		return castFromBfloat16Array[int32], shap, integerFactory[int32]{}, nil
	case dtypes.Int64:
		return castFromBfloat16Array[int64], shap, integerFactory[int64]{}, nil
	default:
		return nil, shapes.Shape{}, nil, errors.Errorf("cast to %v not supported", kind)
	}
}

func castToBfloat16Array[T gotype.NumericNotComplex](xVal Array) (Array, error) {
	x := toArray[T](xVal)
	z := make([]bfloat16.BFloat16, x.shape.Size())
	for i, xi := range x.values {
		z[i] = bfloat16.FromFloat32((float32)(xi))
	}
	return ToBfloat16Array(z, x.shape.Dimensions), nil
}

func castFromBfloat16Array[U gotype.NumericNotComplex](xVal Array) (Array, error) {
	x := toArray[bfloat16.BFloat16](xVal)
	z := make([]U, x.shape.Size())
	for i, xi := range x.values {
		z[i] = U(xi.Float32())
	}
	return ToAlgebraicArray[U](z, x.shape.Dimensions), nil
}

// ToBfloat16Atom converts a value into an atom owned by a backend.
func ToBfloat16Atom(val bfloat16.BFloat16) Array {
	return ToBfloat16Array([]bfloat16.BFloat16{val}, nil)
}

// ToFloatAtom converts a value into an atom owned by a backend.
func ToFloatAtom[T gotype.Float](val T) Array {
	return ToFloatArray([]T{val}, nil)
}

// ToIntegerAtom converts a value into an atom owned by a backend.
func ToIntegerAtom[T gotype.Integer](val T) Array {
	return ToIntegerArray([]T{val}, nil)
}

// ToBfloat16Array converts values and a shape into a native multi-dimensional array owned by a backend.
func ToBfloat16Array(values []bfloat16.BFloat16, dims []int) Array {
	arr := &bfloat16Array{&arrayT[bfloat16.BFloat16]{
		factory: bfloat16Factory{},
		shape:   shapes.Make(dtypes.BFloat16, dims...),
		values:  values,
	}}
	if len(values) != arr.shape.Size() {
		panic(fmt.Sprintf("mismatch between the number of values (=%d) and the number of elements (=%d) in shape %s", len(values), arr.shape.Size(), arr.shape.String()))
	}
	return arr
}

// ToAlgebraicArray converts values and a shape into a native multi-dimensional array owned by a backend.
func ToAlgebraicArray[T gotype.NumericNotComplex](values []T, dims []int) Array {
	arr := &algebraArray[T]{&arrayT[T]{
		factory: algebraicFactory[T]{},
		shape:   shapes.Make(dtypes.FromGenericsType[T](), dims...),
		values:  values,
	}}
	if len(values) != arr.shape.Size() {
		panic(fmt.Sprintf("mismatch between the number of values (=%d) and the number of elements (=%d) in shape %s", len(values), arr.shape.Size(), arr.shape.String()))
	}
	return arr
}

// ToFloatArray converts values and a shape into a native multi-dimensional array owned by a backend.
func ToFloatArray[T gotype.Float](values []T, dims []int) Array {
	return ToAlgebraicArray(values, dims)
}

// ToIntegerArray converts values and a shape into a native multi-dimensional array owned by a backend.
func ToIntegerArray[T gotype.Integer](values []T, dims []int) Array {
	arr := &algebraArray[T]{&arrayT[T]{
		factory: integerFactory[T]{},
		shape:   shapes.Make(dtypes.FromGenericsType[T](), dims...),
		values:  values,
	}}
	if len(values) != arr.shape.Size() {
		panic(fmt.Sprintf("mismatch between the number of values (=%d) and the number of elements (=%d) in shape %s", len(values), arr.shape.Size(), arr.shape.String()))
	}
	return arr
}

// Zero returns an array of zeros given a shape.
func Zero(sh shapes.Shape) (Array, error) {
	switch sh.DType {
	case dtypes.Bool:
		return ToBoolArray(make([]bool, sh.Size()), sh.Dimensions), nil
	case dtypes.BFloat16:
		return ToBfloat16Array(make([]bfloat16.BFloat16, sh.Size()), sh.Dimensions), nil
	case dtypes.Float32:
		return ToFloatArray(make([]float32, sh.Size()), sh.Dimensions), nil
	case dtypes.Float64:
		return ToFloatArray(make([]float64, sh.Size()), sh.Dimensions), nil
	case dtypes.Int32:
		return ToIntegerArray(make([]int32, sh.Size()), sh.Dimensions), nil
	case dtypes.Int64:
		return ToIntegerArray(make([]int64, sh.Size()), sh.Dimensions), nil
	case dtypes.Uint32:
		return ToIntegerArray(make([]uint32, sh.Size()), sh.Dimensions), nil
	case dtypes.Uint64:
		return ToIntegerArray(make([]uint64, sh.Size()), sh.Dimensions), nil
	default:
		return nil, errors.Errorf("cannot create an array of data type %s: not supported", sh.DType)
	}
}
