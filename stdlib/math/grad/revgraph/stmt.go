// Copyright 2025 Google LLC
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

package revgraph

import (
	"go/ast"
	"go/token"

	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/stdlib/math/grad/special"
)

type (
	stmt interface {
		build(*astStmts) bool
	}

	blockStmt struct {
		stmts []stmt
	}
)

func (p *processor) processBlockStmt(isrc *ir.BlockStmt) (*blockStmt, bool) {
	out := &blockStmt{stmts: make([]stmt, len(isrc.List))}
	for i, isrc := range isrc.List {
		var ok bool
		out.stmts[i], ok = p.processStmt(isrc)
		if !ok {
			return nil, false
		}
	}
	return out, true
}

func (n *blockStmt) build(astmts *astStmts) bool {
	for _, stmt := range n.stmts {
		if ok := stmt.build(astmts); !ok {
			return false
		}
	}
	return true
}

func (p *processor) processStmt(isrc ir.Stmt) (stmt, bool) {
	switch srcT := isrc.(type) {
	case *ir.AssignExprStmt:
		return p.processAssignExprStmt(srcT)
	case *ir.ReturnStmt:
		return p.processReturnStmt(srcT)
	default:
		return nil, p.fetcher.Err().Appendf(isrc.Source(), "gradient of %T statement not supported", srcT)
	}
}

type returnStmt struct {
	node[*ir.ReturnStmt]
	exprs []expr
}

func (p *processor) processReturnStmt(isrc *ir.ReturnStmt) (*returnStmt, bool) {
	out := &returnStmt{
		node:  newNodeNoID[*ir.ReturnStmt](p, isrc),
		exprs: make([]expr, len(isrc.Results)),
	}
	for i, expr := range isrc.Results {
		var ok bool
		out.exprs[i], ok = p.processExpr(expr)
		if !ok {
			return nil, false
		}
	}
	return out, true
}

func (n *returnStmt) buildVJPFunctionWRT(astmts *astStmts, param vjpParam) (*ast.FuncLit, bool) {
	bckstmts := astmts.newBackwardStmts(param.wrt)
	rets := make([]*special.Expr, len(n.exprs))
	for i, expr := range n.exprs {
		var ok bool
		rets[i], ok = expr.buildBackward(bckstmts, special.New(
			&ast.Ident{Name: n.graph.nResults.names[i]},
		))
		if !ok {
			return nil, false
		}
	}
	var body []ast.Stmt
	body = append(body, bckstmts.stmts...)
	body = append(body, &ast.ReturnStmt{
		Results: []ast.Expr{special.Add(rets...).RemoveParen().AST()},
	})
	return &ast.FuncLit{
		Type: param.vjpFType,
		Body: &ast.BlockStmt{List: body},
	}, true
}

func (n *returnStmt) build(astmts *astStmts) bool {
	out := &ast.ReturnStmt{
		Results: make([]ast.Expr, len(n.irnode.Results)),
	}
	for i, expr := range n.exprs {
		var ok bool
		out.Results[i], ok = buildSingleForward(astmts, expr)
		if !ok {
			return false
		}
	}
	// Build a backward function for each function parameter.
	names := make([]ast.Expr, len(n.graph.params))
	for _, param := range n.graph.params {
		vjpFuncLit, ok := n.buildVJPFunctionWRT(astmts, param)
		if !ok {
			return false
		}
		root := "selfVJPFunc"
		if len(names) > 1 {
			root += "WRT" + param.wrt.name()
		}
		vjpFuncName := n.graph.unames.Name(root)
		astmts.append(&ast.AssignStmt{
			Tok: token.DEFINE,
			Lhs: []ast.Expr{&ast.Ident{Name: vjpFuncName}},
			Rhs: []ast.Expr{vjpFuncLit},
		})
		out.Results = append(out.Results, &ast.Ident{Name: vjpFuncName})
	}
	astmts.append(out)
	return true
}

type assignExprStmt struct {
	node[*ir.AssignExprStmt]
	exprs []expr
}

func (p *processor) processAssignExprStmt(isrc *ir.AssignExprStmt) (*assignExprStmt, bool) {
	out := &assignExprStmt{
		node:  newNodeNoID[*ir.AssignExprStmt](p, isrc),
		exprs: make([]expr, len(isrc.List)),
	}
	for i, expr := range isrc.List {
		var ok bool
		out.exprs[i], ok = p.processExpr(expr.X)
		if !ok {
			return nil, false
		}
		p.unames.Register(expr.NameDef().Name)
	}
	return out, true
}

func (n *assignExprStmt) build(astmts *astStmts) bool {
	out := &ast.AssignStmt{
		Tok: n.irnode.Src.Tok,
		Lhs: make([]ast.Expr, len(n.exprs)),
		Rhs: make([]ast.Expr, len(n.exprs)),
	}
	for i, x := range n.exprs {
		name := n.irnode.List[i].NameDef().Name
		out.Lhs[i] = &ast.Ident{Name: name}
		var ok bool
		out.Rhs[i], ok = buildSingleForward(astmts, x)
		if !ok {
			return false
		}
		if !astmts.setIdentExpr(name, x, n.irnode.Src) {
			return false
		}
	}
	astmts.append(out)
	return true
}
