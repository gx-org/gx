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
	"go/token"

	"github.com/gx-org/gx/build/fmterr"
	"github.com/gx-org/gx/build/ir"
)

type binaryExpr struct {
	ext ir.BinaryExpr

	typ  typeNode
	x, y exprNode

	val ir.Atomic
}

var _ exprNumber = (*binaryExpr)(nil)

func processBinaryExpr(owner owner, expr *ast.BinaryExpr) (exprNode, bool) {
	x, xOk := processExpr(owner, expr.X)
	y, yOk := processExpr(owner, expr.Y)
	return &binaryExpr{
		ext: ir.BinaryExpr{
			Src: expr,
		},
		x: x,
		y: y,
	}, xOk && yOk
}

func (n *binaryExpr) buildExpr() ir.Expr {
	if n.val != nil {
		return n.val
	}
	n.ext.X = n.x.buildExpr()
	n.ext.Y = n.y.buildExpr()
	n.ext.Typ = n.typ.buildType()
	return &n.ext
}

func (n *binaryExpr) source() ast.Node {
	return n.expr()
}

func (n *binaryExpr) expr() ast.Expr {
	return n.ext.Src
}

func (n *binaryExpr) castTo(eval evaluator) (exprScalar, []*ir.ValueRef, bool) {
	xEval, unknownsX, okX := castExprTo(eval, n.x)
	if !okX {
		return nil, nil, false
	}
	yEval, unknownsY, okY := castExprTo(eval, n.y)
	if !okY {
		return nil, nil, false
	}
	ok := okX && okY
	unknowns := append(unknownsX, unknownsY...)
	if !ok {
		return nil, unknowns, false
	}
	n.typ, ok = toTypeNode(eval.scoper(), eval.want())
	n.ext.Typ = n.typ.buildType()
	if !ok {
		return nil, nil, false
	}
	n.x = xEval
	n.y = yEval
	n.ext.X = xEval.buildExpr()
	n.ext.Y = yEval.buildExpr()
	n.val = eval.cast(&n.ext)
	return n, unknowns, ok
}

func (n *binaryExpr) scalar() ir.Atomic {
	return n.val
}

func (n *binaryExpr) checkKind(scope scoper, x exprNode, typ ir.Type, appendErr bool) (isScalar bool, arrayType *ir.ArrayType, ok bool) {
	isScalar = ir.IsAtomic(typ.Kind())
	var isArray bool
	arrayType, isArray = typ.(*ir.ArrayType)
	if !isScalar && !isArray {
		if appendErr {
			scope.err().Appendf(x.source(), "invalid operation: operator %s not defined on type %s", n.ext.Src.Op.String(), typ.String())
		}
		ok = false
		return
	}
	ok = true
	return
}

func (n *binaryExpr) resolveOperator(scope scoper, ops typeNode) (result typeNode, castNumber, ok bool) {
	result = ops
	opsKind := ops.kind()
	array, isArray := ops.(arrayTypeNode)
	if isArray {
		opsKind = array.dtype().kind()
	}
	op := n.ext.Src.Op
	switch op {
	case token.ADD, token.MUL, token.SUB, token.QUO:
		ok = ir.IsAtomic(opsKind)
	case token.EQL, token.GTR, token.LSS, token.NEQ, token.LEQ, token.GEQ:
		ok = true
		result = &arrayType{
			dtyp: boolType,
			rnk:  array.rank(),
		}
		if isArray {
			result, ok = array.convertTo(scope, n, result)
		}
		castNumber = true
	default:
		scope.err().Appendf(n.ext.Src, "token %s not supported", n.ext.Src.Op.String())
		return invalid, false, false
	}
	if !ok {
		scope.err().Appendf(n.ext.Src, "invalid operation: operator %s not defined on %s", op.String(), ops.kind())
		return invalid, false, false
	}
	return
}

