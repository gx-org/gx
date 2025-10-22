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
	"math/big"

	"github.com/gx-org/gx/build/ir"
)

func isZeroExpr(expr ir.Value) bool {
	switch exprT := expr.(type) {
	case *ir.NumberInt:
		return big.NewInt(0).Cmp(exprT.Val) == 0
	case *ir.NumberCastExpr:
		return isZeroExpr(exprT.X)
	default:
		return false
	}
}

func instantiateAxisExpr(fetcher ir.Fetcher, axis *ir.AxisExpr) ([]ir.AxisLengths, bool) {
	val, ok := instantiateExpr(fetcher, axis.X)
	if !ok {
		return nil, false
	}
	if isZeroExpr(val) {
		return nil, true
	}
	switch valT := val.(type) {
	case *ir.SliceLitExpr:
		axes := make([]ir.AxisLengths, len(valT.Elts))
		for i, el := range valT.Elts {
			axes[i] = &ir.AxisExpr{Src: axis.Src, X: el}
		}
		return axes, true
	case ir.AssignableExpr:
		return []ir.AxisLengths{&ir.AxisExpr{Src: axis.Src, X: valT}}, true
	default:
		return nil, fetcher.Err().AppendInternalf(axis.Src, "cannot instantiate axis %s: value type %T not supported", axis.String(), valT)
	}
}

func exprSource(e ir.Expr) ast.Expr {
	src := e.Source()
	if src == nil {
		return nil
	}
	srcE, _ := src.(ast.Expr)
	return srcE
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
	if _, ok := rank.(*ir.RankInfer); ok {
		return &ir.RankInfer{}, true
	}
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
		gType := extractTypeSpecialiser(group.Type.Typ)
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
func Instantiate(fetcher ir.Fetcher, fType *ir.FuncType) (*ir.FuncType, bool) {
	nameToType := make(map[string]ir.Type)
	for _, tParamValue := range fType.TypeParamsValues {
		nameToType[tParamValue.Field.Name.Name] = tParamValue.Typ
	}
	params, ok := cloneFields(fetcher, fType.Params, instantiateGroupType(fetcher), cloneField)
	if !ok {
		return fType, false
	}
	results, ok := cloneFields(fetcher, fType.Results, instantiateGroupType(fetcher), cloneField)
	if !ok {
		return fType, false
	}
	return &ir.FuncType{
		BaseType:         fType.BaseType,
		Receiver:         fType.Receiver,
		TypeParams:       fType.TypeParams,
		TypeParamsValues: append([]ir.TypeParamValue{}, fType.TypeParamsValues...),
		CompEval:         fType.CompEval,
		Params:           params,
		Results:          results,
	}, true
}
