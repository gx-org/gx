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

type coreStmt[T ir.Stmt] struct {
	node[T]
	fwdStmts *fwdStmts
}

func newCoreStmt[T ir.Stmt](p *processor, isrc T) coreStmt[T] {
	return coreStmt[T]{node: newNodeNoID[T](p, isrc)}
}

type (
	stmt interface {
		build(*astOut) bool
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

func (n *blockStmt) build(outStmts *astOut) bool {
	for _, stmt := range n.stmts {
		if ok := stmt.build(outStmts); !ok {
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
	coreStmt[*ir.ReturnStmt]
	exprs []expr
}

func (p *processor) processReturnStmt(isrc *ir.ReturnStmt) (*returnStmt, bool) {
	out := &returnStmt{
		coreStmt: newCoreStmt[*ir.ReturnStmt](p, isrc),
		exprs:    make([]expr, len(isrc.Results)),
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

func (n *returnStmt) buildVJPFunctionWRT(outStmts *astOut, param VJPParam) (*ast.FuncLit, bool) {
	outWRT := outStmts.newASTOutWRT(param.wrt)
	rets := make([]*special.Expr, len(n.exprs))
	for i, expr := range n.exprs {
		var ok bool
		rets[i], ok = expr.buildBackward(outWRT, special.New(
			&ast.Ident{Name: n.graph.nResults.names[i]},
		))
		if !ok {
			return nil, false
		}
	}
	var body []ast.Stmt
	body = append(body, outWRT.stmts...)
	body = append(body, &ast.ReturnStmt{
		Results: []ast.Expr{special.Add(rets...).RemoveParen().AST()},
	})
	return &ast.FuncLit{
		Type: param.vjpFType,
		Body: &ast.BlockStmt{List: body},
	}, true
}

func (n *returnStmt) build(outStmts *astOut) bool {
	out := &ast.ReturnStmt{
		Results: make([]ast.Expr, len(n.irnode.Results)),
	}
	n.fwdStmts = outStmts.newStmt()
	for i, expr := range n.exprs {
		var ok bool
		out.Results[i], ok = buildSingleForward(n.fwdStmts, expr)
		if !ok {
			return false
		}
	}
	// Build a backward function for each function parameter.
	names := make([]ast.Expr, len(n.graph.params))
	for _, param := range n.graph.params {
		vjpFuncLit, ok := n.buildVJPFunctionWRT(outStmts, param)
		if !ok {
			return false
		}
		root := "selfVJPFunc"
		if len(names) > 1 {
			root += "WRT" + param.wrt.name()
		}
		vjpFuncName := n.graph.unames.Name(root)
		outStmts.append(&ast.AssignStmt{
			Tok: token.DEFINE,
			Lhs: []ast.Expr{&ast.Ident{Name: vjpFuncName}},
			Rhs: []ast.Expr{vjpFuncLit},
		})
		out.Results = append(out.Results, &ast.Ident{Name: vjpFuncName})
	}
	outStmts.append(out)
	return true
}

type assignExprStmt struct {
	coreStmt[*ir.AssignExprStmt]
	exprs []expr
}

func (p *processor) processAssignExprStmt(isrc *ir.AssignExprStmt) (*assignExprStmt, bool) {
	out := &assignExprStmt{
		coreStmt: newCoreStmt[*ir.AssignExprStmt](p, isrc),
		exprs:    make([]expr, len(isrc.List)),
	}
	for i, expr := range isrc.List {
		var ok bool
		out.exprs[i], ok = p.processExpr(expr.X)
		if !ok {
			return nil, false
		}
	}
	return out, true
}

type assignedExpr struct {
	expr

	fwd *ast.Ident
}

func (n *assignedExpr) buildForward(astmts *fwdStmts) ([]ast.Expr, bool) {
	return []ast.Expr{n.fwd}, true
}

func (n *assignedExpr) forwardValue() (*special.Expr, bool) {
	return special.New(n.fwd), true
}

func (n *assignExprStmt) build(outStmts *astOut) bool {
	out := &ast.AssignStmt{
		Tok: token.DEFINE,
		Lhs: make([]ast.Expr, len(n.exprs)),
		Rhs: make([]ast.Expr, len(n.exprs)),
	}
	n.fwdStmts = outStmts.newStmt()
	for i, x := range n.exprs {
		var ok bool
		out.Rhs[i], ok = buildSingleForward(n.fwdStmts, x)
		if !ok {
			return false
		}
		out.Rhs[i] = special.CastIfRequired(out.Rhs[i], n.irnode.List[i].Type())
	}
	for i, x := range n.exprs {
		name := n.irnode.List[i].NameDef().Name
		uname := &ast.Ident{Name: n.graph.unames.Name(name)}
		out.Lhs[i] = uname
		outStmts.setIdentExpr(name, &assignedExpr{
			expr: x,
			fwd:  uname,
		})
	}
	outStmts.append(out)
	return true
}
