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

func atomZeroValue[T dtype.AlgebraType](typ ir.Type) Array {
	handle := kernels.ToAlgebraicAtom[T](0)
	return NewHostArray(typ, kernels.NewBuffer(handle))
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
		handle := kernels.ToBoolAtom(false)
		return NewHostArray(typ, kernels.NewBuffer(handle)), nil
	case ir.Float32Kind:
		return atomZeroValue[float32](typ), nil
	case ir.Float64Kind:
		return atomZeroValue[float64](typ), nil
	case ir.Int32Kind:
		return atomZeroValue[int32](typ), nil
	case ir.Int64Kind:
		return atomZeroValue[int64](typ), nil
	case ir.Uint32Kind:
		return atomZeroValue[uint32](typ), nil
	case ir.Uint64Kind:
		return atomZeroValue[uint64](typ), nil
	case ir.AxisLengthKind:
		return atomZeroValue[ir.Int](typ), nil
	case ir.AxisIndexKind:
		return atomZeroValue[ir.Int](typ), nil
	case ir.TensorKind:
		return arrayZeroValue(typ)
	case ir.SliceKind:
		return sliceZeroValue(typ)
	default:
		return nil, errors.Errorf("cannot create a zero value of %s", kind.String())
	}
}

// ToHost transfers all values recursirvely to the host.
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
