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

type typeAssertExpr struct {
	src *ast.TypeAssertExpr
	x   exprNode
	typ typeExprNode
}

var _ exprNode = (*typeAssertExpr)(nil)

func processTypeAssertExpr(pscope procScope, src *ast.TypeAssertExpr) (exprNode, bool) {
	x, xOk := processExpr(pscope, src.X)
	typScope := defaultTypeProcScope(pscope)
	typ, typOk := processTypeExpr(typScope, src.Type)
	return &typeAssertExpr{
		src: src,
		x:   x,
		typ: typ,
	}, xOk && typOk
}

func (n *typeAssertExpr) source() ast.Node {
	return n.src
}

// resolveType recursively calls resolveTypes in the underlying subtree of nodes.
func (n *typeAssertExpr) buildExpr(rscope resolveScope) (ir.Expr, bool) {
	x, xOk := n.x.buildExpr(rscope)
	typeExpr, toTypOk := n.typ.buildTypeExpr(rscope)
	return &ir.TypeAssertExpr{
		Src: n.src,
		X:   x,
		Typ: typeExpr.Typ,
	}, xOk && toTypOk
}

func (n *typeAssertExpr) String() string {
	return n.typ.String()
}
