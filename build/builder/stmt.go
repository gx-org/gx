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
	src   *ast.BlockStmt
	stmts []stmtNode
}

var _ stmtNode = (*blockStmt)(nil)

func processBlockStmt(pscope procScope, src *ast.BlockStmt) (*blockStmt, bool) {
	n := &blockStmt{src: src}
	ok := true
	for _, stmt := range src.List {
		node, nOk := processStmt(pscope, stmt)
		if node != nil {
			n.stmts = append(n.stmts, node)
		}
		ok = ok && nOk
	}
	return n, ok
}

func (n *blockStmt) buildStmt(scope funResolveScope) (ir.Stmt, bool) {
	return n.buildBlockStmt(scope)
}

func (n *blockStmt) buildBlockStmt(scope funResolveScope) (*ir.BlockStmt, bool) {
	block := &ir.BlockStmt{
		Src:  n.src,
		List: make([]ir.Stmt, len(n.stmts)),
	}
	ok := true
	for i, node := range n.stmts {
		var stmtOk bool
		block.List[i], stmtOk = node.buildStmt(scope)
		ok = ok && stmtOk
	}
	if !ok && scope.err().Errors().Empty() {
		// an error occurred but no explicit error has been added to the list of error.
		// report an error now to help with debugging.
		if scope.err().Errors().Empty() {
			scope.err().AppendInternalf(n.src, "failed to build statement but no error reported to the user")
		}
	}
	return block, ok
}

type exprStmt struct {
	src *ast.ExprStmt
	x   exprNode
}

var _ stmtNode = (*exprStmt)(nil)

func processExprStmt(pscope procScope, src *ast.ExprStmt) (*exprStmt, bool) {
	n := &exprStmt{src: src}
	var ok bool
	n.x, ok = processExpr(pscope, src.X)
	return n, ok
}

func (n *exprStmt) buildStmt(scope funResolveScope) (ir.Stmt, bool) {
	x, ok := n.x.buildExpr(scope)
	if ok && x.Type().Kind() != ir.VoidKind {
		scope.err().Appendf(n.src, "cannot use an expression returning a value as a statement")
	}
	return &ir.ExprStmt{Src: n.src, X: x}, ok
}

func processStmt(pscope procScope, stmt ast.Stmt) (node stmtNode, ok bool) {
	switch s := stmt.(type) {
	case *ast.AssignStmt:
		node, ok = processAssign(pscope, s)
	case *ast.RangeStmt:
		node, ok = processRangeStmt(pscope, s)
	case *ast.ReturnStmt:
		node, ok = processReturnStmt(pscope, s)
	case *ast.IfStmt:
		node, ok = processIfStmt(pscope, s)
	case *ast.BlockStmt:
		node, ok = processBlockStmt(pscope, s)
	case *ast.ExprStmt:
		node, ok = processExprStmt(pscope, s)
	default:
		pscope.err().Appendf(stmt, "statement type not supported: %T", stmt)
		ok = false
	}
	return
}
