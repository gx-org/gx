// Copyright 2026 Google LLC
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

// Package hostio provides structures and interfaces for the inputs and outputs of the host and GX.
package hostio

import (
	"math/big"

	"github.com/pkg/errors"
	"github.com/gomlx/gopjrt/dtypes/bfloat16"
	"github.com/gx-org/backend/dtypes"
	"github.com/gx-org/backend/platform"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/build/ir/irkind"
	"github.com/gx-org/gx/golang/backend/kernels"
)

// FuncInputs are GX values passed to the function call.
type FuncInputs struct {
	// Receiver on which the function call was done.
	// Can be nil.
	Receiver ir.Element

	// Args returns list of arguments passed to the interpreter at call time.
	Args []ir.Element
}

type (
	// Value is a GX value.
	Value interface {
		ir.Element
		ir.StringSourcer

		// ToHost transfers the value to host given an allocator.
		ToHost(platform.Allocator) (Value, error)
	}

	// Valuer is an instance able to produce a GX value.
	Valuer interface {
		GXValue() Value
	}

	// Factory to create value elements.
	Factory interface {
		NewNamedType(val Value, typ ir.TypeMethods) Value
		NewStruct(typ ir.Type, vals []Value) (Value, error)
	}
)

func toHostArray(typ ir.Type, h kernels.Array) (*HostArray, error) {
	return NewHostArray(typ, kernels.NewBuffer(h))
}

// AtomFloatValue returns an array GX value given a Go value.
func AtomFloatValue[T dtypes.GoFloat](typ ir.Type, val T) (*HostArray, error) {
	return toHostArray(typ, kernels.ToFloatAtom[T](val))
}

// AtomBfloat16Value returns an array GX value given a Go value.
func AtomBfloat16Value(typ ir.Type, val bfloat16.BFloat16) (*HostArray, error) {
	return toHostArray(typ, kernels.ToBfloat16Atom(val))
}

// AtomBoolValue returns an array GX value given a boolean value.
func AtomBoolValue(typ ir.Type, val bool) (*HostArray, error) {
	return toHostArray(typ, kernels.ToBoolAtom(val))
}

// AtomIntegerValue returns an array GX value given a Go value.
func AtomIntegerValue[T dtypes.IntegerType](typ ir.Type, val T) (*HostArray, error) {
	return toHostArray(typ, kernels.ToIntegerAtom[T](val))
}

// ArrayBfloat16Value returns an array GX value given a Go value.
func ArrayBfloat16Value(typ ir.Type, vals []bfloat16.BFloat16, dims []int) (*HostArray, error) {
	return toHostArray(typ, kernels.ToBfloat16Array(vals, dims))
}

// ArrayFloatValue returns an array GX value given a Go value.
func ArrayFloatValue[T dtypes.GoFloat](typ ir.Type, vals []T, dims []int) (*HostArray, error) {
	return toHostArray(typ, kernels.ToFloatArray[T](vals, dims))
}

// ArrayBoolValue returns an array GX value given a boolean value.
func ArrayBoolValue(typ ir.Type, vals []bool, dims []int) (*HostArray, error) {
	return toHostArray(typ, kernels.ToBoolArray(vals, dims))
}

