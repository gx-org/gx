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

// Package native fills a GX from Go maps.
package native

import (
	"reflect"

	"github.com/pkg/errors"
	"github.com/gx-org/backend/shape"
	"github.com/gx-org/gx/golang/binder/gobindings/types"
	"github.com/gx-org/gx/golang/encoding"
)

type (
	walkerStruct struct{ data map[string]any }
	walkerSlice  struct{ data []any }
	walkerData   struct{ data any }
)

func (w walkerStruct) Field(name string) (encoding.Data, error) {
	data, ok := w.data[name]
	if !ok {
		return nil, errors.Errorf("no field %s", name)
	}
	return walkerData{data: data}, nil
}

func (w walkerStruct) TagName() string {
	return "native"
}

func (w walkerSlice) Len() int {
	return len(w.data)
}

func (w walkerSlice) Index(i int) (encoding.Data, error) {
	return walkerData{data: w.data[i]}, nil
}

func (w walkerData) ToDataSlice() (encoding.DataSlice, error) {
	data, ok := w.data.([]any)
	if !ok {
		return nil, errors.Errorf("cannot cast %T to %s", w, reflect.TypeFor[[]types.Bridge]().String())
	}
	return walkerSlice{data: data}, nil
}

func (w walkerData) ToDataStruct() (encoding.DataStruct, error) {
	data, ok := w.data.(map[string]any)
	if !ok {
		return nil, errors.Errorf("cannot cast %T to %s", w, reflect.TypeFor[map[string]types.Bridge]().String())
	}
	return walkerStruct{data: data}, nil
}

func (w walkerData) ValueFuture() (encoding.ValueFuture, error) {
	return w, nil
}

func (w walkerData) Value() (types.ArrayBridge, error) {
	return goNumericalToBridge(w.data)
}

// Unmarshal populates a GX structure from a Go map.
func Unmarshal(target types.Bridger, data map[string]any) error {
	return encoding.Unmarshal(nil, target, walkerData{data: data})
}

func goNumericalToBridge(val any) (types.ArrayBridge, error) {
	switch valT := val.(type) {
	case float32:
		return types.Float32(valT), nil
	case float64:
		return types.Float64(valT), nil
	case int32:
		return types.Int32(valT), nil
	case int64:
		return types.Int64(valT), nil
	case uint32:
		return types.Uint32(valT), nil
	case uint64:
		return types.Uint64(valT), nil
	case []float32:
		return types.ArrayFloat32(valT), nil
	case []float64:
		return types.ArrayFloat64(valT), nil
	case []int32:
		return types.ArrayInt32(valT), nil
	case []int64:
		return types.ArrayInt64(valT), nil
	case []uint32:
		return types.ArrayUint32(valT), nil
	case []uint64:
		return types.ArrayUint64(valT), nil
	case shape.ArrayI[float32]:
		return types.ArrayFloat32(valT.Flat(), valT.Shape()...), nil
	case shape.ArrayI[float64]:
		return types.ArrayFloat64(valT.Flat(), valT.Shape()...), nil
	case shape.ArrayI[int32]:
		return types.ArrayInt32(valT.Flat(), valT.Shape()...), nil
	case shape.ArrayI[int64]:
		return types.ArrayInt64(valT.Flat(), valT.Shape()...), nil
	case shape.ArrayI[uint32]:
		return types.ArrayUint32(valT.Flat(), valT.Shape()...), nil
	case shape.ArrayI[uint64]:
		return types.ArrayUint64(valT.Flat(), valT.Shape()...), nil
	default:
		return nil, errors.Errorf("cannot convert %T to %s: not supported", val, reflect.TypeFor[types.Bridge]())
	}
}
