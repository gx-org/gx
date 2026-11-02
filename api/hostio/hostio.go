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
	"github.com/pkg/errors"
	"github.com/gomlx/compute/dtypes/bfloat16"
	"github.com/gomlx/compute/dtypes/gotype"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/golang/backend/kernels"
)

type (
	// Element used by a value.
	Element interface {
		ir.Element
		ir.StringSourcer
	}

	// Value is a GX value.
	Value interface {
		Element

		// ToHost transfers the value to host given an allocator.
		ToHost(kernels.Allocator) (Value, error)

		// Finalize releases the resources allocated for the value.
		Finalize() error
	}

	// Valuer is an instance able to produce a GX value.
	Valuer interface {
		GXValue() Value
	}

	// Under is implemented by values with an underlying type.
	Under interface {
		Underlying() Value
	}
)

// Underlying returns the underlying value of a value.
func Underlying(v Value) Value {
	under, hasUnder := v.(Under)
	if !hasUnder {
		return v
	}
	return Underlying(under.Underlying())
}

func toHostArray(typ ir.Type, h kernels.Array) (*HostArray, error) {
	return NewHostArray(typ, kernels.NewBuffer(h))
}

// AtomFloatValue returns an array GX value given a Go value.
func AtomFloatValue[T gotype.Float](typ ir.Type, val T) (*HostArray, error) {
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
func AtomIntegerValue[T gotype.Integer](typ ir.Type, val T) (*HostArray, error) {
	return toHostArray(typ, kernels.ToIntegerAtom[T](val))
}

// ToHost transfers all values recursively to the host.
func ToHost(alloc kernels.Allocator, vals []Value) ([]Value, error) {
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

// Finalize releases the resources allocated for all values.
func Finalize(vals ...Value) error {
	for _, val := range vals {
		if err := val.Finalize(); err != nil {
			return err
		}
	}
	return nil
}

// FromAtom converts a Go value into a host array.
func FromAtom(x any, typ ir.Type) (*HostArray, error) {
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

// ToElements converts a slice of values into a slice of elements.
func ToElements(vals []Value) []ir.Element {
	els := make([]ir.Element, len(vals))
	for i, arg := range vals {
		els[i] = arg
	}
	return els
}
