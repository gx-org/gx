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

type unaryExpr struct {
	ext ir.UnaryExpr
	x   exprNode

	val ir.Atomic
}

var (
	_ exprNumber = (*unaryExpr)(nil)
	_ exprScalar = (*unaryExpr)(nil)
)

func processUnaryExpr(owner owner, expr *ast.UnaryExpr) (exprNode, bool) {
	x, ok := processExpr(owner, expr.X)
	return &unaryExpr{
		ext: ir.UnaryExpr{
			Src: expr,
		},
		x: x,
	}, ok
}

func (n *unaryExpr) buildExpr() ir.Expr {
	if n.val != nil {
		return n.val
	}
	n.ext.X = n.x.buildExpr()
	return &n.ext
}

func (n *unaryExpr) castTo(eval evaluator) (exprScalar, []*ir.ValueRef, bool) {
	xEval, unknowns, ok := castExprTo(eval, n.x)
	if !ok {
		return nil, nil, false
	}
	n.x = xEval
	n.ext.X = n.x.buildExpr()
	n.val = eval.cast(&n.ext)
	return n, unknowns, ok
}

func (n *unaryExpr) scalar() ir.Atomic {
	return n.val
}

func (n *unaryExpr) source() ast.Node {
	return n.expr()
}

func (n *unaryExpr) expr() ast.Expr {
	return n.ext.Src
}

func (n *unaryExpr) resolveType(scope scoper) (typeNode, bool) {
	return n.x.resolveType(scope)
}

func (n *unaryExpr) String() string {
	return n.ext.Src.Op.String() + n.x.String()
}
