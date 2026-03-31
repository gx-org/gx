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

// Package concrete computes concrete types from generic types.
package concrete

import (
	"fmt"
	"go/ast"

	"github.com/pkg/errors"
	"github.com/gx-org/gx/build/fmterr"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/build/ir/irkind"
	"github.com/gx-org/gx/interp/elements"
)

func concreteTypeParam(fr ir.Evaluator, src ast.Expr, tp *ir.TypeParam) (ir.Type, error) {
	typeExpr := &ir.Ident{
		Src:  tp.Field.Name,
		Stor: tp,
	}
	el, err := fr.EvalExpr(typeExpr)
	if err != nil {
		return nil, fmterr.Internalf(fr.File().FileSet(), src, "cannot cast to %s: %v", tp.ReferString(fr.File()), err)
	}
	elType, isType := el.(ir.Type)
	if !isType {
		return nil, fmterr.Internalf(fr.File().FileSet(), src, "element %T is not a type", el)
	}
	return elType, nil
}

func evalAxisExpr(fr ir.Evaluator, src ast.Expr, axis ir.AxisLengths, x ir.Expr) ([]ir.AxisLengths, ir.CompEvalError, error) {
	el, err := fr.EvalExpr(x)
	if err != nil {
		return []ir.AxisLengths{axis}, nil, err
	}
	elCan, isCan := el.(ir.Canonical)
	if !isCan {
		return []ir.AxisLengths{axis}, nil, err
	}
	canX, cpErr, err := elCan.Expr()
	if cpErr != nil {
		return []ir.AxisLengths{axis}, cpErr, err
	}
	if err != nil {
		return []ir.AxisLengths{axis}, nil, err
	}
	slice, isSlice := canX.(*ir.SliceLitExpr)
	if !isSlice {
		return []ir.AxisLengths{&ir.AxisExpr{X: canX}}, nil, nil
	}
	ext := make([]ir.AxisLengths, len(slice.Elts))
	for i, elt := range slice.Elts {
		ext[i] = &ir.AxisExpr{X: elt}
	}
	return ext, nil, nil
}

func concreteAxisLength(fr ir.Evaluator, src ast.Expr, axis ir.AxisLengths) ([]ir.AxisLengths, ir.CompEvalError, error) {
	switch axisT := axis.(type) {
	case *ir.AxisExpr:
		return evalAxisExpr(fr, src, axis, axisT.X)
	case *ir.AxisInfer:
		return concreteAxisLength(fr, src, axisT.X)
	case *ir.AxisStmt:
		return evalAxisExpr(fr, src, axis, axisT.AsExpr())
	default:
		return []ir.AxisLengths{axis}, nil, fmterr.Errorf(fr.File().FileSet(), src, "cannot convert axis %s:%T to a concrete axis", axisT.SourceString(fr.File()), axisT)
	}
}

func concreteAxisLengths(fr ir.Evaluator, src ast.Expr, axes []ir.AxisLengths) ([]ir.AxisLengths, ir.CompEvalError, error) {
	var crs []ir.AxisLengths
	for _, ax := range axes {
		crAxis, cpErr, err := concreteAxisLength(fr, src, ax)
		if cpErr != nil || err != nil {
			return axes, cpErr, err
		}
		crs = append(crs, crAxis...)
	}
	return crs, nil, nil
}

func concreteRank(fr ir.Evaluator, src ast.Expr, rnk ir.ArrayRank) (ir.ArrayRank, ir.CompEvalError, error) {
	switch rnkT := rnk.(type) {
	case *ir.Rank:
		axes, cpErr, err := concreteAxisLengths(fr, src, rnkT.Ax)
		return &ir.Rank{Src: rnkT.Src, Ax: axes}, cpErr, err
	case *ir.RankInfer:
		return concreteRank(fr, src, rnkT.Rnk)
	default:
		return rnk, nil, fmterr.Errorf(fr.File().FileSet(), src, "cannot convert rank %s:%T to a concrete rank", rnk.SourceString(fr.File()), rnkT)
	}
}

func concreteArrayType(fr ir.Evaluator, src ast.Expr, tp ir.ArrayType) (ir.Type, ir.CompEvalError, error) {
	dtype, cpErr, err := Concrete(fr, src, tp.DataType())
	if cpErr != nil || err != nil {
		return tp, cpErr, err
	}
	rank, cpErr, err := concreteRank(fr, src, tp.Rank())
	if cpErr != nil || err != nil {
		return tp, cpErr, err
	}
	return ir.NewArrayType(src, dtype, rank), nil, nil
}

// Concrete converts a type to its concrete type given the context.
func Concrete(fr ir.Evaluator, src ast.Expr, tp ir.Type) (ir.Type, ir.CompEvalError, error) {
	if irkind.IsConcrete(tp.Kind()) {
		return tp, nil, nil
	}
	switch tpT := tp.(type) {
	case *ir.TypeParam:
		typ, err := concreteTypeParam(fr, src, tpT)
		return typ, nil, err
	case ir.ArrayType:
		return concreteArrayType(fr, src, tpT)
	default:
		return tp, nil, fmterr.Errorf(fr.File().FileSet(), src, "cannot compute the concrete type of %s:%T", tp.ReferString(fr.File()), tpT)
	}
}

// Frame storing key,value pairs.
type Frame interface {
	Define(id *ast.Ident, el ir.Element)
}

// Define field variables from an element.
func Define(ev ir.Evaluator, fr Frame, field *ir.Field, el ir.Element) error {
	fr.Define(field.Name, el)
	arrayType, isArrayType := field.Type().(ir.ArrayType)
	if !isArrayType {
		return nil
	}
	elWithAxes, hasAxes := el.(elements.WithAxes)
	if !hasAxes {
		return errors.Errorf("cannot define parameter %s (type: %s) from argument type %T", field.Name.Name, field.Type().ReferString(ev.File()), el)
	}
	argAxes, err := elWithAxes.Axes(ev)
	if err != nil {
		return fmt.Errorf("cannot define parameter %s (type: %s): %w", field.Name.Name, field.Type().ReferString(ev.File()), err)
	}
	argElts := argAxes.Elements()
	paramAxes := arrayType.Rank().Axes()
	for i, paramAxis := range paramAxes {
		axisStmt, isStmt := paramAxis.(*ir.AxisStmt)
		if !isStmt {
			continue
		}
		if axisStmt.Type().Kind() == irkind.Slice {
			fr.Define(axisStmt.Src, elements.NewSlice(ir.IntLenSliceType(), argElts[i:]))
			return nil
		}
		if i >= len(argElts) {
			return fmt.Errorf("cannot define axis length %s (position: %d) in parameter %s (type: %s): not enough axes", axisStmt.Src.Name, i, field.Name.Name, field.Type().ReferString(ev.File()))
		}
		fr.Define(axisStmt.Src, argElts[i])
	}
	return nil
}