// ArrayIntegerValue returns an array GX value given a Go value.
func ArrayIntegerValue[T dtypes.IntegerType](typ ir.Type, vals []T, dims []int) (*HostArray, error) {
	return toHostArray(typ, kernels.ToIntegerArray[T](vals, dims))
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

func bigIntToInt[T dtypes.IntegerType](x *big.Int) T {
	xI64 := x.Int64()
	return T(xI64)
}

func bigIntToFloat[T float32 | float64](x *big.Int) T {
	xF64, _ := x.Float64()
	return T(xF64)
}

func bigIntToUint[T dtypes.Unsigned](x *big.Int) T {
	xI64 := x.Uint64()
	return T(xI64)
}

// FromAtom converts a Go value into a host array.
func FromAtom[T dtypes.Supported](x T, typ ir.Type) (*HostArray, error) {
	switch xT := any(x).(type) {
	case float32:
		return AtomFloatValue[float32](typ, xT)
	case float64:
		return AtomFloatValue[float64](typ, xT)
	case int:
		return AtomIntegerValue[int](typ, xT)
	case int32:
		return AtomIntegerValue[int32](typ, xT)
	case int64:
		return AtomIntegerValue[int64](typ, xT)
	case uint32:
		return AtomIntegerValue[uint32](typ, xT)
	case uint64:
		return AtomIntegerValue[uint64](typ, xT)
	default:
		return nil, errors.Errorf("cannot convert Go value %T(%v) (GX type: %s) to a host array", xT, xT, typ.ReferString(nil))
	}
}

// AtomNumberInt evaluates a big integer number into a GX array value.
func AtomNumberInt(x *big.Int, typ ir.Type) (*HostArray, error) {
	switch typ.Kind() {
	case irkind.Bfloat16:
		xF64, _ := x.Float64()
		return AtomBfloat16Value(typ, bfloat16.FromFloat64(xF64))
	case irkind.Float32:
		return AtomFloatValue[float32](typ, bigIntToFloat[float32](x))
	case irkind.Float64:
		return AtomFloatValue[float64](typ, bigIntToFloat[float64](x))
	case irkind.Int:
		return AtomIntegerValue[int](typ, bigIntToInt[int](x))
	case irkind.Int32:
		return AtomIntegerValue[int32](typ, bigIntToInt[int32](x))
	case irkind.Int64:
		return AtomIntegerValue[int64](typ, bigIntToInt[int64](x))
	case irkind.Uint32:
		return AtomIntegerValue[uint32](typ, bigIntToUint[uint32](x))
	case irkind.Uint64:
		return AtomIntegerValue[uint64](typ, bigIntToUint[uint64](x))
	}
	return nil, errors.Errorf("cannot convert value %s of type %s (kind: %s) to an atomic integer value", x.String(), typ.ReferString(nil), typ.Kind().String())
}

func bigFloatCast[T dtypes.AlgebraType](x *big.Float) T {
	xF64, _ := x.Float64()
	return T(xF64)
}

// AtomNumberFloat  evaluates a big integer number into a GX array value.
func AtomNumberFloat(x *big.Float, typ ir.Type) (*HostArray, error) {
	switch typ.Kind() {
	case irkind.Bfloat16:
		xF64, _ := x.Float64()
		return AtomBfloat16Value(typ, bfloat16.FromFloat64(xF64))
	case irkind.Float32:
		return AtomFloatValue[float32](typ, bigFloatCast[float32](x))
	case irkind.Float64:
		return AtomFloatValue[float64](typ, bigFloatCast[float64](x))
	case irkind.Int:
		return AtomIntegerValue[int](typ, bigFloatCast[int](x))
	case irkind.Int32:
		return AtomIntegerValue[int32](typ, bigFloatCast[int32](x))
	case irkind.Int64:
		return AtomIntegerValue[int64](typ, bigFloatCast[int64](x))
	case irkind.Uint32:
		return AtomIntegerValue[uint32](typ, bigFloatCast[uint32](x))
	case irkind.Uint64:
		return AtomIntegerValue[uint64](typ, bigFloatCast[uint64](x))
	}
	return nil, errors.Errorf("cannot convert %T(%s) to %s: not implemented", x, x, typ.ReferString(nil))
}

// ToElements converts a slice of values into a slice of elements.
func ToElements(vals []Value) []ir.Element {
	els := make([]ir.Element, len(vals))
	for i, arg := range vals {
		els[i] = arg
	}
	return els
}

// ToKernel returns the kernel of an array.
func ToKernel(array *HostArray) (kernels.Array, error) {
	// Convert the GX value into a Go array with a kernel factory.
	data := array.Buffer().Acquire()
	defer array.Buffer().Release()
	data = append([]byte{}, data...)
	kArray, err := kernels.NewArrayFromRaw(data, array.Shape())
	if err != nil {
		return nil, err
	}
	return kArray, nil
}
