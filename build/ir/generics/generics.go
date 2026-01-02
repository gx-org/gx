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

// Package generics provides functions to handle generics with the GX IR.
package generics

import (
	"go/ast"
	"reflect"

	"github.com/gx-org/gx/build/ir"
)

func newTypeParamDefinition(ftype *ir.FuncType) map[string]ir.Type {
	defined := make(map[string]ir.Type)
	for _, typeParamValue := range ftype.TypeParamsValues {
		defined[typeParamValue.Field.Name.Name] = typeParamValue.Typ
	}
	return defined
}

func newAxisLengthsDefinition(ftype *ir.FuncType) map[string]ir.Element {
	axes := make(map[string]ir.Element)
	for _, axValue := range ftype.AxisLengths {
		axes[axValue.Name()] = axValue.Value
	}
	return axes
}

func typeInclude(fetcher ir.Fetcher, set ir.Type, typ ir.Type) bool {
	isIn, err := ir.TypeInclude(fetcher, set, typ)
	if err != nil {
		return fetcher.Err().Append(err)
	}
	if !isIn {
		return fetcher.Err().Appendf(typ.Node(), "%s does not satisfy %s",
			ir.TypeString(typ), ir.TypeString(set))
	}
	return true
}

func dtypeElement(expr ast.Expr) ast.Expr {
	aType, ok := expr.(*ast.ArrayType)
	if !ok {
		return expr
	}
	return dtypeElement(aType.Elt)
}

func instantiateExpr(fetcher ir.Fetcher, expr ir.Expr) (ir.Value, bool) {
	val, err := fetcher.EvalExpr(expr)
	if err != nil {
		return expr, fetcher.Err().Append(err)
	}
	irVal, ok := val.(ir.Canonical)
	if !ok {
		return expr, fetcher.Err().AppendInternalf(expr.Node(), "cannot convert %T to %s", val, reflect.TypeFor[ir.Canonical]())
	}
	irExpr, err := irVal.Expr()
	if err != nil {
		return expr, fetcher.Err().AppendAt(expr.Node(), err)
	}
	return irExpr, true
}
