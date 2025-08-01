// Copyright 2025 Google LLC
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

// Package cpevelements provides elements for the interpreter for compeval.
package cpevelements

import (
	"fmt"
	"reflect"

	"github.com/pkg/errors"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/interp/canonical"
	"github.com/gx-org/gx/interp/elements"
	"github.com/gx-org/gx/interp/evaluator"
	"github.com/gx-org/gx/interp"
)

// Element returned after an evaluation at compeval.
type Element interface {
	evaluator.NumericalElement
	canonical.Comparable
	fmt.Stringer

	// CanonicalExpr returns the canonical expression used for comparison.
	CanonicalExpr() canonical.Canonical
}

func toElement(x evaluator.NumericalElement) (Element, error) {
	el, ok := x.(Element)
	if !ok {
		return nil, errors.Errorf("cannot build static element: type %T does not implement %s", x, reflect.TypeFor[Element]().String())
	}
	return el, nil
}

func valEqual(x, y Element) bool {
	xEl := elements.ConstantFromElement(x)
	if xEl == nil {
		return false
	}
	yEl := elements.ConstantFromElement(y)
	if yEl == nil {
		return false
	}
	return equalArray(xEl, yEl)
}

func axesFromType(ev ir.Evaluator, typ ir.Type) (*interp.Slice, error) {
	aTyp, ok := typ.(ir.ArrayType)
	if !ok {
		return nil, nil
	}
	rank := aTyp.Rank()
	axes := rank.Axes()
	elts := make([]ir.Element, len(axes))
	for i, ax := range axes {
		var err error
		elts[i], err = ev.EvalExpr(ax)
		if err != nil {
			return nil, err
		}
	}
	return interp.NewSlice(ir.IntLenSliceType(), elts), nil
}

// NewRuntimeValue creates a new runtime value given an expression in a file.
func NewRuntimeValue(file *ir.File, newFunc interp.NewFunc, store ir.Storage) (ir.Element, error) {
	ref := &ir.ValueRef{Src: store.NameDef(), Stor: store}
	typ, ok := store.(ir.Type)
	if !ok { // Check if storage is a type itself.
		// If not, then get the type from the storage.
		typ = store.Type()
	}
	switch typT := typ.(type) {
	case *ir.TypeParam:
		return newTypeParam(elements.NewExprAt(file, ref), typT), nil
	case *ir.StructType:
		fields := make(map[string]ir.Element)
		for _, field := range typT.Fields.Fields() {
			var err error
			fields[field.Name.Name], err = NewRuntimeValue(file, newFunc, field.Storage())
			if err != nil {
				return nil, err
			}
		}
		return interp.NewStruct(typT, fields), nil
	case *ir.NamedType:
		under, err := NewRuntimeValue(file, newFunc, typT.Underlying.Typ)
		if err != nil {
			return nil, err
		}
		return interp.NewNamedType(newFunc, typT, under.(interp.Copier)), nil
	case ir.ArrayType:
		if !ir.IsStatic(typT.DataType()) {
			return NewArray(typT), nil
		}
	}
	return NewVariable(elements.NewNodeAt(file, store)), nil
}
