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

// arrayType defines an array type defined in the source.
type arrayType struct {
	src  *ast.ArrayType
	dtyp typeExprNode
	rnk  rankNode
}

var _ typeExprNode = (*arrayType)(nil)

func processArrayType(pscope procScope, src *ast.ArrayType) (arraySliceExprType, bool) {
	rank, dtype, ok := processDTypeRank(pscope, src)
	return &arrayType{
		src:  src,
		dtyp: dtype,
		rnk:  rank,
	}, ok
}

func (n *arrayType) source() ast.Node {
	return n.src
}

func (n *arrayType) buildTypeExpr(rscope resolveScope) (*ir.TypeValExpr, bool) {
	dtyp, dtypeOk := n.dtyp.buildTypeExpr(rscope)
	if dtypeOk && !ir.IsDataType(dtyp.Typ) {
		rscope.Err().Appendf(n.source(), "array of %s not supported", dtyp.String())
		dtypeOk = false
	}
	if !dtypeOk {
		return nil, false
	}
	// Note that rank.resolve() may return a new concrete rankNode instance if the rank was generic.
	// In that case, we return a new instance of arrayType (see below) and keep the generic arrayType
	// and rank unchanged so they can be specialized again for the next call.
	rank, rankOk := n.rnk.build(rscope)
	arrayType, ok := ir.NewArrayType(n.src, dtyp.Typ, rank), dtypeOk && rankOk
	return &ir.TypeValExpr{X: arrayType, Typ: arrayType}, ok
}

func (n *arrayType) buildExpr(rscope resolveScope) (ir.Expr, bool) {
	return n.buildTypeExpr(rscope)
}

func (n *arrayType) String() string {
	return n.rnk.String() + n.dtyp.String()
}

// arrayLitExpr defines an array with a constant value.
type arrayLitExpr struct {
	src *ast.CompositeLit
	typ *arrayType
	lit *compositeLit
}

func (n *arrayType) processLiteralExpr(pscope procScope, src *ast.CompositeLit) (exprNode, bool) {
	expr := &arrayLitExpr{src: src, typ: n}
	var ok bool
	expr.lit, ok = processUntypedCompositeLit(pscope, src)
	return expr, ok
}

// Pos returns the position of the array in the code.
func (n *arrayLitExpr) source() ast.Node {
	return n.src
}

func (n *arrayLitExpr) buildExpr(rscope resolveScope) (ir.Expr, bool) {
	typ, ok := buildArrayType(rscope, n.typ)
	if !ok {
		return &ir.SliceLitExpr{Src: n.src, Typ: ir.InvalidType()}, false
	}
	under := ir.Underlying(typ)
	arrayType, ok := under.(ir.ArrayType)
	if !ok {
		return &ir.SliceLitExpr{Src: n.src, Typ: ir.InvalidType()}, rscope.Err().AppendInternalf(n.src, "%T is not an array type", under)
	}
	return n.lit.buildExpr(toArrayLitScope(rscope, arrayType))
}

func (n *arrayLitExpr) String() string {
	return n.typ.String() + n.lit.String()
}

func checkNumExprAgainstAxis(rscope resolveScope, pos ast.Node, elts []ir.Expr, axis ir.AxisLengths) bool {
	l := big.NewInt(int64(len(elts)))
	got := &ir.AxisExpr{
		X: &ir.NumberCastExpr{
			X:   &ir.NumberInt{Src: &ast.BasicLit{ValuePos: pos.Pos()}, Val: l},
			Typ: ir.IntLenType(),
		}}
	infer, inferOk := axis.(*ir.AxisInfer)
	if inferOk {
		infer.X = got
		return true
	}
	if !axisEqual(rscope, pos, axis, got) {
		return rscope.Err().Appendf(pos, "cannot use a literal of %d elements to define an axis length %s", len(elts), axis.String())
	}
	return true
}
