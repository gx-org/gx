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
	"math/big"

	"github.com/gx-org/gx/build/ir"
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

		stmts []ast.Stmt
	}
)

func (m *vjpMacro) newStmt(fetcher ir.Fetcher, parent *stmtVJP) *stmtVJP {
	return &stmtVJP{
		macro:   m,
		fetcher: fetcher,
		parent:  parent,
	}
}

func (sg *stmtVJP) appendMainStmt(stmt ast.Stmt) {
	sg.stmts = append(sg.stmts, stmt)
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

var oneIR = &ir.NumberInt{
	Src: one,
	Val: big.NewInt(1),
}

func (sg *stmtVJP) processStmt(src ir.Stmt) bool {
	switch srcT := src.(type) {
	case *ir.ReturnStmt:
		return sg.returnStmt(srcT)
	default:
		return sg.fetcher.Err().Appendf(src.Source(), "gradient of %T statement not supported", srcT)
	}
}

func (sg *stmtVJP) buildVJPFunctionWRT(src *ir.ReturnStmt, param vjpParam) (*ast.FuncLit, bool) {
	backwarder := sg.newExprBackwardVJP(param.wrt)
	ret := &ast.ReturnStmt{Results: make([]ast.Expr, len(src.Results))}
	for i, expr := range src.Results {
		gradExpr, ok := backwarder.backward(&gradExprResult{
			expr: &ast.Ident{Name: sg.macro.nResults.names[i]},
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
		Type: param.vjpFType,
		Body: &ast.BlockStmt{List: body},
	}, true
}

func (sg *stmtVJP) returnStmt(src *ir.ReturnStmt) bool {
	// Generate forward statements for the expressions in the statement.
	forwarder := sg.newExprForwardVJP()
	ret := &ast.ReturnStmt{Results: make([]ast.Expr, len(src.Results))}
	for i, expr := range src.Results {
		var ok bool
		ret.Results[i], ok = forwarder.forwardSingle(expr)
		if !ok {
			return false
		}
	}
	// Build a backward function for each function parameter.
	names := make([]ast.Expr, len(sg.macro.params))
	for _, param := range sg.macro.params {
		vjpFuncLit, ok := sg.buildVJPFunctionWRT(src, param)
		if !ok {
			return false
		}
		root := "selfVJPFunc"
		if len(names) > 1 {
			root += "WRT" + param.wrt.name()
		}
		vjpFuncName := sg.macro.unames.Name(root)
		sg.appendMainStmt(&ast.AssignStmt{
			Tok: token.DEFINE,
			Lhs: []ast.Expr{&ast.Ident{Name: vjpFuncName}},
			Rhs: []ast.Expr{vjpFuncLit},
		})
		ret.Results = append(ret.Results, &ast.Ident{Name: vjpFuncName})
	}
	sg.appendMainStmt(ret)
	return true
}

func gradIdent(src *ast.Ident) *ast.Ident {
	return &ast.Ident{
		NamePos: src.NamePos,
		Name:    "__grad_" + src.Name,
	}
}
