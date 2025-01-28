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

// sliceType defines an slice (which only stays on host).
type sliceType struct {
	ext   *ir.SliceType
	dtype typeNode
	typ   typeNode
}

var (
	_ concreteTypeNode = (*sliceType)(nil)
	_ indexableType    = (*sliceType)(nil)
)

func importSliceType(scope scoper, typ *ir.SliceType) (*sliceType, bool) {
	tp := &sliceType{
		ext: typ,
	}
	var dtypeOk bool
	tp.dtype, dtypeOk = toTypeNode(scope, typ.DType)
	tp.typ = tp
	return tp, dtypeOk
}

func processSliceType(owner owner, typ *ast.ArrayType) (*sliceType, bool) {
	rank := 0
	var elt ast.Node = typ
	for {

		if eltT, ok := elt.(*ast.ArrayType); ok && eltT.Len == nil {
			// The subtype is a slice: increase the rank and continue.
			rank++
			elt = eltT.Elt
			continue
		}
		// Process the data type of the slice.
		dtype, ok := processTypeExpr(owner, elt)
		return &sliceType{
			ext: &ir.SliceType{
				Src:  typ,
				Rank: rank,
			},
			dtype: dtype,
		}, ok
	}
}

func (n *sliceType) subsliceType() (*sliceType, bool) {
	return &sliceType{
		ext: &ir.SliceType{
			Rank: n.ext.Rank - 1,
		},
		dtype: n.dtype,
	}, n.dtype != nil
}

func (n *sliceType) source() ast.Node {
	return n.ext.Source()
}

func (n *sliceType) irType() ir.Type {
	return n.ext
}

func (n *sliceType) kind() ir.Kind {
	return ir.SliceKind
}

func (n *sliceType) convertibleTo(scope scoper, typ typeNode) (bool, []*ir.ValueRef, error) {
	return false, nil, nil
}

func (n *sliceType) isGeneric() bool {
	return false
}

func (n *sliceType) resolveConcreteType(scope scoper) (typeNode, bool) {
	if n.typ != nil {
		return typeNodeOk(n.typ)
	}
	n.typ = n
	_, ok := resolveType(scope, n, n.dtype)
	if !ok {
		return n, false
	}
	n.ext.DType = n.dtype.irType()
	return n, true
}

func (n *sliceType) elementType() (typeNode, error) {
	return n.dtype, nil
}

func (n *sliceType) String() string {
	return n.ext.String()

}

// sliceExpr defines a slice with its content.
type sliceExpr struct {
	ext      ir.SliceExpr
	typ      *sliceType
	elements []exprNode
}

func toSliceExpr(scope scoper, expr ast.Expr, dtype typeNode, elements []exprNode) *sliceExpr {
	return &sliceExpr{
		ext: ir.SliceExpr{Src: expr},
		typ: &sliceType{
			ext:   &ir.SliceType{Rank: 1},
			dtype: dtype,
		},
		elements: elements,
	}
}

func (n *sliceType) toExprNode(owner owner, src *ast.CompositeLit) (exprNode, bool) {
	expr := &sliceExpr{
		ext: ir.SliceExpr{Src: src},
		typ: n,
	}
	ok := true
	for _, eltSrc := range src.Elts {
		eltExpr, eltOk := processExpr(owner, eltSrc)
		ok = eltOk && ok
		expr.elements = append(expr.elements, eltExpr)
	}
	return expr, ok
}

func (n *sliceExpr) resolveType(scope scoper) (typeNode, bool) {
	if n.typ.typ != nil {
		return typeNodeOk(n.typ.typ)
	}
	if _, ok := resolveType(scope, n, n.typ); !ok {
		return n.typ, false
	}
	ok := true
	n.ext.Vals = make([]ir.Expr, len(n.elements))
	for i, elt := range n.elements {
		eltType, eltOk := elt.resolveType(scope)
		if !eltOk {
			ok = false
			continue
		}
		if ir.IsNumber(eltType.kind()) {
			eltType = n.typ.dtype
			elt, eltType, eltOk = castNumber(scope, elt, eltType.irType())
			if !eltOk {
				ok = false
				continue
			}
			n.elements[i] = elt
		}
		_, eltOk, err := assignableTo(scope, n, eltType, n.typ.dtype)
		if err != nil {
			ok = scope.err().AppendAt(elt.source(), err)
		}
		if !eltOk {
			ok = scope.err().Appendf(elt.source(), "cannot use %s as %s in slice", eltType.String(), n.typ.dtype.String())
		}
		n.ext.Vals[i] = elt.buildExpr()
	}
	n.ext.Typ = n.typ.irType()
	return n.typ, ok
}

func (n *sliceExpr) source() ast.Node {
	return n.expr()
}

func (n *sliceExpr) expr() ast.Expr {
	return n.ext.Expr()
}

func (n *sliceExpr) buildExpr() ir.Expr {
	n.ext.Typ = n.typ.irType()
	return &n.ext
}

func (n *sliceExpr) String() string {
	return n.ext.String()
}
