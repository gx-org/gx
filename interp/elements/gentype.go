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

package elements

import (
	"github.com/pkg/errors"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/concrete"
	"github.com/gx-org/gx/interp/engine"
)

// GenericType is an element representing a generic type.
type GenericType struct {
	typ ir.Type
}

var (
	_ concrete.TypeElement = (*GenericType)(nil)
	_ engine.Selector      = (*GenericType)(nil)
	_ engine.WithAxes      = (*GenericType)(nil)
)

// NewGenType returns a new element for a generic type.
func NewGenType(typ ir.Type) *GenericType {
	return &GenericType{typ: typ}
}

// Value of the type encapsulated by the element.
func (gt *GenericType) Value() ir.Type {
	return gt.typ
}

// Type of the element.
func (gt *GenericType) Type() ir.Type {
	return gt.typ.Type()
}

// Select returns an element encapsulating the type of a field of the structure type.
func (gt *GenericType) Select(env *engine.Env, expr *ir.SelectorExpr) (ir.Element, error) {
	name := expr.Stor.NameDef().Name
	structType, ok := ir.Underlying(gt.typ).(*ir.StructType)
	if !ok {
		return nil, errors.Errorf("cannot select field %s: type %s is not a structure", name, gt.typ.ReferString(nil))
	}
	field := structType.Fields.FindField(name)
	if field == nil {
		return nil, errors.Errorf("type %s has no field %s", gt.typ.ReferString(nil), name)
	}
	return NewGenType(field.Type()), nil
}

// Axes returns the axis lengths of the array type as a slice element.
func (gt *GenericType) Axes(ev ir.Evaluator) (ir.FixedSlice, error) {
	arrayType, ok := ir.Underlying(gt.typ).(ir.ArrayType)
	if !ok {
		return nil, errors.Errorf("cannot get the axis lengths of type %s: not an array type", gt.typ.ReferString(nil))
	}
	axes := arrayType.Rank().Axes()
	elts := make([]ir.Element, len(axes))
	for i, ax := range axes {
		var err error
		elts[i], err = ev.EvalExpr(ax.AsExpr())
		if err != nil {
			return nil, err
		}
	}
	slice, err := NewSlice(ir.IntSliceType(), elts)
	if err != nil {
		return nil, err
	}
	return slice, nil
}
