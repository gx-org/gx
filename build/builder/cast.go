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

	"github.com/gx-org/gx/build/ir"
)

type typeCast struct {
	x   ast.Expr
	typ typeNode
}

var _ toTypeCaster = (toTypeCaster)(nil)

func processCast(owner owner, expr ast.Expr) (*typeCast, bool) {
	typ, ok := processTypeExpr(owner, expr)
	return &typeCast{typ: typ, x: expr}, ok
}

func (n *typeCast) expr() ast.Expr {
	return n.x
}

func (n *typeCast) source() ast.Node {
	return n.expr()
}

func (n *typeCast) toTypeCast() *typeCast {
	return n
}

// resolveTypes recursively calls resolveTypes in the underlying subtree of nodes.
func (n *typeCast) resolveType(scope scoper) (typeNode, bool) {
	var ok bool
	n.typ, ok = resolveType(scope, n, n.typ)
	return n.typ, ok
}

func (n *typeCast) buildExpr() ir.Expr {
	return nil
}

func (n *typeCast) String() string {
	return fmt.Sprintf("cast(%s)", n.typ)
}

type castExpr struct {
	ext ir.CastExpr
	typ *typeCast
	x   exprNode
}

func processCastExpr(owner owner, src *ast.CallExpr, typ *typeCast) (*castExpr, bool) {
	n := &castExpr{
		ext: ir.CastExpr{Src: src},
		typ: typ,
	}
	if len(src.Args) == 0 {
		owner.err().Appendf(src, "missing argument in conversion to %s", typ.String())
		return n, false
	}
	if len(src.Args) > 1 {
		owner.err().Appendf(src, "too many arguments in conversion to %s", typ.String())
		return n, false
	}
	var ok bool
	n.x, ok = processExpr(owner, src.Args[0])
	return n, ok
}

func (n *castExpr) source() ast.Node {
	return n.expr()
}

func (n *castExpr) expr() ast.Expr {
	return n.ext.Src
}

// resolveTypes recursively calls resolveTypes in the underlying subtree of nodes.
func (n *castExpr) resolveType(scope scoper) (typeNode, bool) {
	targetType, typeOk := n.typ.resolveType(scope)
	origType, exprOk := n.x.resolveType(scope)
	if origType.kind() == ir.NumberKind {
		n.x, origType, exprOk = buildNumberNode(scope, n.x, targetType.buildType())
	}
	if !typeOk || !exprOk {
		return n.typ.typ, false
	}
	typ, ok := convertTo(scope, n, origType, targetType)
	n.typ = &typeCast{
		x:   n.typ.x,
		typ: typ,
	}
	return typ, ok
}

func (n *castExpr) buildExpr() ir.Expr {
	n.ext.Typ = n.typ.typ.buildType()
	n.ext.X = n.x.buildExpr()
	return &n.ext
}

func (n *castExpr) String() string {
	return fmt.Sprintf("%s(%s)", n.typ, n.x)
}
