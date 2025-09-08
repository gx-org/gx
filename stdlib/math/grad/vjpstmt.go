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
	"fmt"
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

		exprToGrad map[ir.Expr]*vjpExprResult
		block      []ast.Stmt
	}
)

func (m *vjpMacro) newStmt(fetcher ir.Fetcher, parent *stmtVJP) *stmtVJP {
	var exprToGrad map[ir.Expr]*vjpExprResult
	var parentScope scope.Scope[ir.Type]
	if parent != nil {
		parentScope = parent.scope
		exprToGrad = parent.exprToGrad
	} else {
		exprToGrad = make(map[ir.Expr]*vjpExprResult)
	}
	return &stmtVJP{
		macro:      m,
		fetcher:    fetcher,
		parent:     parent,
		scope:      scope.NewScope[ir.Type](parentScope),
		exprToGrad: exprToGrad,
	}
}

func (sg *stmtVJP) appendStmt(stmt ast.Stmt) {
	sg.block = append(sg.block, stmt)
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

func (sg *exprVJP) assignElementary(expr ir.Expr, elementary ast.Expr, grad *gradExprResult) {
	name := fmt.Sprintf("__fwd%d", len(sg.exprToGrad))
	sg.assignGradAs(expr, name, grad)
	casted := addCastIfRequired(elementary, expr.Type())
	sg.appendStmt(&ast.AssignStmt{
		Tok: token.DEFINE,
		Lhs: []ast.Expr{&ast.Ident{Name: name}},
		Rhs: []ast.Expr{casted},
	})
}

func (sg *exprVJP) assignGradAs(expr ir.Expr, name string, grad *gradExprResult) {
	sg.exprToGrad[expr] = &vjpExprResult{
		name: name,
		grad: grad,
	}
}

func (sg *stmtVJP) processBlock(src *ir.BlockStmt) (*ast.BlockStmt, bool) {
	sub := sg.newSub()
	for _, stmt := range src.List {
		if ok := sub.processStmt(stmt); !ok {
			return nil, false
		}
	}
	return &ast.BlockStmt{
		List: sub.block,
	}, true
}

func (sg *stmtVJP) processStmt(src ir.Stmt) bool {
	switch srcT := src.(type) {
	case *ir.ReturnStmt:
		return sg.gradReturnStmt(srcT)
	default:
		return sg.fetcher.Err().Appendf(src.Source(), "gradient of %T statement not supported", srcT)
	}
}

func (sg *stmtVJP) gradReturnStmt(src *ir.ReturnStmt) bool {
	// Generate forward statements for the expressions in the statement.
	ge := sg.newExprVJP()
	for _, expr := range src.Results {
		if ok := ge.process(expr); !ok {
			return false
		}
	}
	// Append to the results the names of the variables of the original expressions.
	var res []ast.Expr
	for _, expr := range src.Results {
		res = append(res, &ast.Ident{Name: sg.exprToGrad[expr].name})
	}
	for i, expr := range src.Results {
		mul := buildMul(
			&gradExprResult{expr: argAt(i)},
			sg.exprToGrad[expr].grad,
		)
		res = append(res, mul.expr)
	}
	sg.appendStmt(&ast.ReturnStmt{Results: res})
	return true
}

func gradIdent(src *ast.Ident) *ast.Ident {
	return &ast.Ident{
		NamePos: src.NamePos,
		Name:    "__grad_" + src.Name,
	}
}
