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

package generics

import (
	"go/ast"

	"github.com/gx-org/gx/build/ir"
)

func instantiateAxisExpr(fetcher ir.Fetcher, axis *ir.AxisExpr) ([]ir.AxisLengths, bool) {
	ext := []ir.AxisLengths{
		&ir.AxisExpr{Src: axis.Src, X: axis.X},
	}
	val, ok := instantiateExpr(fetcher, axis.X)
	if !ok {
		return ext, false

	}
	expr, ok := val.(ir.AssignableExpr)
	if !ok {
		return ext, false
	}
	return []ir.AxisLengths{
		&ir.AxisExpr{Src: axis.Src, X: expr},
	}, ok
}

func exprSource(e ir.Expr) ast.Expr {
	src := e.Source()
	if src == nil {
		return nil
	}
	srcE, _ := src.(ast.Expr)
	return srcE
}

func extractAxisGroup(elts []ir.AssignableExpr) *ir.AxisGroup {
	if len(elts) != 1 {
		return nil
	}
	valRef, ok := elts[0].(*ir.ValueRef)
	if !ok {
		return nil
	}
	group, _ := valRef.Stor.(*ir.AxisGroup)
	return group
}

func instantiateAxisGroup(fetcher ir.Fetcher, axis *ir.AxisGroup) ([]ir.AxisLengths, bool) {
	expr, ok := instantiateExpr(fetcher, &ir.ValueRef{
		Src: &ast.Ident{
			NamePos: axis.Source().Pos(),
			Name:    axis.Name,
		},
		Stor: axis,
	})
	if !ok {
		return []ir.AxisLengths{axis}, false
	}
	switch exprT := expr.(type) {
	case *ir.AxisGroup:
		return []ir.AxisLengths{exprT}, true
	case *ir.SliceLitExpr:
		if group := extractAxisGroup(exprT.Elts); group != nil {
			return []ir.AxisLengths{group}, true
		}
		ext := make([]ir.AxisLengths, len(exprT.Elts))
		for i, elt := range exprT.Elts {
			ext[i] = &ir.AxisExpr{Src: exprSource(elt), X: elt}
		}
		return ext, true
	case *ir.Rank:
		return exprT.Ax, true
	case *ir.ValueRef:
		return []ir.AxisLengths{&ir.AxisGroup{Src: exprT.Src, Name: exprT.Src.Name}}, true
	default:
		return []ir.AxisLengths{axis}, fetcher.Err().AppendInternalf(axis.Src, "cannot process expression %v as axis group: %T not supported", exprT, exprT)
	}
}

func instantiateAxisInfer(fetcher ir.Fetcher, axis *ir.AxisInfer) ([]ir.AxisLengths, bool) {
	switch axis.Src.Name {
	case "_":
		return []ir.AxisLengths{axis}, true
	default:
		return []ir.AxisLengths{axis}, fetcher.Err().AppendInternalf(axis.Src, "unknown inference token %s", axis.Src.Name)
	}
}

func instantiateAxis(fetcher ir.Fetcher, axis ir.AxisLengths) ([]ir.AxisLengths, bool) {
	switch axisT := axis.(type) {
	case *ir.AxisExpr:
		return instantiateAxisExpr(fetcher, axisT)
	case *ir.AxisGroup:
		return instantiateAxisGroup(fetcher, axisT)
	case *ir.AxisInfer:
		if axisT.X != nil {
			return instantiateAxis(fetcher, axisT.X)
		}
		return instantiateAxisInfer(fetcher, axisT)
	default:
		return []ir.AxisLengths{axisT}, false
	}
}

func (i *array) instantiateRank(fetcher ir.Fetcher, rank ir.ArrayRank) (ir.ArrayRank, bool) {
	var axes []ir.AxisLengths
	ok := true
	for _, axis := range rank.Axes() {
		axs, axisOk := instantiateAxis(fetcher, axis)
		if !axisOk {
			ok = false
			continue
		}
		axes = append(axes, axs...)
	}
	return &ir.Rank{
		Src: rank.Source().(*ast.ArrayType),
		Ax:  axes,
	}, ok
}

func instantiateGroupType(fetcher ir.Fetcher) groupCloner {
	return func(fetcher ir.Fetcher, group *ir.FieldGroup) *ir.FieldGroup {
		ext := &ir.FieldGroup{
			Src:  group.Src,
			Type: group.Type,
		}
		gType := extractTypeParamName(group)
		if gType == nil {
			return ext
		}
		iType := gType.instantiate(fetcher)
		if iType == nil {
			return nil
		}
		ext.Type = &ir.TypeValExpr{X: group.Type.X, Typ: iType}
		return ext
	}
}

// Instantiate replaces data types either specified or inferred.
func Instantiate(fetcher ir.Fetcher, fExpr *ir.FuncValExpr) (*ir.FuncValExpr, bool) {
	params, ok := cloneFields(fetcher, fExpr.T.Params, instantiateGroupType(fetcher), cloneField)
	if !ok {
		return fExpr, false
	}
	results, ok := cloneFields(fetcher, fExpr.T.Results, instantiateGroupType(fetcher), cloneField)
	if !ok {
		return fExpr, false
	}
	fType := &ir.FuncType{
		BaseType:         fExpr.T.BaseType,
		Receiver:         fExpr.T.Receiver,
		TypeParams:       fExpr.T.TypeParams,
		TypeParamsValues: append([]ir.TypeParamValue{}, fExpr.T.TypeParamsValues...),
		CompEval:         fExpr.T.CompEval,
		Params:           params,
		Results:          results,
	}
	return &ir.FuncValExpr{
		X: fExpr.X,
		F: fExpr.F,
		T: fType,
	}, true
}
