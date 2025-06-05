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

func processSliceType(pscope procScope, src *ast.ArrayType) (*sliceType, bool) {
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
	expr := &ir.SliceType{
		BaseType: baseType(n.src),
		Rank:     n.rank,
		DType:    dtypeExpr,
	}
	return &ir.TypeValExpr{X: expr, Typ: expr}, ok
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
		return &ir.SliceLitExpr{Src: n.src, Typ: ir.InvalidType()}, rscope.err().AppendInternalf(n.src, "%T is not a slice type", under)
	}
	return n.lit.buildExpr(newSliceLitScope(rscope, sliceType))
}

func (n *sliceLitExpr) String() string {
	return n.typ.String() + n.lit.String()
}
