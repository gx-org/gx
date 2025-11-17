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
	"go/token"

	"github.com/pkg/errors"
	"github.com/gx-org/gx/build/fmterr"
	"github.com/gx-org/gx/build/ir"
)

type blockStmt struct {
	src   *ast.BlockStmt
	stmts []stmtNode
}

var _ stmtNode = (*blockStmt)(nil)

func processBlockStmt(pscope procScope, src *ast.BlockStmt) (*blockStmt, bool) {
	n := &blockStmt{src: src}
	ok := true
	for _, stmt := range src.List {
		node, nOk := processStmt(pscope, stmt)
		if node != nil {
			n.stmts = append(n.stmts, node)
		}
		ok = ok && nOk
	}
	return n, ok
}

func (n *blockStmt) source() ast.Node {
	return n.src
}

func (n *blockStmt) buildStmt(scope fnResolveScope) (ir.Stmt, bool, bool) {
	return n.buildBlockStmt(scope)
}

func (n *blockStmt) buildBlockStmt(parentScope fnResolveScope) (*ir.BlockStmt, bool, bool) {
	scope, ok := newBlockScope(parentScope, n)
	if !ok {
		return nil, false, false
	}
	block := &ir.BlockStmt{
		Src:  n.src,
		List: make([]ir.Stmt, len(n.stmts)),
	}
	stop := false
	unreachableReported := false
	for i, node := range n.stmts {
		if stop && !unreachableReported {
			ok = scope.Err().Appendf(node.source(), "unreachable code")
			unreachableReported = true
		}
		var stmtStop, stmtOk bool
		block.List[i], stmtStop, stmtOk = node.buildStmt(scope)
		ok = ok && stmtOk
		stop = stop || stmtStop
	}
	if !ok && scope.Err().Empty() {
		// an error occurred but no explicit error has been added to the list of error.
		// report an error now to help with debugging.
		scope.Err().AppendInternalf(n.src, "failed to build statement but no error reported to the user")
	}
	return block, stop, ok
}

func buildFuncBody(scope fnResolveScope, body *blockStmt) (*ir.BlockStmt, bool) {
	irBody, stop, ok := body.buildBlockStmt(scope)
	if !stop && scope.funcType().Results.Len() > 0 {
		ok = scope.Err().Append(fmterr.AtPos(scope.Err().FSet().FSet, body.src.End(), errors.Errorf("missing return")))
	}
	return irBody, ok
}

type exprStmt struct {
	src *ast.ExprStmt
	x   exprNode
}

var _ stmtNode = (*exprStmt)(nil)

func processExprStmt(pscope procScope, src *ast.ExprStmt) (*exprStmt, bool) {
	n := &exprStmt{src: src}
	var ok bool
	n.x, ok = processExpr(pscope, src.X)
	return n, ok
}

func (n *exprStmt) source() ast.Node {
	return n.src
}

// declStmt represents a 'var' declaration statement in a function body.
type declStmt struct {
	src   *ast.DeclStmt
	decls []*varSpec
}

var _ stmtNode = (*declStmt)(nil)

func processDeclStmt(pscope procScope, src *ast.DeclStmt) (stmtNode, bool) {
	genDecl, isGenDecl := src.Decl.(*ast.GenDecl)
	if !isGenDecl || genDecl.Tok != token.VAR {
		return nil, pscope.Err().Appendf(src, "unsupported declaration in function body (only 'var' is supported)")
	}

	n := &declStmt{src: src}
	ok := true

	for _, spec := range genDecl.Specs {
		valueSpec, isValueSpec := spec.(*ast.ValueSpec)
		if !isValueSpec {
			ok = pscope.Err().Appendf(spec, "internal error: expected var declaration to contain value specs, got %T", spec)
			continue
		}

		vs := &varSpec{src: valueSpec}

		if valueSpec.Type == nil {
			ok = pscope.Err().Appendf(valueSpec, "local variable declaration must have a type")
			continue
		}
		var typeOk bool
		vs.typ, typeOk = processTypeExpr(pscope.axisLengthScope(), valueSpec.Type)
		ok = ok && typeOk

		if len(valueSpec.Values) > 0 {
			ok = pscope.Err().Appendf(valueSpec, "local variable with an initial value is not yet supported")
		}

		namesInSpec := make(map[string]bool)
		for _, name := range valueSpec.Names {
			if namesInSpec[name.Name] {
				ok = pscope.Err().Appendf(name, "variable %q redeclared in this block", name.Name)
				continue
			}
			namesInSpec[name.Name] = true
			vr := &varExpr{
				spec: vs,
				name: name,
			}
			vs.exprs = append(vs.exprs, vr)
		}
		n.decls = append(n.decls, vs)
	}

	return n, ok
}

func (n *declStmt) source() ast.Node {
	return n.src
}

// buildStmt builds the IR for a declaration statement.
func (n *declStmt) buildStmt(scope fnResolveScope) (ir.Stmt, bool, bool) {
	decls := make([]*ir.VarSpec, 0, len(n.decls))
	declsOk := true

	for _, d := range n.decls {
		varSpec, declOk := d.buildDecl(scope)
		if !declOk {
			declsOk = false
			continue
		}

		for _, varExpr := range varSpec.Exprs {
			if !defineLocalVar(scope, varExpr) {
				declsOk = false
			}
		}

		decls = append(decls, varSpec)
	}

	return &ir.DeclStmt{Src: n.src, Decls: decls}, false, declsOk
}

func (n *exprStmt) buildStmt(scope fnResolveScope) (ir.Stmt, bool, bool) {
	x, ok := n.x.buildExpr(scope)
	if ok && x.Type().Kind() != ir.VoidKind {
		scope.Err().Appendf(n.src, "cannot use an expression returning a value as a statement")
	}
	return &ir.ExprStmt{Src: n.src, X: x}, false, ok
}

func processStmt(pscope procScope, stmt ast.Stmt) (node stmtNode, ok bool) {
	switch s := stmt.(type) {
	case *ast.AssignStmt:
		node, ok = processAssign(pscope, s)
	case *ast.RangeStmt:
		node, ok = processRangeStmt(pscope, s)
	case *ast.ReturnStmt:
		node, ok = processReturnStmt(pscope, s)
	case *ast.IfStmt:
		node, ok = processIfStmt(pscope, s)
	case *ast.BlockStmt:
		node, ok = processBlockStmt(pscope, s)
	case *ast.ExprStmt:
		node, ok = processExprStmt(pscope, s)
	case *ast.DeclStmt:
		node, ok = processDeclStmt(pscope, s)
	default:
		ok = pscope.Err().Appendf(stmt, "statement type not supported: %T", stmt)
	}
	return
}
