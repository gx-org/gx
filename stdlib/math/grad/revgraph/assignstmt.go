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
	"fmt"
	"go/ast"
	"go/token"
	"strconv"

	"github.com/gx-org/gx/build/fmterr"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/stdlib/math/grad/special"
)

type astStmts struct {
	graph   *Graph
	fetcher ir.Fetcher
	uroot   string
	stmts   []ast.Stmt
}

func (g *Graph) newForwardStmts(fetcher ir.Fetcher) *astStmts {
	return &astStmts{graph: g, fetcher: fetcher, uroot: "fwd"}
}

func (a *astStmts) append(s ast.Stmt) {
	a.stmts = append(a.stmts, s)
}

func (a *astStmts) assignExpr(id nodeID, expr ast.Expr) *ast.Ident {
	return a.assignExprs(id, []ast.Expr{expr}, 1, "")[0]
}

func (a *astStmts) buildIdents(id nodeID, n int, suffix string) []*ast.Ident {
	idents := make([]*ast.Ident, n)
	for i := range n {
		var root string
		if n == 1 {
			root = fmt.Sprintf("%s%d%s", a.uroot, id, suffix)
		} else {
			root = fmt.Sprintf("%s%dr%d%s", a.uroot, id, i, suffix)
		}
		idents[i] = &ast.Ident{Name: a.graph.unames.Name(root)}
	}
	return idents
}

func toExprs(idents []*ast.Ident) []ast.Expr {
	exprs := make([]ast.Expr, len(idents))
	for i, ident := range idents {
		exprs[i] = ident
	}
	return exprs
}

func (a *astStmts) assignExprs(id nodeID, expr []ast.Expr, n int, suffix string) []*ast.Ident {
	idents := a.buildIdents(id, n, suffix)
	a.stmts = append(a.stmts, &ast.AssignStmt{
		Tok: token.DEFINE,
		Lhs: toExprs(idents),
		Rhs: expr,
	})
	return idents
}

func (a *astStmts) callTraceSpecials(exprs []*special.Expr) {
	if !traceAll {
		return
	}
	var idents []*ast.Ident
	for _, expr := range exprs {
		id, ok := expr.Expr.(*ast.Ident)
		if !ok {
			continue
		}
		idents = append(idents, id)
	}
	a.callTrace(idents)
}

func (a *astStmts) callTrace(idents []*ast.Ident) {
	if !traceAll {
		return
	}
	if len(idents) == 0 {
		return
	}
	var traceArgs []ast.Expr
	for _, ident := range idents {
		traceArgs = append(traceArgs,
			&ast.BasicLit{Value: strconv.Quote(ident.Name)},
			ident,
		)

	}
	a.stmts = append(a.stmts, &ast.ExprStmt{
		X: &ast.CallExpr{
			Fun:  &ast.Ident{Name: "trace"},
			Args: traceArgs,
		},
	})
}

func (a *astStmts) assignFuncCall(id nodeID, ftype *ir.FuncType, calleeName string, call *ast.CallExpr) ([]*ast.Ident, []string) {
	nVals := ftype.Results.Len()
	idents := a.buildIdents(id, nVals, "")
	stmt := &ast.AssignStmt{
		Tok: token.DEFINE,
		Lhs: toExprs(idents),
		Rhs: []ast.Expr{call},
	}
	a.stmts = append(a.stmts, stmt)
	root := fmt.Sprintf("%sVJP", calleeName)
	calleeParams := ftype.Params.Fields()
	vjps := make([]string, len(calleeParams))
	for i, param := range calleeParams {
		vjpFuncName := root
		if len(calleeParams) > 1 {
			vjpFuncName += param.Name.Name
		}
		vjps[i] = a.graph.unames.Name(vjpFuncName)
		stmt.Lhs = append(stmt.Lhs, &ast.Ident{Name: vjps[i]})
	}
	a.callTrace(idents)
	return idents, vjps
}

func (a *astStmts) err() *fmterr.Appender {
	return a.fetcher.Err()
}

type bckStmts struct {
	astStmts
	wrt withRespectTo
}

func (a *astStmts) newBackwardStmts(wrt withRespectTo) *bckStmts {
	return &bckStmts{
		astStmts: astStmts{
			graph:   a.graph,
			fetcher: a.fetcher,
			uroot:   "bck",
		},
		wrt: wrt,
	}
}

func (b *bckStmts) assignSpecialExpr(id nodeID, expr *special.Expr) *special.Expr {
	if expr.Value != special.Any {
		return expr
	}
	return &special.Expr{Expr: b.assignExpr(id, expr.Expr)}
}
