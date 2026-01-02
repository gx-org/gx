// Copyright 2024 Google LLC
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

package builder

import (
	"go/ast"
	"reflect"

	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/build/ir/irkind"
)

// numberLit is the literal of a number.
type numberLit struct {
	ext ir.Number
}

var _ exprNode = (*numberLit)(nil)

func (n *numberLit) buildExpr(resolveScope) (ir.Expr, bool) {
	return n.ext, true
}

// Pos returns the position of the literal in the code.
func (n *numberLit) source() ast.Node {
	return n.ext.Source()
}

func (n *numberLit) String() string {
	return n.ext.String()
}

func castNumber(scope resolveScope, expr ir.Expr, target ir.Type) (*ir.NumberCastExpr, bool) {
	cast := &ir.NumberCastExpr{
		X:   expr,
		Typ: target,
	}
	if cast.Typ.Kind() == irkind.Unknown {
		// No specification on what we want.
		// For example:
		//   a := 5.2
		// Then we cast the number, 5.2 in this example, to a default type.
		cast.Typ = ir.DefaultNumberType(expr.Type().Kind())
	}
	if cast.Typ.Kind() == irkind.Array {
		// The required type is an array. For example:
		//   a := 5 * [2]float32{1, 2}
		// We cast the number, 5 in this example, to the data type of the array.
		underlying := ir.Underlying(cast.Typ)
		arrayType, ok := underlying.(ir.ArrayType)
		if !ok {

			return cast, scope.Err().AppendInternalf(expr.Source(), "type %T has %s but does not implement %s", underlying, irkind.Array.String(), reflect.TypeFor[ir.ArrayType]().Name())
		}
		cast.Typ = arrayType.DataType()
	}
	if !ir.CanBeNumber(cast.Typ) {
		return cast, scope.Err().Appendf(expr.Source(), "cannot use a number as %v", cast.Typ)
	}
	if ir.IsFloat(expr.Type()) && ir.IsInteger(cast.Typ) {
		return cast, scope.Err().Appendf(expr.Source(), "cannot use %s (untyped FLOAT constant) as %s value", expr.String(), cast.Typ.String())
	}
	return cast, true
}
