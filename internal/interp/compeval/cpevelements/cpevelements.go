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
	"github.com/gx-org/gx/build/fmterr"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/interp/canonical"
	"github.com/gx-org/gx/interp/elements"
	"github.com/gx-org/gx/interp/evaluator"
)

type (
	// Element returned after an evaluation at compeval.
	Element interface {
		elements.NumericalElement
		canonical.Comparable
		fmt.Stringer

		// CanonicalExpr returns the canonical expression used for comparison.
		CanonicalExpr() canonical.Canonical
	}

	// IRArrayElement is an array with a IR representation of its axes.
	IRArrayElement interface {
		Axes(ir.Fetcher) (*elements.Slice, error)
	}
)

func toElement(x elements.NumericalElement) (Element, error) {
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

func sliceElementFromIRType(fetcher ir.Fetcher, typ ir.Type) (*elements.Slice, error) {
	aTyp, ok := typ.(ir.ArrayType)
	if !ok {
		return nil, nil
	}
	rank := aTyp.Rank()
	axes := rank.Axes()
	elts := make([]elements.Element, len(axes))
	for i, ax := range axes {
		cVal, err := fetcher.EvalExpr(ax)
		if err != nil {
			return nil, err
		}
		elts[i], ok = cVal.(elements.Element)
		if !ok {
			return nil, fmterr.Internal(errors.Errorf("%T is not %s", cVal, reflect.TypeFor[elements.Element]()))
		}
	}
	return elements.NewSlice(ir.IntLenSliceType(), elts), nil
}

// NewRuntimeValue creates a new runtime value given an expression in a file.
func NewRuntimeValue(ctx evaluator.Context, store ir.Storage) (elements.Element, error) {
	ref := &ir.ValueRef{Src: store.NameDef(), Stor: store}
	typ, ok := store.(ir.Type)
	if !ok { // Check if storage is a type itself.
		// If not, then get the type from the storage.
		typ = store.Type()
	}
	switch typT := typ.(type) {
	case *ir.TypeParam:
		return newTypeParam(elements.NewExprAt(ctx.File(), ref), typT), nil
	case *ir.StructType:
		fields := make(map[string]elements.Element)
		for _, field := range typT.Fields.Fields() {
			var err error
			fields[field.Name.Name], err = NewRuntimeValue(ctx, field.Storage())
			if err != nil {
				return nil, err
			}
		}
		return elements.NewStruct(typT, elements.NewValueAt(ctx.File(), ref), fields), nil
	case *ir.NamedType:
		under, err := NewRuntimeValue(ctx, typT.Underlying.Typ)
		if err != nil {
			return nil, err
		}
		return elements.NewNamedType(ctx.Evaluation().Evaluator().NewFunc, typT, under.(elements.Copier)), nil
	case ir.ArrayType:
		if !ir.IsStatic(typT.DataType()) {
			return NewArray(typT), nil
		}
	}
	return NewVariable(elements.NewNodeAt(ctx.File(), store)), nil
}
