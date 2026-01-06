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
	"go/ast"
	"reflect"

	"github.com/pkg/errors"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/interp/canonical"
	"github.com/gx-org/gx/interp/elements"
	"github.com/gx-org/gx/interp/evaluator"
	"github.com/gx-org/gx/interp/fun"
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

func valEqual(x, y Element) (bool, error) {
	xEl, err := elements.ConstantFromElement(x)
	if err != nil {
		return false, err
	}
	if xEl == nil {
		return false, nil
	}
	yEl, err := elements.ConstantFromElement(y)
	if err != nil {
		return false, err
	}
	if yEl == nil {
		return false, nil
	}
	return equalArray(xEl, yEl), nil
}

func axesFromType(ev ir.Evaluator, typ ir.Type) (*elements.Slice, error) {
	aTyp, ok := typ.(ir.ArrayType)
	if !ok {
		return nil, nil
	}
	rank := aTyp.Rank()
	axes := rank.Axes()
	elts := make([]ir.Element, len(axes))
	for i, ax := range axes {
		var err error
		elts[i], err = ev.EvalExpr(ax.AsExpr())
		if err != nil {
			return nil, err
		}
	}
	return elements.NewSlice(ir.IntLenSliceType(), elts), nil
}

// NewRuntimeValue creates a new runtime value given an expression in a file.
func NewRuntimeValue(file *ir.File, store ir.Storage) (ir.Element, error) {
	ref := &ir.Ident{Src: store.NameDef(), Stor: store}
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
			fields[field.Name.Name], err = NewRuntimeValue(file, field.Storage())
			if err != nil {
				return nil, err
			}
		}
		return elements.NewStruct(typT, fields), nil
	case *ir.NamedType:
		under, err := NewRuntimeValue(file, typT.Underlying.Val())
		if err != nil {
			return nil, err
		}
		return fun.NewNamedType(NewFunc, typT, under.(elements.Copier)), nil
	case ir.ArrayType:
		if !ir.IsStatic(typT.DataType()) {
			return NewArray(typT), nil
		}
	case *ir.FuncType:
		return NewFunc(&ir.FuncLit{
			Src:   &ast.FuncLit{Type: typT.Src},
			FType: typT,
		}, nil), nil
	}
	return NewVariable(elements.NewNodeAt(file, store)), nil
}
