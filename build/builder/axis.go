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
)

// axisLengthNode defines the dimension of an array.
type axisLengthNode interface {
	build(*defineLocalScope) (ir.AxisLengths, bool)
	String() string
}

func processAxisLengthExpr(axScope axLenPScope, array *ast.ArrayType) (axis axisLengthNode, ok bool) {
	if array.Len == nil {
		return nil, axScope.err().Appendf(array, "array of slices is not supported. Please specify an array axis length using '[length]', '[_]', or '[...]' if the size is unknown, to specify a dense array")
	}
	switch lenT := array.Len.(type) {
	case *ast.Ellipsis:
		return nil, true
	case *ast.Ident:
		if lenT.Name == ir.DefineAxisGroup {
			return nil, true
		}
		if lenT.Name == ir.DefineAxisLength {
			return &genericAxisLength{src: lenT}, true
		}
		return axScope.processIdent(lenT)
	case ast.Expr:
		return processExprAxisLength(axScope, lenT)
	default:
		return nil, axScope.err().Appendf(array, "array dimension type %T not supported", lenT)
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
		Src: dim.src,
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

func processExprAxisLength(pscope axLenPScope, src ast.Expr) (*exprAxisLength, bool) {
	x, ok := processExpr(pscope, src)
	return &exprAxisLength{src: src, x: x}, ok
}

func (dim *exprAxisLength) build(rscope *defineLocalScope) (ir.AxisLengths, bool) {
	ext := &ir.AxisExpr{Src: dim.src}
	var xOk bool
	ext.X, xOk = buildAExpr(rscope, dim.x)
	if !xOk {
		return ext, false
	}
	if ir.IsNumber(ext.X.Type().Kind()) {
		ext.X, xOk = castNumber(rscope, ext.X, ir.IntLenType())
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
	x    *exprAxisLength
}

var _ axisLengthNode = (*defineAxisLength)(nil)

func (dim *defineAxisLength) build(rscope *defineLocalScope) (ir.AxisLengths, bool) {
	rscope.defineAxis(
		&ast.Ident{
			NamePos: dim.src.NamePos,
			Name:    dim.name,
		},
		ir.IntLenType())
	return dim.x.build(rscope)
}

func (dim *defineAxisLength) String() string {
	return dim.src.Name
}

// genericAxisLength is a dimension specified with _ and that needs to be inferred.
type genericAxisLength struct {
	src *ast.Ident
}

var _ axisLengthNode = (*genericAxisLength)(nil)

func (dim *genericAxisLength) String() string {
	return dim.src.Name
}

func (dim *genericAxisLength) build(rscope *defineLocalScope) (ir.AxisLengths, bool) {
	return &ir.AxisInfer{Src: dim.src}, true
}

// axisGroup is a group of axis with a name.
type axisGroup struct {
	src  *ast.Ident
	name string
}

var _ axisLengthNode = (*axisGroup)(nil)

func (dim *axisGroup) String() string {
	return dim.src.Name
}

func (dim *axisGroup) build(rscope *defineLocalScope) (ir.AxisLengths, bool) {
	grp := &ir.AxisGroup{Src: dim.src, Name: dim.name}
	rscope.defineAxis(
		&ast.Ident{
			NamePos: dim.src.NamePos,
			Name:    dim.name,
		},
		ir.IntLenSliceType(),
	)
	return grp, true
}
