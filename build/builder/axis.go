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
	"math/big"

	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/build/ir/irkind"
)

// axisLengthNode defines the dimension of an array.
type axisLengthNode interface {
	build(*defineLocalScope) (ir.AxisLengths, bool)
	String() string
}

func processAxisLengthExpr(axScope procAxLenScope, array *ast.ArrayType) (axis axisLengthNode, ok bool) {
	if array.Len == nil {
		return nil, axScope.Err().Appendf(array, "array of slices is not supported. Please specify an array axis length using '[length]' or '[...]'")
	}
	switch lenT := array.Len.(type) {
	case *ast.Ellipsis:
		return nil, true
	case *ast.Ident:
		if lenT.Name == ir.DefineAxisGroup {
			return nil, true
		}
		return axScope.processAxisExpr(lenT)
	case ast.Expr:
		return axScope.processAxisExpr(lenT)
	default:
		return nil, axScope.Err().Appendf(array, "array dimension type %T not supported", lenT)
	}
}

// inferredFromLiteralAxisLength computed from a literal.
type inferredFromLiteralAxisLength struct {
	src ast.Expr // Point to the literal to report mismatch errors.
	val int
}

var _ axisLengthNode = (*inferredFromLiteralAxisLength)(nil)

func (dim *inferredFromLiteralAxisLength) build(rscope *defineLocalScope) (ir.AxisLengths, bool) {
	return &ir.AxisExpr{
		X: &ir.NumberCastExpr{
			X: &ir.NumberInt{
				Src: &ast.BasicLit{ValuePos: dim.src.Pos()},
				Val: big.NewInt(int64(dim.val)),
			},
			Typ: ir.IntLenType(),
		},
	}, true
}

func (dim *inferredFromLiteralAxisLength) String() string {
	return "inferredAxisFromLiteral"
}

// exprAxisLength is a dimension specified with an expression
type exprAxisLength struct {
	src ast.Expr
	x   exprNode
}

var _ axisLengthNode = (*exprAxisLength)(nil)

func processExprAxisLength(axScope procAxLenScope, src ast.Expr) (*exprAxisLength, bool) {
	x, ok := processExpr(axScope, src)
	return &exprAxisLength{src: src, x: x}, ok
}

func (dim *exprAxisLength) build(rscope *defineLocalScope) (ir.AxisLengths, bool) {
	ext := &ir.AxisExpr{}
	var xOk bool
	ext.X, xOk = buildAExpr(rscope, dim.x)
	if !xOk {
		return ext, false
	}
	if irkind.IsNumber(ext.X.Type().Kind()) {
		ext.X, xOk = castNumber(rscope, ext.X, ir.IntLenType())
	}
	xType := ext.X.Type()
	if !ir.IsAxisLengthType(xType) {
		xOk = rscope.Err().Appendf(dim.src, "cannot use type %s as axis length: want type intlen or []intlen", xType.String())
	}
	return ext, xOk
}

const cannotInferAxisLength = "cannot infer array axis length"

func (dim *exprAxisLength) String() string {
	return dim.x.String()
}

// defineAxisLength is an axis that defines a name for an axis length
// (e.g. [_X]int32 defines X as an axis length (of type intlen)
type defineAxisLength struct {
	src  *ast.Ident
	name string
	typ  ir.Type
}

var _ axisLengthNode = (*defineAxisLength)(nil)

func (dim *defineAxisLength) source() ast.Node {
	return dim.src
}

func (dim *defineAxisLength) build(rscope *defineLocalScope) (ir.AxisLengths, bool) {
	src := *dim.src
	src.Name = dim.name
	ax := &ir.AxisStmt{
		Src: dim.src,
		Typ: dim.typ,
	}
	rscope.defineAxis(ax)
	return ax, true
}

func (dim *defineAxisLength) String() string {
	return dim.src.Name
}
