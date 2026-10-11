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
	"go/token"

	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/build/ir/irkind"
	"github.com/gx-org/gx/src/gx/ast"
)

type unaryExpr struct {
	src *ast.UnaryExpr
	x   exprNode
}

var _ exprNode = (*unaryExpr)(nil)

func processUnaryExpr(pscope procScope, expr *ast.UnaryExpr) (exprNode, bool) {
	x, ok := processExpr(pscope, expr.X)
	n := &unaryExpr{src: expr, x: x}
	switch expr.Op {
	case token.SUB, token.ADD, token.NOT:
		return n, ok
	}
	return n, pscope.Err().Appendf(expr, "unary operator %s not supported", expr.Op.String())
}

func (n *unaryExpr) source() ast.Node {
	return n.src
}

func (n *unaryExpr) checkSub(scope resolveScope, x ir.Expr) bool {
	xType := x.Type()
	if ir.IsInvalidType(xType) || xType.Kind() == irkind.Unknown {
		return true
	}
	if array, isArray := xType.(ir.ArrayType); isArray {
		xType = array.DataType()
	}
	if !ir.IsSigned(xType) {
		return scope.Err().Appendf(n.src, "operator %s not defined on %s (expect a signed type)", n.src.Op.String(), xType.ReferString(scope.fileScope().irFile()))
	}
	return true
}

func (n *unaryExpr) buildExpr(scope resolveScope) (ir.Expr, bool) {
	x, ok := buildExpr(scope, n.x)
	ext := &ir.UnaryExpr{Src: n.src, X: x}
	if !ok {
		return ext, false
	}
	if n.src.Op == token.SUB {
		ok = n.checkSub(scope, x)
	}
	return ext, ok
}

func (n *unaryExpr) String() string {
	return n.src.Op.String() + n.x.String()
}
