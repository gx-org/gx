// Copyright 2025 Google LLC
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
	"reflect"

	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/build/ir/irkind"
)

type varargsType struct {
	src *ast.Ellipsis

	elt typeExprNode
}

func processVarArgsType(pscope typeProcScope, src *ast.Ellipsis) (typeExprNode, bool) {
	elt, ok := processTypeExpr(pscope, src.Elt)
	grpScope, _ := pscope.(*fieldGroupProcScope)
	var fnParamScope *funcParamScope
	if grpScope != nil {
		fnParamScope = grpScope.fieldListProcScope.typeProcScope.(*funcParamScope)
	}
	n := &varargsType{
		src: src,
		elt: elt,
	}
	if grpScope == nil || fnParamScope == nil {
		ok = pscope.Err().Appendf(src, "unexpected ..., expected type")
	} else {
		if !grpScope.isLast() {
			ok = pscope.Err().Appendf(src, "can only use ... with final parameter")
		}
		fnParamScope.ftype.varargs = n
	}
	return n, ok
}

func (n *varargsType) source() ast.Node {
	return n.src
}

func (n *varargsType) buildTypeExpr(rscope resolveScope) (*ir.TypeValExpr, bool) {
	elt, ok := n.elt.buildTypeExpr(rscope)
	typ := &ir.VarArgsType{
		Src: n.src,
		Typ: &ir.SliceType{
			BaseType: ir.BaseType[ast.Expr]{Src: n.src},
			DType:    elt,
			Rank:     1,
		},
	}
	return ir.TypeExpr(&ir.VarArgsExpr{
		Elt: typ,
	}, typ), ok
}

func (n *varargsType) String() string {
	return "..." + n.elt.String()
}

type unpackExpr struct {
	x exprNode
}

func processExprWithEllipsis(pscope procScope, expr ast.Expr) (exprNode, bool) {
	x, ok := processExpr(pscope, expr)
	return &unpackExpr{x: x}, ok
}

func (n *unpackExpr) source() ast.Node {
	return n.x.source()
}

func (n *unpackExpr) buildExpr(rscope resolveScope) (ir.Expr, bool) {
	var ok bool
	ext := &ir.UnpackExpr{}
	ext.X, ok = n.x.buildExpr(rscope)
	if !ok {
		return ext, false
	}
	xType := ext.X.Type()
	if xType.Kind() != irkind.Slice {
		return ext, rscope.Err().Appendf(n.source(), "cannot unpack type %s", xType.ReferString(rscope.fileScope().irFile()))
	}
	slType, ok := xType.(ir.SlicerType)
	if !ok {
		return ext, rscope.Err().AppendInternalf(n.source(), "cannot convert %T to %s", xType, reflect.TypeFor[ir.SlicerType]().String())
	}
	ext.EltTyp, ok = slType.ElementType()
	if !ok {
		return ext, rscope.Err().AppendInternalf(n.source(), "cannot access element type of %s (implementation: %T)", slType.ReferString(rscope.fileScope().irFile()), slType)
	}
	return ext, ok
}

func (n *unpackExpr) String() string {
	return n.x.String() + "..."
}
