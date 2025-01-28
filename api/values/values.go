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
)

func toHostArray(typ ir.Type, h kernels.Array) *HostArray {
	return NewHostArray(typ, kernels.NewBuffer(h))
}

// AtomFloatValue returns an array GX value given a Go value.
func AtomFloatValue[T dtype.Float](typ ir.Type, val T) *HostArray {
	return toHostArray(typ, kernels.ToFloatAtom[T](val))
}

// AtomBoolValue returns an array GX value given a boolean value.
func AtomBoolValue(typ ir.Type, val bool) *HostArray {
	return toHostArray(typ, kernels.ToBoolAtom(val))
}

// AtomIntegerValue returns an array GX value given a Go value.
func AtomIntegerValue[T dtype.IntegerType](typ ir.Type, val T) *HostArray {
	return toHostArray(typ, kernels.ToIntegerAtom[T](val))
}

// ArrayFloatValue returns an array GX value given a Go value.
func ArrayFloatValue[T dtype.Float](typ ir.Type, vals []T, dims []int) *HostArray {
	return toHostArray(typ, kernels.ToFloatArray[T](vals, dims))
}

// ArrayBoolValue returns an array GX value given a boolean value.
func ArrayBoolValue(typ ir.Type, vals []bool, dims []int) *HostArray {
	return toHostArray(typ, kernels.ToBoolArray(vals, dims))
}

// ArrayIntegerValue returns an array GX value given a Go value.
func ArrayIntegerValue[T dtype.IntegerType](typ ir.Type, vals []T, dims []int) *HostArray {
	return toHostArray(typ, kernels.ToIntegerArray[T](vals, dims))
}

func arrayZeroValue(typ ir.Type) (Array, error) {
	// TODO(degris): not really implemented: should be enough for a workaround today.
	// Does not return an error because some tests need this workaround.
	return NewDeviceArray(typ, nil), nil
}

func sliceZeroValue(typ ir.Type) (*Slice, error) {
	return NewSlice(typ, nil)
}

// Zero returns a zero value given a GX type.
func Zero(typ ir.Type) (Value, error) {
	kind := typ.Kind()
	switch kind {
	case ir.BoolKind:
		return AtomBoolValue(typ, false), nil
	case ir.Float32Kind:
		return AtomFloatValue[float32](typ, 0), nil
	case ir.Float64Kind:
		return AtomFloatValue[float64](typ, 0), nil
	case ir.Int32Kind:
		return AtomIntegerValue[int32](typ, 0), nil
	case ir.Int64Kind:
		return AtomIntegerValue[int64](typ, 0), nil
	case ir.Uint32Kind:
		return AtomIntegerValue[uint32](typ, 0), nil
	case ir.Uint64Kind:
		return AtomIntegerValue[uint64](typ, 0), nil
	case ir.IntLenKind:
		return AtomIntegerValue[ir.Int](typ, 0), nil
	case ir.IntIdxKind:
		return AtomIntegerValue[ir.Int](typ, 0), nil
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
