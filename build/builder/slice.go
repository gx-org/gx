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
	"strings"

	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/build/ir/irkind"
)

// sliceType defines an slice (which only stays on host).
type sliceType struct {
	src   *ast.ArrayType
	rank  int
	dtype typeExprNode
}

var (
	_ exprNode     = (*sliceType)(nil)
	_ typeExprNode = (*sliceType)(nil)
)

func processSliceType(pscope typeProcScope, src *ast.ArrayType) (*sliceType, bool) {
	rank := 0
	var elt ast.Node = src
	for {

		if eltT, ok := elt.(*ast.ArrayType); ok && eltT.Len == nil {
			// The subtype is a slice: increase the rank and continue.
			rank++
			elt = eltT.Elt
			continue
		}
		// Process the data type of the slice.
		dtype, ok := processTypeExpr(pscope, elt)
		return &sliceType{
			src:   src,
			rank:  rank,
			dtype: dtype,
		}, ok
	}
}

func (n *sliceType) source() ast.Node {
	return n.src
}

func (n *sliceType) buildTypeExpr(rscope resolveScope) (*ir.TypeValExpr, bool) {
	dtypeExpr, ok := n.dtype.buildTypeExpr(rscope)
	return ir.TypeExpr(nil, &ir.SliceType{
		BaseType: ir.BaseType[ast.Expr]{Src: n.src},
		Rank:     n.rank,
		DType:    dtypeExpr,
	}), ok
}

func (n *sliceType) buildExpr(rscope resolveScope) (ir.Expr, bool) {
	return n.buildTypeExpr(rscope)
}

func (n *sliceType) String() string {
	return fmt.Sprintf("%s%s", n.dtype.String(), strings.Repeat("[]", n.rank))

}

// sliceLitExpr defines a slice with its content.
type sliceLitExpr struct {
	src *ast.CompositeLit
	typ *sliceType
	lit *compositeLit
}

var _ exprNode = (*sliceLitExpr)(nil)

func (n *sliceType) processLiteralExpr(pscope procScope, src *ast.CompositeLit) (exprNode, bool) {
	expr := &sliceLitExpr{src: src, typ: n}
	var ok bool
	expr.lit, ok = processUntypedCompositeLit(pscope, src)
	return expr, ok
}

func (n *sliceLitExpr) source() ast.Node {
	return n.src
}

func (n *sliceLitExpr) buildExpr(rscope resolveScope) (ir.Expr, bool) {
	typ, typOk := buildSliceType(rscope, n.typ)
	if !typOk {
		return &ir.SliceLitExpr{Src: n.src, Typ: ir.InvalidType()}, false
	}
	under := ir.Underlying(typ)
	sliceType, ok := under.(*ir.SliceType)
	if !ok {
		return &ir.SliceLitExpr{Src: n.src, Typ: ir.InvalidType()}, rscope.Err().AppendInternalf(n.src, "%T is not a slice type", under)
	}
	return n.lit.buildExpr(newSliceLitScope(rscope, sliceType))
}

func (n *sliceLitExpr) String() string {
	return n.typ.String() + n.lit.String()
}

type sliceExpr struct {
	src       *ast.SliceExpr
	x         exprNode
	low, high exprNode
}

func processSliceExpr(pscope procScope, src *ast.SliceExpr) (exprNode, bool) {
	if src.Slice3 {
		return nil, pscope.Err().Appendf(src, "3-index slice expression not supported")
	}
	n := &sliceExpr{src: src}
	var xOk bool
	n.x, xOk = processExpr(pscope, src.X)
	lowOk := true
	if src.Low != nil {
		n.low, lowOk = processExpr(pscope, src.Low)
	}
	highOk := true
	if src.High != nil {
		n.high, highOk = processExpr(pscope, src.High)
	}
	return n, xOk && lowOk && highOk
}

func (n *sliceExpr) source() ast.Node {
	return n.src
}

func buildBound(rscope resolveScope, b exprNode) (ir.Expr, bool) {
	if b == nil {
		return nil, true
	}
	ext, ok := b.buildExpr(rscope)
	if !ok {
		return ext, ok
	}
	if !irkind.IsNumber(ext.Type().Kind()) {
		return ext, ok
	}
	return castNilAndNumber(rscope, ext, ir.DefaultIntType)
}
func (n *sliceExpr) buildExpr(rscope resolveScope) (ir.Expr, bool) {
	ext := &ir.SliceExpr{Src: n.src}
	var xOk bool
	ext.X, xOk = n.x.buildExpr(rscope)
	if xOk && ext.X.Type().Kind() != irkind.Slice {
		from := rscope.fileScope().irFile()
		xOk = rscope.Err().Appendf(ext.Node(), "cannot slice %s (%s)", ext.X.SourceString(from), ext.X.Type().ReferString(from))
	}
	var lowOk, highOk bool
	ext.Low, lowOk = buildBound(rscope, n.low)
	ext.High, highOk = buildBound(rscope, n.high)
	return ext, xOk && lowOk && highOk
}

func (n *sliceExpr) String() string {
	low := ""
	if n.low != nil {
		low = n.low.String()
	}
	high := ""
	if n.high != nil {
		high = n.high.String()
	}
	return fmt.Sprintf("%s[%s:%s]", n.x.String(), low, high)
}
