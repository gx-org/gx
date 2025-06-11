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

func processIfStmt(pscope procScope, stmt *ast.IfStmt) (*ifStmt, bool) {
	n := &ifStmt{
		ext: ir.IfStmt{
			Src: stmt,
		},
	}
	initOk := true
	if stmt.Init != nil {
		n.init, initOk = processStmt(pscope, stmt.Init)
	}
	var condOk bool
	n.cond, condOk = processExpr(pscope, stmt.Cond)
	var bodyOk bool
	n.body, bodyOk = processBlockStmt(pscope, stmt.Body)
	elseOk := true
	if stmt.Else != nil {
		n.elseStmt, elseOk = processStmt(pscope, stmt.Else)
	}
	return n, initOk && condOk && bodyOk && elseOk
}

func (n *ifStmt) checkConditionType(scope resolveScope, typ ir.Type) bool {
	compEval, compEvalOk := scope.compEval()
	if !compEvalOk {
		return false
	}
	ok, err := typ.Equal(compEval, ir.BoolType())
	if err != nil {
		return scope.err().Appendf(n.cond.source(), "cannot evaluatate expression type: %v", err)
	}
	if !ok {
		return scope.err().Appendf(n.cond.source(), "non-boolean condition in if statement")
	}
	return true
}

func (n *ifStmt) buildStmt(rscope iFuncResolveScope) (ir.Stmt, bool) {
	bScope, ok := newBlockScope(rscope)
	if !ok {
		return nil, false
	}
	var init ir.Stmt
	initOk := true
	if n.init != nil {
		init, initOk = n.init.buildStmt(bScope)
	}
	cond, condOk := buildAExpr(bScope, n.cond)
	if condOk {
		condOk = n.checkConditionType(bScope, cond.Type())
	}
	body, bodyOk := n.body.buildBlockStmt(bScope)
	var els ir.Stmt
	elseOk := true
	if n.elseStmt != nil {
		els, elseOk = n.elseStmt.buildStmt(bScope)
	}
	return &ir.IfStmt{
		Init: init,
		Cond: cond,
		Body: body,
		Else: els,
	}, initOk && condOk && bodyOk && elseOk
}
