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

type ifStmt struct {
	ext      ir.IfStmt
	cond     exprNode
	init     stmtNode
	body     *blockStmt
	elseStmt stmtNode
}

var _ stmtNode = (*ifStmt)(nil)

func processIfStmt(owner owner, fn *funcDecl, stmt *ast.IfStmt) (*ifStmt, bool) {
	n := &ifStmt{
		ext: ir.IfStmt{
			Src: stmt,
		},
	}
	initOk := true
	if stmt.Init != nil {
		n.init, initOk = processStmt(owner, fn, stmt.Init)
	}
	var condOk bool
	n.cond, condOk = processExpr(owner, stmt.Cond)
	var bodyOk bool
	n.body, bodyOk = processBlockStmt(owner, fn, stmt.Body)
	elseOk := true
	if stmt.Else != nil {
		n.elseStmt, elseOk = processStmt(owner, fn, stmt.Else)
	}
	return n, initOk && condOk && bodyOk && elseOk
}

func (n *ifStmt) checkConditionType(scope scoper, typ typeNode) bool {
	ok, err := typ.buildType().Equal(scope.evalFetcher(), boolType.buildType())
	if err != nil {
		scope.err().Appendf(n.cond.source(), "cannot evaluatate expression type: %v", err)
		return false
	}
	if !ok {
		scope.err().Appendf(n.cond.source(), "non-boolean condition in if statement")
		return false
	}
	return true
}

func (n *ifStmt) resolveType(scope *scopeBlock) bool {
	block := scope.scopeBlock()
	initOk := true
	if n.init != nil {
		initOk = n.init.resolveType(block)
	}
	condTyp, condOk := n.cond.resolveType(block)
	if condOk {
		condOk = n.checkConditionType(block, condTyp)
	}
	bodyOk := n.body.resolveType(block)
	elseOk := true
	if n.elseStmt != nil {
		elseOk = n.elseStmt.resolveType(block)
	}
	return initOk && condOk && bodyOk && elseOk
}

func (n *ifStmt) buildStmt() ir.Stmt {
	if n.init != nil {
		n.ext.Init = n.init.buildStmt()
	}
	n.ext.Cond = n.cond.buildExpr()
	n.ext.Body = n.body.buildBlockStmt()
	if n.elseStmt != nil {
		n.ext.Else = n.elseStmt.buildStmt()
	}
	return &n.ext
}

// source is the position of the if statement in the code.
func (n *ifStmt) source() ast.Node {
	return n.ext.Source()
}
