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
	src *ast.UnaryExpr
	x   exprNode
}

var _ exprNode = (*unaryExpr)(nil)

func processUnaryExpr(pscope procScope, expr *ast.UnaryExpr) (exprNode, bool) {
	x, ok := processExpr(pscope, expr.X)
	return &unaryExpr{src: expr, x: x}, ok
}

func (n *unaryExpr) source() ast.Node {
	return n.src
}

func (n *unaryExpr) buildExpr(scope resolveScope) (ir.Expr, bool) {
	x, ok := buildAExpr(scope, n.x)
	return &ir.UnaryExpr{Src: n.src, X: x}, ok
}

func (n *unaryExpr) String() string {
	return n.src.Op.String() + n.x.String()
}
