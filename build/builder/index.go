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

type (
	indexableType interface {
		elementType() (typeNode, error)
	}

	// indexExpr references an element (or subarray) of an array.
	indexExpr struct {
		ext ir.IndexExpr
		src *ast.IndexExpr

		x     exprNode
		index exprNode

		typ typeNode
	}
)

var _ exprNode = (*indexExpr)(nil)

func processIndexExpr(owner owner, expr *ast.IndexExpr) (*indexExpr, bool) {
	n := &indexExpr{
		src: expr,
		ext: ir.IndexExpr{Src: expr},
	}
	var ok bool
	n.x, ok = processExpr(owner, expr.X)
	if !ok {
		return nil, false
	}
	n.index, ok = processExpr(owner, expr.Index)
	return n, ok
}

func (n *indexExpr) source() ast.Node {
	return n.expr()
}

func (n *indexExpr) expr() ast.Expr {
	return n.src
}

func (n *indexExpr) String() string {
	return toString(n.src)
}

func (n *indexExpr) buildExpr() ir.Expr {
	n.ext.X = n.x.buildExpr()
	n.ext.Index = n.index.buildExpr()
	n.ext.Typ = n.typ.buildType()
	return &n.ext
}

func (n *indexExpr) checkIndexType(scope scoper, typ typeNode) (ok bool) {
	if typ.kind() == ir.NumberKind {
		// Coerce index type to a concrete integer type.
		n.index, _, ok = buildNumberNode(scope, n.index, ir.DefaultIntType)
		return
	}
	if !ir.IsIndexKind(typ.kind()) {
		scope.err().Appendf(n.source(), "invalid argument: index %s must be integer", typ.String())
		return
	}
	return true
}

func (n *indexExpr) resolveType(scope scoper) (typeNode, bool) {
	if n.typ != nil {
		return typeNodeOk(n.typ)
	}

	indexType, indexOk := n.index.resolveType(scope)
	if indexOk {
		indexOk = n.checkIndexType(scope, indexType)
	}
	exprType, ok := n.x.resolveType(scope)
	if !ok {
		n.typ, _ = invalidType()
		return n.typ, false
	}
	idxType, ok := exprType.(indexableType)
	if !ok {
		scope.err().Appendf(n.source(), "indexing unsupported on %T", exprType)
		n.typ, _ = invalidType()
		return n.typ, false
	}
	var err error
	n.typ, err = idxType.elementType()
	if err != nil {
		scope.err().AppendAt(n.source(), err)
	}
	return n.typ, indexOk
}
