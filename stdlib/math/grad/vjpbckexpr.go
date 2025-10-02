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

	"github.com/gx-org/gx/base/uname"
	"github.com/gx-org/gx/build/ir"
)

// exprBackwardVJP decomposes expressions into elementary assignment statements.
type exprBackwardVJP struct {
	*stmtVJP
	wrt   withRespectTo
	stmts []ast.Stmt
}

func (m *stmtVJP) newExprBackwardVJP(wrt withRespectTo) *exprBackwardVJP {
	return &exprBackwardVJP{stmtVJP: m, wrt: wrt}
}

func (m *exprBackwardVJP) singleForwardValue(expr ir.Expr) (forwardValues, bool) {
	fv, ok := m.macro.exprToName[expr]
	if !ok {
		return nil, m.fetcher.Err().AppendInternalf(expr.Source(), "forward expression %T:%s has no name", expr, expr.String())
	}
	if fv.numVars() != 1 {
		return nil, m.fetcher.Err().AppendInternalf(expr.Source(), "forward expression %T:%s has multiple names in a single value context", expr, expr.String())
	}
	return fv, true
}

func (m *exprBackwardVJP) singleForwardName(expr ir.Expr) (_ uname.Name, ok bool) {
	fv, ok := m.singleForwardValue(expr)
	if !ok {
		return
	}
	alloc, ok := fv.(*allocForwardValues)
	if !ok {
		m.fetcher.Err().AppendInternalf(expr.Source(), "forward expression %T:%s has no allocated name", expr, expr.String())
		return
	}
	return alloc.forwards[0], true
}

func (m *exprBackwardVJP) singleForwardIdent(expr ir.Expr) (ast.Expr, bool) {
	fv, ok := m.singleForwardValue(expr)
	if !ok {
		return nil, false
	}
	return fv.idents()[0], true
}

func (m *exprBackwardVJP) appendVJPStmt(stmt ast.Stmt) {
	m.stmts = append(m.stmts, stmt)
}

func (m *exprBackwardVJP) assign(fwdExpr ir.Expr, expr *gradExprResult) (*gradExprResult, bool) {
	name, ok := m.singleForwardName(fwdExpr)
	if !ok {
		return nil, false
	}
	idName := &ast.Ident{Name: name.NameFor(m.macro.bckRoot).String()}
	m.appendVJPStmt(&ast.AssignStmt{
		Tok: token.DEFINE,
		Lhs: []ast.Expr{idName},
		Rhs: []ast.Expr{expr.expr},
	})
	return &gradExprResult{expr: idName}, true
}

func (m *exprBackwardVJP) backward(bck *gradExprResult, expr ir.Expr) (vjp *gradExprResult, ok bool) {
	switch exprT := expr.(type) {
	case *ir.CallExpr:
		return m.callExpr(bck, exprT)
	case *ir.NumberCastExpr:
		return m.numberCastExpr(bck, exprT)
	case *ir.ValueRef:
		return m.valueRef(bck, exprT)
	case *ir.BinaryExpr:
		return m.binaryExpr(bck, exprT)
	default:
		return nil, m.fetcher.Err().Appendf(expr.Source(), "gradient of %T expression not supported", exprT)
	}
}

func (m *exprBackwardVJP) binaryExpr(bck *gradExprResult, expr *ir.BinaryExpr) (*gradExprResult, bool) {
	xBack, xOk := m.backward(bck, expr.X)
	yBack, yOk := m.backward(bck, expr.Y)
	if !xOk || !yOk {
		return nil, false
	}
	switch expr.Src.Op {
	case token.ADD:
		return buildAdd(xBack, yBack), true
	default:
		return nil, m.fetcher.Err().Appendf(expr.Source(), "gradient of binary operator %s not supported", expr.Src.Op)
	}
}

func (m *exprBackwardVJP) numberCastExpr(bck *gradExprResult, expr *ir.NumberCastExpr) (*gradExprResult, bool) {
	src := expr.X.Source().(ast.Expr)
	return zeroValueOf(src), true
}

func (m *exprBackwardVJP) gradFieldStorage(bck *gradExprResult, expr *ir.ValueRef, stor *ir.FieldStorage) (*gradExprResult, bool) {
	if m.wrt.same(stor.Field) {
		return bck, true
	}
	return zeroValueOf(expr.Source()), true
}

func (m *exprBackwardVJP) valueRef(bck *gradExprResult, expr *ir.ValueRef) (*gradExprResult, bool) {
	fieldStorage, isField := expr.Stor.(*ir.FieldStorage)
	if isField {
		return m.gradFieldStorage(bck, expr, fieldStorage)
	}
	return &gradExprResult{expr: gradIdent(expr.Stor.NameDef())}, true
}

func (m *exprBackwardVJP) callExpr(bck *gradExprResult, expr *ir.CallExpr) (*gradExprResult, bool) {
	if len(expr.Args) == 0 {
		return zeroValueOf(expr.Source()), true
	}
	// Call the derivative of the original function with the value of
	// the forward of its argument.
	args := make([]ast.Expr, len(expr.Args))
	for i, arg := range expr.Args {
		var ok bool
		args[i], ok = m.singleForwardIdent(arg)
		if !ok {
			return nil, false
		}
	}
	// Compute the derivative of its arguments.
	forwardValues := m.macro.exprToName[expr].(*allocForwardValues)
	selfCall := &ast.CallExpr{
		Fun:  &ast.Ident{Name: forwardValues.vjp},
		Args: args,
	}
	bckIdent, ok := m.assign(expr, buildMul(bck, &gradExprResult{expr: selfCall}))
	if !ok {
		return nil, false
	}
	argBackwardIdents := make([]*gradExprResult, len(expr.Args))
	for i, argI := range expr.Args {
		var ok bool
		argBackwardIdents[i], ok = m.backward(bckIdent, argI)
		if !ok {
			return nil, false
		}
	}
	if len(argBackwardIdents) == 1 {
		return argBackwardIdents[0], true
	}
	return m.assign(expr, buildAdd(argBackwardIdents...))
}
