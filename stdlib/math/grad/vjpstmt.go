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

type (
	vjpExprResult struct {
		name string
		grad *gradExprResult
	}

	stmtVJP struct {
		macro   *vjpMacro
		fetcher ir.Fetcher
		parent  *stmtVJP
		scope   *scope.RWScope[ir.Type]

		stmts []ast.Stmt
	}
)

func (m *vjpMacro) newStmt(fetcher ir.Fetcher, parent *stmtVJP) *stmtVJP {
	var parentScope scope.Scope[ir.Type]
	if parent != nil {
		parentScope = parent.scope
	}
	return &stmtVJP{
		macro:   m,
		fetcher: fetcher,
		parent:  parent,
		scope:   scope.NewScope[ir.Type](parentScope),
	}
}

func (sg *stmtVJP) appendMainStmt(stmt ast.Stmt) {
	sg.stmts = append(sg.stmts, stmt)
}

func (sg *stmtVJP) registerFieldNames(list *ir.FieldList) {
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

func (sg *stmtVJP) newSub() *stmtVJP {
	return sg.macro.newStmt(sg.fetcher, sg)
}

func (sg *stmtVJP) processBlock(src *ir.BlockStmt) (*ast.BlockStmt, bool) {
	sub := sg.newSub()
	for _, stmt := range src.List {
		if ok := sub.processStmt(stmt); !ok {
			return nil, false
		}
	}
	return &ast.BlockStmt{
		List: sub.stmts,
	}, true
}

func (sg *stmtVJP) processStmt(src ir.Stmt) bool {
	switch srcT := src.(type) {
	case *ir.ReturnStmt:
		return sg.returnStmt(srcT)
	default:
		return sg.fetcher.Err().Appendf(src.Source(), "gradient of %T statement not supported", srcT)
	}
}

func (sg *stmtVJP) buildVJPFunction(src *ir.ReturnStmt) (*ast.FuncLit, bool) {
	backwarder := sg.newExprBackwardVJP()
	ret := &ast.ReturnStmt{Results: make([]ast.Expr, len(src.Results))}
	for i, expr := range src.Results {
		gradExpr, ok := backwarder.backward(&gradExprResult{
			expr: &ast.Ident{Name: sg.macro.resultNames[i]},
		}, expr)
		if !ok {
			return nil, false
		}
		ret.Results[i] = gradExpr.expr
	}
	var body []ast.Stmt
	body = append(body, backwarder.stmts...)
	body = append(body, ret)
	return &ast.FuncLit{
		Type: sg.macro.backward,
		Body: &ast.BlockStmt{List: body},
	}, true
}

func (sg *stmtVJP) returnStmt(src *ir.ReturnStmt) bool {
	// Generate forward statements for the expressions in the statement.
	forwarder := sg.newExprForwardVJP()
	ret := &ast.ReturnStmt{Results: make([]ast.Expr, len(src.Results))}
	for i, expr := range src.Results {
		var ok bool
		ret.Results[i], ok = forwarder.forward(expr)
		if !ok {
			return false
		}
	}
	// Build a backward function.
	backward, ok := sg.buildVJPFunction(src)
	if !ok {
		return false
	}
	selfVJPFuncName := sg.macro.unames.Name("selfVJPFunc")
	sg.appendMainStmt(&ast.AssignStmt{
		Tok: token.DEFINE,
		Lhs: []ast.Expr{&ast.Ident{Name: selfVJPFuncName}},
		Rhs: []ast.Expr{backward},
	})
	// Append the backward function to the return.
	ret.Results = append(ret.Results, &ast.Ident{Name: selfVJPFuncName})
	sg.appendMainStmt(ret)
	return true
}

func gradIdent(src *ast.Ident) *ast.Ident {
	return &ast.Ident{
		NamePos: src.NamePos,
		Name:    "__grad_" + src.Name,
	}
}
