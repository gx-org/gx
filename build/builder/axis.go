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

// axisLengthNode defines the dimension of an array.
type axisLengthNode interface {
	exprNode
	axisLength() ir.AxisLength
	reconcileWith(scoper, axisLengthNode) (axisLengthNode, bool)
}

func processAxisLengthExpr(owner owner, array *ast.ArrayType) (axis axisLengthNode, ok bool) {
	if array.Len == nil {
		owner.err().Appendf(array, "array of slices is not supported. Please specify an array axis length using '[length]', '[_]', or '[...]' if the size is unknown, to specify a dense array")
		return nil, false
	}
	switch lenT := array.Len.(type) {
	case *ast.Ellipsis:
		return nil, true
	case *ast.Ident:
		if lenT.Name == "___" {
			return nil, true
		}
		if lenT.Name == "_" {
			return &genericAxisLength{
				ext: &ir.AxisEllipsis{Src: lenT},
			}, true
		}
		x, ok := processExpr(owner, array.Len)
		return newExprAxisLength(x), ok
	case ast.Expr:
		x, ok := processExpr(owner, array.Len)
		return newExprAxisLength(x), ok
	default:
		owner.err().Appendf(array, "array dimension type %T not supported", lenT)
		return nil, false
	}
}

func importAxis(scope scoper, irDim ir.AxisLength) (axisLengthNode, bool) {
	switch irDimT := irDim.(type) {
	case *ir.AxisEllipsis:
		dim := &genericAxisLength{ext: irDimT}
		if irDimT.X == nil {
			return dim, true
		}
		dim.x = newExprAxisLength(&irExprNode{x: irDimT.X})
		return dim, true
	case *ir.AxisExpr:
		dim := newExprAxisLength(&irExprNode{x: irDimT.X})
		_, ok := dim.resolveType(scope)
		return dim, ok
	default:
		scope.err().Appendf(irDim.Source(), "cannot import array axis: %T not supported", irDim)
		return nil, false
	}
}

func lengthEquals(scope scoper, x, y []axisLengthNode) bool {
	if len(x) != len(y) {
		return false
	}
	for i, xi := range x {
		eq, err := xi.axisLength().Equal(scope.evalFetcher(), y[i].axisLength())
		if err != nil {
			scope.err().Append(scope.err().FSet().Position(xi.source(), err))
			return false
		}
		if !eq {
			return false
		}
	}
	return true
}

// inferredFromLiteralAxisLength computed from a literal.
type inferredFromLiteralAxisLength struct {
	src ast.Expr // Point to the literal to report mismatch errors.
	val int
}

var _ axisLengthNode = (*inferredFromLiteralAxisLength)(nil)

func (dim *inferredFromLiteralAxisLength) scalar() *ir.AtomicValueT[ir.Int] {
	return &ir.AtomicValueT[ir.Int]{
		Src: dim.src,
		Val: ir.Int(dim.val),
		Typ: ir.IntLenType(),
	}
}

func (dim *inferredFromLiteralAxisLength) axisLength() ir.AxisLength {
	return &ir.AxisExpr{
		Src: dim.src,
		X:   dim.scalar(),
	}
}

func (dim *inferredFromLiteralAxisLength) expr() ast.Expr {
	return dim.src
}

func (dim *inferredFromLiteralAxisLength) buildExpr() ir.Expr {
	return dim.axisLength()
}

func (dim *inferredFromLiteralAxisLength) String() string {
	return "inferredAxisFromLiteral"
}

// resolveTypes recursively calls resolveTypes in the underlying subtree of nodes.
func (dim *inferredFromLiteralAxisLength) resolveType(scoper) (typeNode, bool) {
	return axisLengthType, true
}

func (dim *inferredFromLiteralAxisLength) reconcileWith(scope scoper, other axisLengthNode) (axisLengthNode, bool) {
	return dim, false
}

func (dim *inferredFromLiteralAxisLength) source() ast.Node {
	return dim.expr()
}

// exprAxisLength is a dimension specified with an expression
type exprAxisLength struct {
	ext *ir.AxisExpr
	x   exprNode
}

var _ axisLengthNode = (*exprAxisLength)(nil)

func newExprAxisLength(x exprNode) *exprAxisLength {
	return &exprAxisLength{
		ext: &ir.AxisExpr{
			Src: x.expr(),
		},
		x: x,
	}
}

func (dim *exprAxisLength) exprNode() exprNode {
	return dim.x
}

