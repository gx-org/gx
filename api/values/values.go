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
	"github.com/gx-org/gx/api/hostio"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/build/ir/irkind"
)

type factory struct{}

var f factory

// Factory to create host elements.
func Factory() hostio.Factory {
	return f
}

func (factory) NewNamedType(val hostio.Value, typ ir.TypeMethods) hostio.Value {
	return NewNamedType(val, typ)
}

func (factory) NewStruct(typ ir.Type, vals []hostio.Value) (hostio.Value, error) {
	return NewStruct(typ, vals)
}

func sliceZeroValue(typ ir.Type) (*Slice, error) {
	return NewSlice(typ, nil)
}

func arrayZeroValue(typ ir.Type) (hostio.Array, error) {
	// TODO(degris): not really implemented: should be enough for a workaround today.
	return hostio.NewDeviceArray(typ, nil)
}

// Zero returns a zero value given a GX type.
func Zero(typ ir.Type) (hostio.Value, error) {
	kind := typ.Kind()
	switch kind {
	case irkind.Bool:
		return hostio.AtomBoolValue(typ, false)
	case irkind.Bfloat16:
		return hostio.AtomBfloat16Value(typ, 0)
	case irkind.Float32:
		return hostio.AtomFloatValue[float32](typ, 0)
	case irkind.Float64:
		return hostio.AtomFloatValue[float64](typ, 0)
	case irkind.Int:
		return hostio.AtomIntegerValue[ir.Int](typ, 0)
	case irkind.Int32:
		return hostio.AtomIntegerValue[int32](typ, 0)
	case irkind.Int64:
		return hostio.AtomIntegerValue[int64](typ, 0)
	case irkind.Uint32:
		return hostio.AtomIntegerValue[uint32](typ, 0)
	case irkind.Uint64:
		return hostio.AtomIntegerValue[uint64](typ, 0)
	case irkind.Array:
		return arrayZeroValue(typ)
	case irkind.Slice:
		return sliceZeroValue(typ)
	default:
		return nil, errors.Errorf("cannot create a zero value of %s", kind.String())
	}
}

// Underlying returns the underlying element.
func Underlying(val hostio.Value) hostio.Value {
	named, ok := val.(*NamedType)
	if !ok {
		return val
	}
	return Underlying(named.Underlying())
}
