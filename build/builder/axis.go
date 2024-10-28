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

// arrayAxes defines the dimension of an array.
type arrayAxes interface {
	exprNode() exprNode
	expr() ir.AxisLength
	resolve(scoper) bool
	reconcileWith(scoper, arrayAxes) (arrayAxes, bool)
}

func processArrayDimensionExpr(owner owner, array *ast.ArrayType) (axis arrayAxes, ok bool) {
	if array.Len == nil {
		owner.err().Appendf(array, "array of slices is not supported. Please specify an array dimension using '[length]', '[_]', or '[...]' if the size is unknown, to specify a dense array")
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
			return &genericDimension{
				ext: &ir.AxisEllipsis{Src: lenT},
			}, true
		}
		x, ok := processExpr(owner, array.Len)
		return newExprDimension(x), ok
	case ast.Expr:
		x, ok := processExpr(owner, array.Len)
		return newExprDimension(x), ok
	default:
		owner.err().Appendf(array, "array dimension type %T not supported", lenT)
		return nil, false
	}
}

func importAxis(scope scoper, irDim ir.AxisLength) (arrayAxes, bool) {
	switch irDimT := irDim.(type) {
	case *ir.AxisEllipsis:
		dim := &genericDimension{ext: irDimT}
		if irDimT.X == nil {
			return dim, true
		}
		dim.x = newExprDimension(&irExprNode{x: irDimT.X.X})
		return dim, true

	case *ir.AxisExpr:
		dim := newExprDimension(&irExprNode{x: irDimT.X})
		return dim, dim.resolve(scope)
	default:
		scope.err().Appendf(irDim.Source(), "cannot import array axis: %T not supported", irDim)
		return nil, false
	}
}

func dimEquals(scope scoper, x, y []arrayAxes) bool {
	if len(x) != len(y) {
		return false
	}
	for i, xi := range x {
		eq, err := xi.expr().Equal(scope.evalFetcher(), y[i].expr())
		if err != nil {
			scope.err().Append(scope.err().FSet().Position(xi.exprNode().source(), err))
			return false
		}
		if !eq {
			return false
		}
	}
	return true
}

// valueDimension computed from a literal.
type valueDimension struct {
	src *ast.CompositeLit
	val int
}

var _ arrayAxes = (*valueDimension)(nil)

func (dim *valueDimension) scalar() *ir.AtomicValueT[ir.Int] {
	return &ir.AtomicValueT[ir.Int]{
		Src: dim.src,
		Val: ir.Int(dim.val),
		Typ: ir.AxisLengthType(),
	}
}

func (dim *valueDimension) exprNode() exprNode {
	return &basicLit{
		ext: dim.scalar(),
		typ: axisLengthType,
	}
}

func (dim *valueDimension) expr() ir.AxisLength {
	return &ir.AxisExpr{
		Src: dim.src,
		X:   dim.scalar(),
	}
}

func (dim *valueDimension) resolve(scoper) bool {
	return true
}

func (dim *valueDimension) reconcileWith(scope scoper, other arrayAxes) (arrayAxes, bool) {
	return dim, scope.err().Appendf(dim.src, "cannot reconcile a dimension computed from a literal with %T", other)
}

// exprDimension is a dimension specified with an expression
type exprDimension struct {
	ext *ir.AxisExpr
	x   exprNode
}

func newExprDimension(x exprNode) *exprDimension {
	return &exprDimension{
		ext: &ir.AxisExpr{
			Src: x.expr(),
		},
		x: x,
	}
}

func (dim *exprDimension) exprNode() exprNode {
	return dim.x
}

func (dim *exprDimension) resolve(scope scoper) bool {
	typ, ok := dim.x.resolveType(scope)
	if !ok {
		return false
	}
	want := ir.AxisLengthType()
	if typ.kind() == ir.NumberKind {
		dim.x, typ, ok = buildNumberNode(scope, dim.x, want)
		if !ok {
			return false
		}
	}
	eq, err := typ.buildType().Equal(scope.evalFetcher(), want)
	if err != nil {
		return scope.err().AppendAt(dim.ext.Src, err)
	}
	if !eq {
		return scope.err().Appendf(dim.ext.Src, "cannot use %s as %s in axis length", typ.String(), want.String())
	}
	return eq
}

func (dim *exprDimension) isEqual(scope scoper, other ir.AxisLength) bool {
	eq, err := dim.expr().Equal(scope.evalFetcher(), other)
	if err != nil {
		scope.err().Append(err)
		return false
	}
	return eq
}

const cannotInferAxisLength = "cannot infer array axis length"

func (dim *exprDimension) reconcileWith(scope scoper, other arrayAxes) (arrayAxes, bool) {
	switch otherT := other.(type) {
	case *exprDimension:
		return dim, dim.isEqual(scope, otherT.expr())
	case *valueDimension:
		return dim, dim.isEqual(scope, otherT.expr())
	case *genericDimension:
		if otherT.x == nil {
			return dim, true
		}
		return dim.reconcileWith(scope, otherT.x)
	default:
		return dim, scope.err().Appendf(dim.ext.Src, cannotInferAxisLength)
	}
}

func (dim *exprDimension) String() string {
	return dim.expr().String()
}

func (dim *exprDimension) dimExpr() *ir.AxisExpr {
	if dim.ext.X != nil {
		return dim.ext
	}
	dim.ext.X = dim.x.buildExpr()
	return dim.ext
}

func (dim *exprDimension) expr() ir.AxisLength {
	return dim.dimExpr()
}

// genericDimension is a dimension specified with _
type genericDimension struct {
	ext *ir.AxisEllipsis
	x   *exprDimension
}

func (dim *genericDimension) resolve(scope scoper) bool {
	if dim.x == nil {
		return true
	}
	return dim.x.resolve(scope)
}

func (dim *genericDimension) reconcileWithExpr(scope scoper, other *exprDimension) (arrayAxes, bool) {
	if dim.x == nil {
		return &genericDimension{ext: dim.ext, x: other}, true
	}
	return dim, dim.x.isEqual(scope, other.expr())
}

func (dim *genericDimension) reconcileWith(scope scoper, other arrayAxes) (arrayAxes, bool) {
	switch otherT := other.(type) {
	case *genericDimension:
		if dim.x != nil && otherT.x == nil {
			return dim, scope.err().Appendf(dim.ext.Src, "cannot reconcile a resolved axis length with an unresolved axis length")
		}
		if dim.x == nil && otherT.x == nil {
			return dim, true
		}
		return dim.reconcileWithExpr(scope, otherT.x)
	case *exprDimension:
		return dim.reconcileWithExpr(scope, otherT)
	case *valueDimension:
		exprDim := newExprDimension(otherT.exprNode())
		return dim.reconcileWith(scope, exprDim)
	default:
		scope.err().Appendf(dim.ext.Src, "dimension type %T not supported", otherT)
		return dim, false
	}
}

func (dim *genericDimension) String() string {
	return dim.expr().String()
}

func (dim *genericDimension) exprNode() exprNode {
	return dim.x.exprNode()
}

func (dim *genericDimension) expr() ir.AxisLength {
	if dim.x == nil {
		return dim.ext
	}
	dim.ext.X = dim.x.dimExpr()
	return dim.ext
}
