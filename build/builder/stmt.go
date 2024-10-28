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

type blockStmt struct {
	ext   ir.BlockStmt
	stmts []stmtNode
}

func processBlockStmt(owner owner, fn *funcDecl, block *ast.BlockStmt) (*blockStmt, bool) {
	n := &blockStmt{
		ext: ir.BlockStmt{
			Src: block,
		},
	}
	ok := true
	for _, stmt := range block.List {
		node, nOk := processStmt(owner, fn, stmt)
		if node != nil {
			n.stmts = append(n.stmts, node)
		}
		ok = ok && nOk
	}
	return n, ok
}

func (n *blockStmt) resolveType(scope *scopeBlock) bool {
	for _, stmt := range n.stmts {
		if !stmt.resolveType(scope) {
			// resolveType is expected to report an explicit error in the event of failure, so if none is
			// found here, return a generic error for completeness.
			if scope.err().Errors().Empty() {
				scope.err().Appendf(stmt.source(), "failed to typecheck statement; probably a compiler bug")
			}
			return false
		}
	}
	return true
}

func (n *blockStmt) buildBlockStmt() *ir.BlockStmt {
	n.ext.List = make([]ir.Stmt, len(n.stmts))
	for i, node := range n.stmts {
		n.ext.List[i] = node.buildStmt()
	}
	return &n.ext
}

func (n *blockStmt) buildStmt() ir.Stmt {
	return n.buildBlockStmt()
}

func (n *blockStmt) source() ast.Node {
	return n.ext.Source()
}

type exprStmt struct {
	ext ir.ExprStmt
	x   exprNode
}

func processExprStmt(owner owner, fn *funcDecl, stmt *ast.ExprStmt) (*exprStmt, bool) {
	n := &exprStmt{
		ext: ir.ExprStmt{
			Src: stmt,
		},
	}
	var ok bool
	n.x, ok = processExpr(owner, stmt.X)
	return n, ok
}

func (n *exprStmt) resolveType(scope *scopeBlock) bool {
	typ, ok := n.x.resolveType(scope)
	if !ok || !ir.IsValid(typ.buildType()) {
		return false
	}
	if typ.kind() != ir.VoidKind {
		scope.err().Appendf(n.ext.Src, "cannot use an expression returning a value as a statement")
		return false
	}
	return true
}

func (n *exprStmt) buildStmt() ir.Stmt {
	n.ext.X = n.x.buildExpr()
	return &n.ext
}

func (n *exprStmt) source() ast.Node {
	return n.ext.Source()
}

func processStmt(owner owner, fn *funcDecl, stmt ast.Stmt) (node stmtNode, ok bool) {
	switch s := stmt.(type) {
	case *ast.AssignStmt:
		node, ok = processAssign(owner, s)
	case *ast.RangeStmt:
		node, ok = processRangeStmt(owner, fn, s)
	case *ast.ReturnStmt:
		node, ok = processReturnStmt(owner, fn, s)
	case *ast.IfStmt:
		node, ok = processIfStmt(owner, fn, s)
	case *ast.BlockStmt:
		node, ok = processBlockStmt(owner, fn, s)
	case *ast.ExprStmt:
		node, ok = processExprStmt(owner, fn, s)
	default:
		owner.err().Appendf(stmt, "statement type not supported: %T", stmt)
		ok = false
	}
	return
}
