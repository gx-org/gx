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
	"fmt"
	"go/ast"
	"go/token"

	"github.com/gx-org/gx/build/ir"
)

type binaryExpr struct {
	src  *ast.BinaryExpr
	x, y exprNode
}

var _ exprNode = (*binaryExpr)(nil)

func processBinaryExpr(pscope procScope, expr *ast.BinaryExpr) (exprNode, bool) {
	x, xOk := processExpr(pscope, expr.X)
	y, yOk := processExpr(pscope, expr.Y)
	return &binaryExpr{
		src: expr,
		x:   x,
		y:   y,
	}, xOk && yOk
}

func (n *binaryExpr) source() ast.Node {
	return n.src
}

func (n *binaryExpr) checkKind(scope resolveScope, x exprNode, typ ir.Type, appendErr bool) (isScalar bool, arrayType ir.ArrayType, ok bool) {
	isScalar = ir.SupportOperators(typ)
	var isArray bool
	arrayType, isArray = typ.(ir.ArrayType)
	if !isScalar && !isArray {
		if appendErr {
			scope.err().Appendf(x.source(), "invalid operation: operator %s not defined on type %s", n.src.Op.String(), typ.String())
		}
		ok = false
		return
	}
	ok = true
	return
}

func (n *binaryExpr) determineOutputType(scope resolveScope, ops ir.Type) (result ir.Type, forceCastNumber, ok bool) {
	result = ops
	array, isArray := ops.(ir.ArrayType)
	if isArray {
		ops = array.DataType()
	}
	opsKind := ops.Kind()
	op := n.src.Op
	switch op {
	case token.ADD, token.MUL, token.SUB, token.QUO:
		ok = ir.SupportOperators(ops) && opsKind != ir.BoolKind
	case token.REM, token.SHL, token.SHR, token.AND, token.OR, token.XOR:
		ok = ir.SupportOperators(ops) && ir.IsInteger(ops)
	case token.EQL, token.GTR, token.LSS, token.NEQ, token.LEQ, token.GEQ:
		if isArray {
			result = ir.NewArrayType(&ast.ArrayType{}, ir.BoolType(), array.Rank())
		} else {
			result = ir.BoolType()
		}
		ok = true
		// Force cast of the operands to a default type for an expression like: 3 == 3.
		forceCastNumber = true
	case token.LAND, token.LOR:
		ok = ir.SupportOperators(ops) && opsKind == ir.BoolKind
	default:
		scope.err().Appendf(n.src, "token %s not supported", n.src.Op.String())
		return ir.InvalidType(), false, false
	}
	if !ok {
		scope.err().Appendf(n.src, "operator %s not defined on %s", op.String(), ops)
		return ir.InvalidType(), false, false
	}
	return
}

func (n *binaryExpr) buildOperands(scope resolveScope) (ir.AssignableExpr, ir.AssignableExpr, ir.Type) {
	xExpr, xOk := buildAExpr(scope, n.x)
	yExpr, yOk := buildAExpr(scope, n.y)
	if !xOk || !yOk {
		return xExpr, yExpr, ir.InvalidType()
	}
	xKind := xExpr.Type().Kind()
	yKind := yExpr.Type().Kind()
	// Both operands are numbers, so this binary expression becomes a number.
	if ir.IsNumber(xKind) && ir.IsNumber(yKind) {
		typ := ir.NumberIntType()
		// If either operand is a float, the result is a float number.
		// For example: 3.2+4 is a float number.
		if xKind == ir.NumberFloatKind || yKind == ir.NumberFloatKind {
			typ = ir.NumberFloatType()
		}
		return xExpr, yExpr, typ
	}
	// Only one operand is a number, so we cast the number operand to the other type operand.
	if ir.IsNumber(xKind) {
		xExpr, xOk = castNumber(scope, xExpr, yExpr.Type())
	}
	if ir.IsNumber(yKind) {
		yExpr, yOk = castNumber(scope, yExpr, xExpr.Type())
	}
	if !xOk || !yOk {
		return xExpr, yExpr, ir.InvalidType()
	}
	// No operand is a number: check that we have a scalar or an array.
	xType := xExpr.Type()
	xIsScalar, xArrayType, xOk := n.checkKind(scope, n.x, xType, true)
	yType := yExpr.Type()
	yIsScalar, yArrayType, yOk := n.checkKind(scope, n.y, yType, xOk)
	if !xOk || !yOk {
		return xExpr, yExpr, ir.InvalidType()
	}

	var scalarType ir.Type
	var arrayType ir.ArrayType
	if xIsScalar && yArrayType != nil {
		scalarType = xType
		arrayType = yArrayType
	}
	if yIsScalar && xArrayType != nil {
		scalarType = yType
		arrayType = xArrayType
	}
	compEval, compEvalOk := scope.compEval()
	if !compEvalOk {
		return xExpr, yExpr, ir.InvalidType()
	}
	if scalarType != nil && arrayType != nil {
		// We have a scalar and an array:
		// check that the scalar matches with the array data type.
		dtype := arrayType.DataType()
		eq, err := dtype.Equal(compEval, scalarType)
		if err != nil {
			scope.err().Append(err)
			return xExpr, yExpr, ir.InvalidType()
		}
		if !eq {
			scope.err().Appendf(n.source(), "mismatched types %s and %s", scalarType.String(), dtype.String())
			return xExpr, yExpr, ir.InvalidType()
		}
		return xExpr, yExpr, arrayType
	}

	// Default case: check that both sides have the same type.
	eq, err := xType.Equal(compEval, yType)
	if err != nil {
		scope.err().Append(err)
		return xExpr, yExpr, ir.InvalidType()
	}
	if !eq {
		scope.err().Appendf(n.source(), "mismatched types %s and %s", xType.String(), yType.String())
		return xExpr, yExpr, ir.InvalidType()
	}
	return xExpr, yExpr, xType
}

func (n *binaryExpr) buildExpr(scope resolveScope) (ir.Expr, bool) {
	expr := &ir.BinaryExpr{Src: n.src}
	expr.X, expr.Y, expr.Typ = n.buildOperands(scope)
	if isInvalid(expr.Typ) {
		return expr, false
	}
	outTyp, forceCastNumber, ok := n.determineOutputType(scope, expr.Typ)
	if !ok {
		return expr, false
	}
	if forceCastNumber && ir.IsNumber(expr.Typ.Kind()) {
		var xOk, yOk bool
		expr.X, xOk = castNumber(scope, expr.X, ir.UnknownType())
		expr.Y, yOk = castNumber(scope, expr.Y, ir.UnknownType())
		ok = xOk && yOk
	}
	expr.Typ = outTyp
	return expr, ok
}

func (n *binaryExpr) String() string {
	return fmt.Sprintf("%s %s %s", n.x.String(), n.src.Op.String(), n.y.String())
}
