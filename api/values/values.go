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

// Package values implements all values that GX can represent.
package values

import (
	"math/big"

	"github.com/pkg/errors"
	"github.com/gx-org/backend/dtype"
	"github.com/gx-org/backend/platform"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/golang/backend/kernels"
)

type (
	// Value is a GX value.
	Value interface {
		value() // Make sure all GX types are implemented in this package.

		// Type returns the type of the value.
		Type() ir.Type

		// ToHost transfers the value to host given an allocator.
		ToHost(platform.Allocator) (Value, error)

		// String representation of the value.
		// The returned string is a string reported to the user.
		String() string
	}

	// Valuer is an instance able to produce a GX value.
	Valuer interface {
		GXValue() Value
	}

	// FuncInputs are GX values passed to the function call.
	FuncInputs struct {
		// Receiver on which the function call was done.
		// Can be nil.
		Receiver Value

		// Args returns list of arguments passed to the interpreter at call time.
		Args []Value
	}
)

func toHostArray(typ ir.Type, h kernels.Array) (*HostArray, error) {
	return NewHostArray(typ, kernels.NewBuffer(h))
}

// AtomFloatValue returns an array GX value given a Go value.
func AtomFloatValue[T dtype.Float](typ ir.Type, val T) (*HostArray, error) {
	return toHostArray(typ, kernels.ToFloatAtom[T](val))
}

// AtomBfloat16Value returns an array GX value given a Go value.
func AtomBfloat16Value(typ ir.Type, val dtype.Bfloat16T) (*HostArray, error) {
	return toHostArray(typ, kernels.ToBfloat16Atom(val))
}

// AtomBoolValue returns an array GX value given a boolean value.
func AtomBoolValue(typ ir.Type, val bool) (*HostArray, error) {
	return toHostArray(typ, kernels.ToBoolAtom(val))
}

// AtomIntegerValue returns an array GX value given a Go value.
func AtomIntegerValue[T dtype.IntegerType](typ ir.Type, val T) (*HostArray, error) {
	return toHostArray(typ, kernels.ToIntegerAtom[T](val))
}

// ArrayBfloat16Value returns an array GX value given a Go value.
func ArrayBfloat16Value(typ ir.Type, vals []dtype.Bfloat16T, dims []int) (*HostArray, error) {
	return toHostArray(typ, kernels.ToBfloat16Array(vals, dims))
}

// ArrayFloatValue returns an array GX value given a Go value.
func ArrayFloatValue[T dtype.Float](typ ir.Type, vals []T, dims []int) (*HostArray, error) {
	return toHostArray(typ, kernels.ToFloatArray[T](vals, dims))
}

// ArrayBoolValue returns an array GX value given a boolean value.
func ArrayBoolValue(typ ir.Type, vals []bool, dims []int) (*HostArray, error) {
	return toHostArray(typ, kernels.ToBoolArray(vals, dims))
}

// ArrayIntegerValue returns an array GX value given a Go value.
func ArrayIntegerValue[T dtype.IntegerType](typ ir.Type, vals []T, dims []int) (*HostArray, error) {
	return toHostArray(typ, kernels.ToIntegerArray[T](vals, dims))
}

func arrayZeroValue(typ ir.Type) (Array, error) {
	// TODO(degris): not really implemented: should be enough for a workaround today.
	return NewDeviceArray(typ, nil)
}

func sliceZeroValue(typ ir.Type) (*Slice, error) {
	return NewSlice(typ, nil)
}

// Zero returns a zero value given a GX type.
func Zero(typ ir.Type) (Value, error) {
	kind := typ.Kind()
	switch kind {
	case ir.BoolKind:
		return AtomBoolValue(typ, false)
	case ir.Bfloat16Kind:
		return AtomBfloat16Value(typ, 0)
	case ir.Float32Kind:
		return AtomFloatValue[float32](typ, 0)
	case ir.Float64Kind:
		return AtomFloatValue[float64](typ, 0)
	case ir.Int32Kind:
		return AtomIntegerValue[int32](typ, 0)
	case ir.Int64Kind:
		return AtomIntegerValue[int64](typ, 0)
	case ir.Uint32Kind:
		return AtomIntegerValue[uint32](typ, 0)
	case ir.Uint64Kind:
		return AtomIntegerValue[uint64](typ, 0)
	case ir.IntLenKind:
		return AtomIntegerValue[ir.Int](typ, 0)
	case ir.IntIdxKind:
		return AtomIntegerValue[ir.Int](typ, 0)
	case ir.ArrayKind:
		return arrayZeroValue(typ)
	case ir.SliceKind:
		return sliceZeroValue(typ)
	default:
		return nil, errors.Errorf("cannot create a zero value of %s", kind.String())
	}
}

