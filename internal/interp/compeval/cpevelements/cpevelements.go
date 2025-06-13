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

func sliceElementFromIRType(fetcher ir.Fetcher, src ast.Expr, typ ir.Type) (*elements.Slice, error) {
	aTyp, ok := typ.(ir.ArrayType)
	if !ok {
		return nil, nil
	}
	rank := aTyp.Rank()
	axes := rank.Axes()
	elts := make([]elements.Element, len(axes))
	exprs := make([]ir.AssignableExpr, len(axes))
	for i, ax := range axes {
		exprs[i] = ax.AxisValue()
		if exprs[i] == nil {
			infer, inferOk := ax.(*ir.AxisInfer)
			if !inferOk {
				return nil, fmterr.Internalf(fetcher.File().FileSet(), src, "%T is not %s", ax, reflect.TypeFor[*ir.AxisInfer]())
			}
			store := &ir.AnonymousStorage{
				Src: infer.Src,
				Typ: ir.IntLenType(),
			}
			exprs[i] = &ir.ValueRef{
				Src:  infer.Src,
				Stor: store,
			}
			elts[i] = NewVariable(elements.NewNodeAt[ir.Storage](fetcher.File(), store))
			continue
		}
		cVal, err := fetcher.Eval(exprs[i])
		if err != nil {
			return nil, err
		}
		elts[i], ok = cVal.(elements.Element)
		if !ok {
			return nil, fmterr.Internalf(fetcher.File().FileSet(), src, "%T is not %s", cVal, reflect.TypeFor[elements.Element]())
		}
	}
	sliceSrc := elements.NewExprAt(
		fetcher.File(),
		ir.NewSliceOfAxisLengths(src, exprs),
	)
	return elements.NewSlice(sliceSrc, elts), nil
}

// NewRuntimeValue creates a new runtime value given an expression in a file.
func NewRuntimeValue(ctx evaluator.Context, src ir.AssignableExpr) (elements.Element, error) {
	return newRuntimeValue(ctx, src, src.Type())
}

func newRuntimeValue(ctx evaluator.Context, src ir.AssignableExpr, typ ir.Type) (elements.Element, error) {
	switch typT := typ.(type) {
	case *ir.TypeParam:
		return newTypeParam(elements.NewExprAt(ctx.File(), src), typT), nil
	case *ir.StructType:
		fields := make(map[string]elements.Element)
		for _, field := range typT.Fields.Fields() {
			var err error
			fields[field.Name.Name], err = newRuntimeValue(ctx, src, field.Type())
			if err != nil {
				return nil, err
			}
		}
		return elements.NewStruct(typT, elements.NewValueAt(ctx.File(), src), fields), nil
	case *ir.NamedType:
		under, err := newRuntimeValue(ctx, src, typT.Underlying.Typ)
		if err != nil {
			return nil, err
		}
		return elements.NewNamedType(ctx.Evaluation().Evaluator().NewFunc, typT, under.(elements.Copier)), nil
	case *ir.SliceType:
		return newSlice(elements.NewExprAt(ctx.File(), src), typT), nil
	case ir.ArrayType:
		return NewArray(elements.NewExprAt(ctx.File(), src), typT), nil
	default:
		return nil, errors.Errorf("cannot convert %T to a compeval value: not supported", typT)
	}
}
