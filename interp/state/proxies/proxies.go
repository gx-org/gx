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

// Package proxies provides proxy for all GX values.
package proxies

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/gx-org/backend/shape"
	"github.com/gx-org/gx/api/values"
	"github.com/gx-org/gx/build/ir"
)

type (
	// Value is a value for which we know the type but not the value yet.
	Value interface {
		proxy()

		// Type returns the GX type of the value.
		Type() ir.Type
	}

	base struct {
		typ ir.Type
	}

	// Array is the proxy for an atomic or an array value.
	Array struct {
		base
		shape *shape.Shape
	}

	// Struct is the proxy for a structure value.
	Struct struct {
		base
		strctType *ir.StructType
		vals      map[string]Value
	}

	// Slice is the proxy for a slice value.
	Slice struct {
		base
		sliceType *ir.SliceType
		vals      []Value
	}
)

func prBase(typ ir.Type) base {
	return base{typ: typ}
}

func (pr *base) Type() ir.Type { return pr.typ }

func (base) proxy() {}

// NewArray returns a new proxy array.
func NewArray(typ ir.Type, shape *shape.Shape) *Array {
	return &Array{
		base:  base{typ: typ},
		shape: shape,
	}
}

// Shape of the value.
func (pr *Array) Shape() *shape.Shape {
	return pr.shape
}

// StructType returns the structure type of the value.
func (pr *Struct) StructType() *ir.StructType {
	return pr.strctType
}

// Field returns the value of a field.
func (pr *Struct) Field(name string) Value {
	return pr.vals[name]
}

// Element return the ith element of the proxy value.
func (pr *Slice) Element(i int) (Value, error) {
	if i < 0 || i >= len(pr.vals) {
		return nil, errors.Errorf("invalid argument: index %d out of bounds [0:%d]", i, len(pr.vals))
	}
	return pr.vals[i], nil
}

// Size returns the number of element in the slice.
func (pr *Slice) Size() int {
	return len(pr.vals)
}

// SliceType returns the type of the slice.
func (pr *Slice) SliceType() *ir.SliceType {
	return pr.sliceType
}

// ToProxy returns the proxy value of a given value.
func ToProxy(val values.Value, typ ir.Type) (Value, error) {
	if val == nil {
		var err error
		val, err = values.Zero(typ)
		if err != nil {
			return nil, err
		}
	}
	switch valT := val.(type) {
	case values.Array:
		return &Array{
			base:  prBase(valT.Type()),
			shape: valT.Shape(),
		}, nil
	case *values.Struct:
		vals, err := structToProxyValues(valT)
		if err != nil {
			return nil, err
		}
		return &Struct{
			base:      prBase(valT.Type()),
			strctType: valT.StructType(),
			vals:      vals,
		}, nil
	case *values.Slice:
		vals, err := sliceToProxyValues(valT)
		if err != nil {
			return nil, err
		}
		return &Slice{
			base:      prBase(valT.Type()),
			sliceType: valT.SliceType(),
			vals:      vals,
		}, nil
	default:
		return nil, errors.Errorf("cannot convert %T to a proxy value", valT)
	}
}

// ToProxies converts GX values to proxy values keeping only the type of the value.
func ToProxies(vals []values.Value, typs []*ir.Field) ([]Value, error) {
	pVals := make([]Value, len(vals))
	for i, val := range vals {
		var err error
		pVals[i], err = ToProxy(val, typs[i].Type())
		if err != nil {
			return nil, err
		}
	}
	return pVals, nil
}

func structToProxyValues(st *values.Struct) (map[string]Value, error) {
	fields := st.StructType().Fields.Fields()
	prs := make(map[string]Value, len(fields))
	for _, field := range fields {
		var err error
		prs[field.Name.Name], err = ToProxy(st.FieldValue(field.Name.Name), field.Type())
		if err != nil {
			return nil, fmt.Errorf("cannot set field %s in structure %s: %w", field.Name.Name, st.Type().String(), err)
		}
	}
	return prs, nil
}

func sliceToProxyValues(sl *values.Slice) ([]Value, error) {
	elementType := sl.SliceType().DType
	prs := make([]Value, sl.Size())
	for i := range prs {
		var err error
		prs[i], err = ToProxy(sl.Element(i), elementType)
		if err != nil {
			return nil, err
		}
	}
	return prs, nil
}
