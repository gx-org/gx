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

func typeInclude(fetcher ir.Fetcher, set ir.Type, typ ir.Type) bool {
	isIn, err := ir.TypeInclude(fetcher, set, typ)
	if err != nil {
		return fetcher.Err().Append(err)
	}
	if !isIn {
		return fetcher.Err().Appendf(typ.Node(), "%s does not satisfy %s",
			typ.ReferString(fetcher.File()),
			set.ReferString(fetcher.File()))
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
	irExpr, cpErr, err := irVal.Expr(fetcher, expr.Expr())
	if cpErr != nil {
		return expr, fetcher.Err().AppendAt(expr.Node(), cpErr)
	}
	if err != nil {
		return expr, fetcher.Err().AppendAt(expr.Node(), err)
	}
	return irExpr, true
}

// AssignTo checks if an expression can be assigned to a generic non-type parameter.
func AssignTo(fetcher ir.Fetcher, x ir.Expr, genAxis *ir.GenericNonTypeParam, tp ir.Type) bool {
	isAssignable, err := x.Type().AssignableTo(fetcher, tp)
	if err != nil {
		return fetcher.Err().Append(err)
	}
	if !isAssignable {
		from := fetcher.File()
		return fetcher.Err().Appendf(x.Node(), "cannot use %s (type: %s) as %s value in assignment for type parameter %s", x.SourceString(from), x.Type().ReferString(from), tp.ReferString(from), genAxis.NameDef().Name)
	}
	return true
}
