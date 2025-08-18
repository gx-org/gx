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

package grad

import (
	"go/ast"
	"go/token"

	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/base/scope"
)

// stmtGrader computes the gradient of statements.
type stmtGrader struct {
	macro   *gradMacro
	fetcher ir.Fetcher
	parent  *stmtGrader
	scope   *scope.RWScope[ir.Type]
}

func (m *gradMacro) newStmtGrader(fetcher ir.Fetcher, parent *stmtGrader) *stmtGrader {
	var parentScope scope.Scope[ir.Type]
	if parent != nil {
		parentScope = parent.scope
	}
	return &stmtGrader{
		macro:   m,
		fetcher: fetcher,
		parent:  parent,
		scope:   scope.NewScope[ir.Type](parentScope),
	}
}

func (sg *stmtGrader) registerFieldNames(list *ir.FieldList) {
	if list == nil {
		return
	}
	for _, field := range list.Fields() {
		if field.Name == nil {
			continue
		}
		sg.scope.Define(field.Name.Name, field.Type())
	}
}

func (sg *stmtGrader) newSub() *stmtGrader {
	return sg.macro.newStmtGrader(sg.fetcher, sg)
}

func (sg *stmtGrader) gradBlock(fetcher ir.Fetcher, src *ir.BlockStmt) (*ast.BlockStmt, bool) {
	var block []ast.Stmt
	sub := sg.newSub()
	for _, stmt := range src.List {
		var ok bool
		stmts, ok := sub.gradStmt(fetcher, stmt)
		if !ok {
			return nil, false
		}
		block = append(block, stmts...)
	}
	return &ast.BlockStmt{
		List: block,
	}, true
}

func (sg *stmtGrader) gradStmt(fetcher ir.Fetcher, src ir.Stmt) ([]ast.Stmt, bool) {
	switch srcT := src.(type) {
	case *ir.ReturnStmt:
		ret, ok := sg.gradReturnStmt(fetcher, srcT)
		return []ast.Stmt{ret}, ok
	case *ir.AssignExprStmt:
		return sg.gradAssignExprStmt(fetcher, srcT)
	default:
		return nil, fetcher.Err().Appendf(src.Source(), "gradient of %T statement not supported", srcT)
	}
}

func (sg *stmtGrader) gradReturnStmt(fetcher ir.Fetcher, src *ir.ReturnStmt) (*ast.ReturnStmt, bool) {
	stmt := &ast.ReturnStmt{Results: make([]ast.Expr, len(src.Results))}
	ge := sg.newExprGrader(false)
	for i, expr := range src.Results {
		res, ok := ge.gradExpr(expr)
		if !ok {
			return nil, false
		}
		if res != nil {
			// The expression depends on arg: nothing left to do.
			stmt.Results[i] = res.expr
			continue
		}
		// The expression does not depend on arg: replace it with a zero value.
		res = zeroValueOf(expr.Source())
		stmt.Results[i] = res.expr
	}
	return stmt, true
}

func (sg *stmtGrader) define(name string, typ ir.Type) bool {
	_, found := sg.scope.Find(name)
	sg.scope.Define(name, typ)
	return found
}

func (sg *stmtGrader) gradAssignExprStmt(fetcher ir.Fetcher, src *ir.AssignExprStmt) ([]ast.Stmt, bool) {
	gradStmt := &ast.AssignStmt{
		Tok: token.DEFINE,
		Lhs: make([]ast.Expr, len(src.List)),
		Rhs: make([]ast.Expr, len(src.List)),
	}
	ge := sg.newExprGrader(true)
	allKnown := true
	for i, aexpr := range src.List {
		gExpr, ok := ge.gradExpr(aexpr.X)
		if !ok {
			return nil, false
		}
		grIdent := sg.macro.gradIdent(aexpr.Storage.NameDef())
		gradStmt.Lhs[i] = grIdent
		known := sg.define(grIdent.Name, aexpr.Storage.Type())
		allKnown = known && allKnown
		gradStmt.Rhs[i] = gExpr.expr
	}
	if allKnown {
		gradStmt.Tok = token.ASSIGN
	}
	return []ast.Stmt{gradStmt, src.Src}, true
}
