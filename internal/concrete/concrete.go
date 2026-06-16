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
	"go/ast"

	"github.com/gx-org/gx/build/fmterr"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/build/ir/irkind"
)

func concreteTypeParam(fr ir.Evaluator, src ast.Expr, tp *ir.GenericTypeParam) (ir.Type, error) {
	typeExpr := &ir.Ident{
		Src:  tp.OrigField().Name,
		Stor: tp,
	}
	el, err := fr.EvalExpr(typeExpr)
	if err != nil {
		return nil, fmterr.Internalf(fr.File().FileSet(), src, "cannot cast to %s: %v", tp.ReferString(fr.File()), err)
	}
	elType, isType := ir.BareValue(el).(ir.Type)
	if !isType {
		return nil, fmterr.Internalf(fr.File().FileSet(), src, "element %T is not a type", el)
	}
	return elType, nil
}

func evalAxisExpr(fr ir.Evaluator, src ast.Expr, axis ir.AxisLengths, x ir.Expr) ([]ir.AxisLengths, error) {
	el, err := fr.EvalExpr(x)
	if err != nil {
		return []ir.AxisLengths{axis}, err
	}
	elCan, isCan := el.(ir.Canonical)
	if !isCan {
		return []ir.AxisLengths{axis}, err
	}
	xs, err := ir.ToExpr(fr, src, elCan)
	var axlens []ir.AxisLengths
	for _, x := range xs {
		sliceLit, isSliceLit := x.(*ir.SliceLitExpr)
		if !isSliceLit {
			axlens = append(axlens, &ir.AxisExpr{X: x})
			continue
		}
		for _, elt := range sliceLit.Elts {
			axlens = append(axlens, &ir.AxisExpr{X: elt})
		}
	}
	return axlens, err
}

func concreteAxisLength(fr ir.Evaluator, src ast.Expr, axis ir.AxisLengths) ([]ir.AxisLengths, error) {
	switch axisT := axis.(type) {
	case *ir.AxisExpr:
		return evalAxisExpr(fr, src, axis, axisT.X)
	case *ir.AxisInfer:
		return concreteAxisLength(fr, src, axisT.X)
	default:
		return []ir.AxisLengths{axis}, fmterr.Errorf(fr.File().FileSet(), src, "cannot convert axis %s:%T to a concrete axis", axisT.SourceString(fr.File()), axisT)
	}
}

func concreteAxisLengths(fr ir.Evaluator, src ast.Expr, axes []ir.AxisLengths) ([]ir.AxisLengths, error) {
	var crs []ir.AxisLengths
	for _, ax := range axes {
		crAxis, err := concreteAxisLength(fr, src, ax)
		if err != nil {
			return axes, err
		}
		crs = append(crs, crAxis...)
	}
	return crs, nil
}

func concreteRank(fr ir.Evaluator, src ast.Expr, rnk ir.ArrayRank) (ir.ArrayRank, error) {
	switch rnkT := rnk.(type) {
	case *ir.Rank:
		axes, err := concreteAxisLengths(fr, src, rnkT.Ax)
		return &ir.Rank{Src: rnkT.Src, Ax: axes}, err
	case *ir.RankInfer:
		return concreteRank(fr, src, rnkT.Rnk)
	default:
		return rnk, fmterr.Errorf(fr.File().FileSet(), src, "cannot convert rank %s:%T to a concrete rank", rnk.SourceString(fr.File()), rnkT)
	}
}

func concreteArrayType(fr ir.Evaluator, src ast.Expr, tp ir.ArrayType) (ir.Type, error) {
	dtype, err := Concrete(fr, src, tp.DataType())
	if err != nil {
		return tp, err
	}
	rank, err := concreteRank(fr, src, tp.Rank())
	if err != nil {
		return tp, err
	}
	return ir.NewArrayType(src, dtype, rank), nil
}

// Concrete converts a type to its concrete type given the context.
func Concrete(fr ir.Evaluator, src ast.Expr, tp ir.Type) (ir.Type, error) {
	if irkind.IsConcrete(tp.Kind()) {
		return tp, nil
	}
	switch tpT := tp.(type) {
	case *ir.GenericTypeParam:
		typ, err := concreteTypeParam(fr, src, tpT)
		return typ, err
	case ir.ArrayType:
		if tpT.Rank().IsAtomic() {
			// Avoid a infinite loop on the data type of an array.
			return tp, nil
		}
		return concreteArrayType(fr, src, tpT)
	default:
		return tp, fmterr.Errorf(fr.File().FileSet(), src, "cannot compute the concrete type of %s:%T", tp.ReferString(fr.File()), tpT)
	}
}

// Frame storing key,value pairs.
type Frame interface {
	Define(id *ast.Ident, el ir.Element)
}