func (dim *exprAxisLength) resolveType(scope scoper) (typeNode, bool) {
	typ, ok := dim.x.resolveType(scope)
	if !ok {
		return invalid, false
	}
	want := ir.IntLenType()
	if ir.IsNumber(typ.kind()) {
		dim.x, typ, ok = castNumber(scope, dim.x, want)
		if !ok {
			return invalid, false
		}
	}
	eq, err := typ.irType().Equal(scope.evalFetcher(), want)
	if err != nil {
		return invalid, scope.err().AppendAt(dim.ext.Src, err)
	}
	if !eq {
		return invalid, scope.err().Appendf(dim.ext.Src, "cannot use %s as %s in axis length", typ.String(), want.String())
	}
	return typeNodeOk(typ)
}

func (dim *exprAxisLength) isEqual(scope scoper, other ir.AxisLength) bool {
	eq, err := dim.axisLength().Equal(scope.evalFetcher(), other)
	if err != nil {
		scope.err().Append(err)
		return false
	}
	return eq
}

const cannotInferAxisLength = "cannot infer array axis length"

func (dim *exprAxisLength) reconcileWith(scope scoper, other axisLengthNode) (axisLengthNode, bool) {
	if _, ok := dim.resolveType(scope); !ok {
		return dim, false
	}
	if _, ok := other.resolveType(scope); !ok {
		return dim, false
	}
	switch otherT := other.(type) {
	case *exprAxisLength:
		return dim, dim.isEqual(scope, otherT.axisLength())
	case *inferredFromLiteralAxisLength:
		return dim, dim.isEqual(scope, otherT.axisLength())
	case *genericAxisLength:
		if otherT.x == nil {
			return dim, true
		}
		return dim.reconcileWith(scope, otherT.x)
	default:
		return dim, scope.err().Appendf(dim.ext.Src, cannotInferAxisLength)
	}
}

func (dim *exprAxisLength) String() string {
	return dim.x.String()
}

func (dim *exprAxisLength) buildExpr() ir.Expr {
	return dim.axisLength()
}

func (dim *exprAxisLength) axisLength() ir.AxisLength {
	if dim.ext.X != nil {
		return dim.ext
	}
	dim.ext.X = dim.x.buildExpr()
	return dim.ext
}

func (dim *exprAxisLength) source() ast.Node {
	return dim.expr()
}

func (dim *exprAxisLength) expr() ast.Expr {
	return dim.x.expr()
}

// genericAxisLength is a dimension specified with _
type genericAxisLength struct {
	ext *ir.AxisEllipsis
	x   *exprAxisLength
}

var _ axisLengthNode = (*genericAxisLength)(nil)

func (dim *genericAxisLength) source() ast.Node {
	return dim.expr()
}

func (dim *genericAxisLength) expr() ast.Expr {
	return dim.ext.Src
}

func (dim *genericAxisLength) resolveType(scope scoper) (typeNode, bool) {
	if dim.x == nil {
		return axisLengthType, true
	}
	return dim.x.resolveType(scope)
}

func (dim *genericAxisLength) reconcileWithExpr(scope scoper, other *exprAxisLength) (axisLengthNode, bool) {
	if dim.x == nil {
		return &genericAxisLength{ext: dim.ext, x: other}, true
	}
	return dim, dim.x.isEqual(scope, other.axisLength())
}

func (dim *genericAxisLength) reconcileWith(scope scoper, other axisLengthNode) (axisLengthNode, bool) {
	switch otherT := other.(type) {
	case *genericAxisLength:
		if dim.x != nil && otherT.x == nil {
			return dim, scope.err().Appendf(dim.ext.Src, "cannot reconcile a resolved axis length with an unresolved axis length")
		}
		if dim.x == nil && otherT.x == nil {
			return dim, true
		}
		return dim.reconcileWithExpr(scope, otherT.x)
	case *exprAxisLength:
		return dim.reconcileWithExpr(scope, otherT)
	case *inferredFromLiteralAxisLength:
		exprDim := newExprAxisLength(otherT)
		return dim.reconcileWith(scope, exprDim)
	default:
		scope.err().Appendf(dim.ext.Src, "dimension type %T not supported", otherT)
		return dim, false
	}
}

func (dim *genericAxisLength) String() string {
	return dim.axisLength().String()
}

func (dim *genericAxisLength) buildExpr() ir.Expr {
	return dim.axisLength()
}

func (dim *genericAxisLength) axisLength() ir.AxisLength {
	if dim.x == nil {
		return dim.ext
	}
	dim.ext.X = dim.x.axisLength()
	return dim.ext
}
