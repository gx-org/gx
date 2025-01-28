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

type parenExpr struct {
	ext ir.ParenExpr

	typ typeNode
	x   exprNode

	val ir.StaticExpr
}

var _ exprScalar = (*parenExpr)(nil)

func processParenExpr(owner owner, expr *ast.ParenExpr) (exprNode, bool) {
	x, ok := processExpr(owner, expr.X)
	return &parenExpr{
		ext: ir.ParenExpr{
			Src: expr,
		},
		x: x,
	}, ok
}

func (n *parenExpr) toTypeCast() *typeCast {
	return toTypeCast(n.x)
}

func (n *parenExpr) buildExpr() ir.Expr {
	if n.val != nil {
		return n.val
	}
	return &n.ext
}

func (n *parenExpr) scalar() ir.StaticExpr {
	return n.val
}

func (n *parenExpr) source() ast.Node {
	return n.expr()
}

func (n *parenExpr) expr() ast.Expr {
	return n.ext.Src
}

func (n *parenExpr) resolveType(scope scoper) (typeNode, bool) {
	if n.typ != nil {
		return typeNodeOk(n.typ)
	}
	var ok bool
	n.typ, ok = n.x.resolveType(scope)
	n.ext.X = n.x.buildExpr()
	if !ok {
		return n.typ, ok
	}
	return typeNodeOk(n.typ)
}

func (n *parenExpr) String() string {
	return n.typ.String()
}
