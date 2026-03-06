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
	"github.com/gx-org/gx/stdlib/math/grad/wrt"
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
		return nil, p.fetcher.Err().Appendf(isrc.Node(), "gradient of %T statement not supported", srcT)
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

func (n *returnStmt) buildVJPExprWRT(out *astOutWRT) (ast.Expr, bool) {
	switch wrtT := out.wrt.(type) {
	case *wrt.Array:
		astOutArray := out.newASTOutWRTArray(wrtT)
		return n.buildVJPBodyWRTArray(astOutArray)
	case *wrt.Struct:
		astOutStruct := out.newASTOutWRTStruct(wrtT)
		return n.buildVJPBodyWRTStruct(astOutStruct)
	default:
		return nil, n.err().AppendInternalf(out.wrt.FuncType(), "gradient with respect to %T not supported", wrtT)
	}
}

func (n *returnStmt) buildVJPFunctionWRT(outCore *astOutCore, gradParam wrt.WRT) (*ast.FuncLit, bool) {
	outWRT := outCore.newASTOutWRT(gradParam)
	ret, ok := n.buildVJPExprWRT(outWRT)
	if !ok {
		return nil, false
	}
	var body []ast.Stmt
	body = append(body, outWRT.stmts...)
	body = append(body, &ast.ReturnStmt{
		Results: []ast.Expr{ret},
	})
	return &ast.FuncLit{
		Type: gradParam.FuncType(),
		Body: &ast.BlockStmt{List: body},
	}, true
}

func (n *returnStmt) buildVJPBodyWRTStruct(outWRT *astOutWRTStruct) (ast.Expr, bool) {
	fields := outWRT.wrtStruct.Fields()
	elts := make([]ast.Expr, len(fields))
	for i, field := range fields {
		outField, err := outWRT.newASTOutFromField(field)
		if err != nil {
			return nil, n.err().AppendAt(n.source(), err)
		}
		val, ok := n.buildVJPExprWRT(outField)
		if !ok {
			return nil, false
		}
		elts[i] = &ast.KeyValueExpr{
			Key:   field.Storage().NameDef(),
			Value: val,
		}

	}
	return &ast.CompositeLit{
		Type: outWRT.wrtStruct.Type().NameDef(),
		Elts: elts,
	}, true
}

func (n *returnStmt) buildVJPBodyWRTArray(outWRT *astOutWRTArray) (ast.Expr, bool) {
	rets := make([]*special.Expr, len(n.exprs))
	for i, x := range n.exprs {
		var ok bool
		rets[i], ok = buildBackward(outWRT.astOutWRT, special.New(
			&ast.Ident{Name: n.graph.nResults.names[i]},
		), x)
		if !ok {
			return nil, false
		}
	}
	return special.Add(rets...).RemoveParen().AST(), true
}

func (n *returnStmt) build(out *astOut) bool {
	retStmt := &ast.ReturnStmt{
		Results: make([]ast.Expr, len(n.irnode.Results)),
	}
	n.fwdStmts = out.newStmt()
	for i, expr := range n.exprs {
		var ok bool
		retStmt.Results[i], ok = buildSingleForward(n.fwdStmts, expr)
		if !ok {
			return false
		}
	}
	// Build a backward function for each function parameter.
	names := make([]ast.Expr, len(n.graph.wrts))
	for _, gradParam := range n.graph.wrts {
		vjpFuncLit, ok := n.buildVJPFunctionWRT(out.astOutCore, gradParam)
		if !ok {
			return false
		}
		root := "selfVJPFunc"
		if len(names) > 1 {
			root += "WRT" + wrt.ToName(gradParam.Name())
		}
		vjpFuncName := n.graph.unames.Name(root)
		out.append(&ast.AssignStmt{
			Tok: token.DEFINE,
			Lhs: []ast.Expr{&ast.Ident{Name: vjpFuncName}},
			Rhs: []ast.Expr{vjpFuncLit},
		})
		retStmt.Results = append(retStmt.Results, &ast.Ident{Name: vjpFuncName})
	}
	out.append(retStmt)
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
