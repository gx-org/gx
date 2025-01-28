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

	"github.com/gx-org/gx/build/ir"
)

func defaultNumberType(scope scoper, expr exprNode) ir.Type {
	typ, ok := expr.resolveType(scope)
	if !ok {
		return invalid.irType()
	}
	irType := typ.irType()
	defaultType := ir.DefaultNumberType(irType.Kind())
	return defaultType
}

type numberCastExpr struct {
	ext ir.NumberCastExpr
	x   exprNode
	typ typeNode
}

var _ exprScalar = (*numberCastExpr)(nil)

func castNumber(scope scoper, expr exprNode, want ir.Type) (exprNode, typeNode, bool) {
	if want.Kind() == ir.UnknownKind {
		// No specification on what we want.
		// For example:
		//   a := 5.2
		// Then we cast the number, 5.2 in this example, to a default type.
		want = defaultNumberType(scope, expr)
	}
	if want.Kind() == ir.ArrayKind {
		// The required type is an array. For example:
		//   a := 5 * [2]float32{1, 2}
		// We cast the number, 5 in this example, to the data type of the array.
		underlying := ir.Underlying(want)
		arrayType, ok := underlying.(ir.ArrayType)
		if !ok {
			scope.err().AppendInternalf(expr.source(), "type %T has TensorKind but is not an *ir.ArrayType", underlying)
			return expr, invalid, false
		}
		want = arrayType.DataType()
	}
	if !ir.CanBeNumber(want.Kind()) {
		scope.err().Appendf(expr.source(), "cannot use a number as %v", want)
		return expr, invalid, false
	}
	typ, wantOk := toTypeNode(scope, want)
	numberType, numberOk := expr.resolveType(scope)
	if ir.IsFloat(numberType.kind()) && ir.IsInteger(typ.kind()) {
		scope.err().Appendf(expr.source(), "cannot use %s (untyped FLOAT constant) as %s value", expr.String(), typ.String())

	}
	return &numberCastExpr{
		ext: ir.NumberCastExpr{
			Typ: want,
		},
		x:   expr,
		typ: typ,
	}, typ, typ.kind() != ir.InvalidKind && wantOk && numberOk
}

func (n *numberCastExpr) source() ast.Node {
	return n.x.source()
}

func (n *numberCastExpr) scalar() ir.StaticExpr {
	if n.ext.X != nil {
		return &n.ext
	}
	n.ext.X = n.x.buildExpr()
	return &n.ext
}

func (n *numberCastExpr) resolveType(scope scoper) (typeNode, bool) {
	return typeNodeOk(n.typ)
}

func (n *numberCastExpr) expr() ast.Expr {
	return n.x.expr()
}

func (n *numberCastExpr) buildExpr() ir.Expr {
	return n.scalar()
}

func (n *numberCastExpr) String() string {
	return "numberCastExpr"
}