func (n *binaryExpr) resolveOperands(scope scoper) (typeNode, bool) {
	xType, xOk := n.x.resolveType(scope)
	yType, yOk := n.y.resolveType(scope)
	if !xOk || !yOk {
		return invalid, false
	}
	// Both operands are numbers, so this binary expression becomes a number.
	if xType.kind() == ir.NumberKind && yType.kind() == ir.NumberKind {
		return numberType, true
	}
	// Only one operand is a number, so we cast the number operand to the other type operand.
	if xType.kind() == ir.NumberKind {
		n.x, xType, xOk = buildNumberNode(scope, n.x, yType.buildType())
	}
	if yType.kind() == ir.NumberKind {
		n.y, yType, yOk = buildNumberNode(scope, n.y, xType.buildType())
	}
	if !xOk || !yOk {
		return invalid, false
	}
	// No operand is a number: check that we have a scalar or an array.
	xGXType := xType.buildType()
	xIsScalar, xArrayType, xOk := n.checkKind(scope, n.x, xGXType, true)
	yGXType := yType.buildType()
	yIsScalar, yArrayType, yOk := n.checkKind(scope, n.y, yGXType, xOk)
	if !xOk || !yOk {
		return invalid, false
	}

	var scalarType ir.Type
	var arrayType *ir.ArrayType
	var arrayTypeNode typeNode
	if xIsScalar && yArrayType != nil {
		scalarType = xGXType
		arrayType = yArrayType
		arrayTypeNode = yType
	}
	if yIsScalar && xArrayType != nil {
		scalarType = yGXType
		arrayType = xArrayType
		arrayTypeNode = xType
	}
	if scalarType != nil && arrayType != nil {
		// We have a scalar and an array:
		// check that the scalar matches with the array data type.
		dtype := arrayType.DataType()
		eq, err := dtype.Equal(scope.evalFetcher(), scalarType)
		if err != nil {
			scope.err().AppendInternalf(n.source(), "cannot compare %s and %s: %v", xType.String(), yType.String(), err)
			n.typ, n.ext.Typ = invalidType()
			return typeNodeOk(n.typ)
		}
		if !eq {
			scope.err().Appendf(n.source(), "mismatched types %s and %s", scalarType.String(), dtype.String())
			n.typ, n.ext.Typ = invalidType()
			return typeNodeOk(n.typ)
		}
		return arrayTypeNode, true
	}

	// Default case: check that both sides have the same type.
	eq, err := xGXType.Equal(scope.evalFetcher(), yGXType)
	if err != nil {
		scope.err().AppendAt(n.ext.Src, fmterr.Internal(err, "cannot compare %s and %s", xType.String(), yType.String()))
		return invalid, false
	}
	if !eq {
		scope.err().Appendf(n.source(), "mismatched types %s and %s", xType.String(), yType.String())
		return invalid, false
	}
	return xType, true
}

func (n *binaryExpr) resolveType(scope scoper) (typeNode, bool) {
	if n.typ != nil {
		return typeNodeOk(n.typ)
	}

	var ok bool
	n.typ, ok = n.resolveOperands(scope)
	if !ok {
		n.typ, n.ext.Typ = invalidType()
		return typeNodeOk(n.typ)
	}
	opResult, castNumber, ok := n.resolveOperator(scope, n.typ)
	if !ok {
		n.typ, n.ext.Typ = invalidType()
		return typeNodeOk(n.typ)
	}
	if castNumber && n.typ.kind() == ir.NumberKind {
		var xOk, yOk bool
		n.x, _, xOk = buildNumberNode(scope, n.x, float64Type.buildType())
		n.y, _, yOk = buildNumberNode(scope, n.y, float64Type.buildType())
		if !xOk || !yOk {
			n.typ, n.ext.Typ = invalidType()
			return typeNodeOk(n.typ)
		}
	}
	n.typ = opResult
	n.ext.Typ = n.typ.buildType()
	return typeNodeOk(n.typ)
}

func (n *binaryExpr) String() string {
	return n.typ.String()
}