// ToHost transfers all values recursively to the host.
func ToHost(alloc platform.Allocator, vals []Value) ([]Value, error) {
	out := make([]Value, len(vals))
	for i, val := range vals {
		var err error
		out[i], err = val.ToHost(alloc)
		if err != nil {
			return nil, err
		}
	}
	return out, nil
}

func bigIntToInt[T dtype.IntegerType](x *big.Int) T {
	xI64 := x.Int64()
	return T(xI64)
}

func bigIntToFloat[T float32 | float64](x *big.Int) T {
	xF64, _ := x.Float64()
	return T(xF64)
}

func bigIntToUint[T dtype.Unsigned](x *big.Int) T {
	xI64 := x.Uint64()
	return T(xI64)
}

// AtomNumberInt evaluates a big integer number into a GX array value.
func AtomNumberInt(x *big.Int, typ ir.Type) (*HostArray, error) {
	switch typ.Kind() {
	case ir.Bfloat16Kind:
		xF64, _ := x.Float64()
		return AtomBfloat16Value(typ, dtype.BFloat16FromFloat64(xF64))
	case ir.Float32Kind:
		return AtomFloatValue[float32](typ, bigIntToFloat[float32](x))
	case ir.Float64Kind:
		return AtomFloatValue[float64](typ, bigIntToFloat[float64](x))
	case ir.Int32Kind:
		return AtomIntegerValue[int32](typ, bigIntToInt[int32](x))
	case ir.Int64Kind:
		return AtomIntegerValue[int64](typ, bigIntToInt[int64](x))
	case ir.Uint32Kind:
		return AtomIntegerValue[uint32](typ, bigIntToUint[uint32](x))
	case ir.Uint64Kind:
		return AtomIntegerValue[uint64](typ, bigIntToUint[uint64](x))
	case ir.IntLenKind:
		return AtomIntegerValue[ir.Int](typ, bigIntToInt[ir.Int](x))
	case ir.IntIdxKind:
		return AtomIntegerValue[ir.Int](typ, bigIntToInt[ir.Int](x))
	}
	return nil, errors.Errorf("cannot convert %T(%s) to %s: not implemented", x, x, typ.String())
}

func bigFloatCast[T dtype.AlgebraType](x *big.Float) T {
	xF64, _ := x.Float64()
	return T(xF64)
}

// AtomNumberFloat  evaluates a big integer number into a GX array value.
func AtomNumberFloat(x *big.Float, typ ir.Type) (Array, error) {
	switch typ.Kind() {
	case ir.Bfloat16Kind:
		xF64, _ := x.Float64()
		return AtomBfloat16Value(typ, dtype.BFloat16FromFloat64(xF64))
	case ir.Float32Kind:
		return AtomFloatValue[float32](typ, bigFloatCast[float32](x))
	case ir.Float64Kind:
		return AtomFloatValue[float64](typ, bigFloatCast[float64](x))
	case ir.Int32Kind:
		return AtomIntegerValue[int32](typ, bigFloatCast[int32](x))
	case ir.Int64Kind:
		return AtomIntegerValue[int64](typ, bigFloatCast[int64](x))
	case ir.Uint32Kind:
		return AtomIntegerValue[uint32](typ, bigFloatCast[uint32](x))
	case ir.Uint64Kind:
		return AtomIntegerValue[uint64](typ, bigFloatCast[uint64](x))
	case ir.IntLenKind:
		return AtomIntegerValue[ir.Int](typ, bigFloatCast[ir.Int](x))
	case ir.IntIdxKind:
		return AtomIntegerValue[ir.Int](typ, bigFloatCast[ir.Int](x))
	}
	return nil, errors.Errorf("cannot convert %T(%s) to %s: not implemented", x, x, typ.String())
}

// Underlying returns the underlying element.
func Underlying(val Value) Value {
	named, ok := val.(*NamedType)
	if !ok {
		return val
	}
	return Underlying(named.Underlying())
}
