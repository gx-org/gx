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

	"github.com/gx-org/gx/base/uname"
	"github.com/gx-org/gx/build/ir"
)

type (
	forwardValues interface {
		numVars() int
		idents() []ast.Expr
	}

	allocForwardValues struct {
		sub      *uname.Root
		forwards []uname.Name
		vjps     []string
	}

	refForwardValue struct {
		name string
	}
)

func (m *exprForwardVJP) newForwardValues(expr ir.Expr, numVals int) *allocForwardValues {
	fv := &allocForwardValues{
		forwards: make([]uname.Name, numVals),
	}
	for i := range numVals {
		fv.forwards[i] = m.macro.fwdRoot.Next()
	}
	m.macro.exprToName[expr] = fv
	return fv
}

func (fv *allocForwardValues) idents() []ast.Expr {
	exprs := make([]ast.Expr, len(fv.forwards))
	for i, name := range fv.forwards {
		exprs[i] = &ast.Ident{Name: name.String()}
	}
	return exprs
}

func (fv *allocForwardValues) numVars() int {
	return len(fv.forwards)
}

func (ref *refForwardValue) idents() []ast.Expr {
	return []ast.Expr{&ast.Ident{Name: ref.name}}
}

func (ref *refForwardValue) numVars() int {
	return 1
}

// exprForwardVJP decomposes expressions into elementary assignment statements.
type exprForwardVJP struct {
	*stmtVJP
}

func (m *stmtVJP) newExprForwardVJP() *exprForwardVJP {
	return &exprForwardVJP{stmtVJP: m}
}

func (m *exprForwardVJP) assignAs(expr ir.Expr) (*allocForwardValues, bool) {
	src, isExpr := expr.Source().(ast.Expr)
	if !isExpr {
		m.fetcher.Err().Appendf(expr.Source(), "%T not an AST expression", expr.Source())
	}
	return m.assignElementary(expr, src), true
}

func (m *exprForwardVJP) assignFuncCall(expr *ir.CallExpr, calleeName string, call *ast.CallExpr) *allocForwardValues {
	nVals := expr.Callee.FuncType().Results.Len()
	fv := m.newForwardValues(expr, nVals)
	stmt := &ast.AssignStmt{
		Tok: token.DEFINE,
		Lhs: fv.idents(),
		Rhs: []ast.Expr{call},
	}
	m.appendMainStmt(stmt)
	root := fmt.Sprintf("%sVJP", calleeName)
	calleeParams := expr.Callee.Func().FuncType().Params.Fields()
	fv.vjps = make([]string, len(calleeParams))
	for i, param := range calleeParams {
		vjpFuncName := root
		if len(calleeParams) > 1 {
			vjpFuncName += param.Name.Name
		}
		fv.vjps[i] = m.macro.unames.Name(vjpFuncName)
		stmt.Lhs = append(stmt.Lhs, &ast.Ident{Name: fv.vjps[i]})
	}
	return fv
}

func (m *exprForwardVJP) assignElementary(expr ir.Expr, elementary ast.Expr) *allocForwardValues {
	fv := m.newForwardValues(expr, 1)
	casted := addCastIfRequired(elementary, expr.Type())
	m.appendMainStmt(&ast.AssignStmt{
		Tok: token.DEFINE,
		Lhs: fv.idents(),
		Rhs: []ast.Expr{casted},
	})
	return fv
}

func asExpr(vars []*ast.Ident) []ast.Expr {
	exprs := make([]ast.Expr, len(vars))
	for i, vr := range vars {
		exprs[i] = vr
	}
	return exprs
}

func (m *exprForwardVJP) forward(expr ir.Expr) (forwardValues, bool) {
	switch exprT := expr.(type) {
	case *ir.CallExpr:
		return m.callExpr(exprT)
	case *ir.NumberCastExpr:
		return m.assignAs(expr)
	case *ir.ValueRef:
		return m.valueRef(exprT)
	case *ir.BinaryExpr:
		return m.binaryExpr(exprT)
	default:
		return nil, m.fetcher.Err().Appendf(expr.Source(), "gradient of %T expression not supported", exprT)
	}
}

func (m *exprForwardVJP) forwardSingle(expr ir.Expr) (ast.Expr, bool) {
	fv, ok := m.forward(expr)
	if !ok {
		return nil, false
	}
	idents := fv.idents()
	if len(idents) != 1 {
		return nil, m.fetcher.Err().AppendInternalf(expr.Source(), "multiple values from expression %T:%s in single value context", expr, expr.String())
	}
	return idents[0], true
}

func (m *exprForwardVJP) binaryExpr(expr *ir.BinaryExpr) (*allocForwardValues, bool) {
	x, xOk := m.forwardSingle(expr.X)
	y, yOk := m.forwardSingle(expr.Y)
	return m.assignElementary(expr, &ast.BinaryExpr{
		Op: expr.Src.Op,
		X:  x,
		Y:  y,
	}), xOk && yOk
}

func (m *exprForwardVJP) callExpr(expr *ir.CallExpr) (*allocForwardValues, bool) {
	if len(expr.Args) == 0 {
		return m.assignAs(expr)
	}
	args := make([]ast.Expr, len(expr.Args))
	for i, argI := range expr.Args {
		var ok bool
		args[i], ok = m.forwardSingle(argI)
		if !ok {
			return nil, false
		}
	}
	calleeT, ok := expr.Callee.(*ir.FuncValExpr)
	if !ok {
		return nil, m.fetcher.Err().AppendInternalf(expr.Source(), "callee type %T not supported", expr.Callee)
	}
	fnName, buildVJPCall, ok := m.macro.vjpFunc(m.fetcher, calleeT)
	if !ok {
		return nil, false
	}
	call := &ast.CallExpr{
		Fun:  buildVJPCall,
		Args: args,
	}
	return m.assignFuncCall(expr, fnName, call), true
}

func (m *exprForwardVJP) numberCastExpr(expr *ir.NumberCastExpr) (*allocForwardValues, bool) {
	src := expr.X.Source().(ast.Expr)
	return m.assignElementary(expr, src), true
}

func (m *exprForwardVJP) valueRef(expr *ir.ValueRef) (*refForwardValue, bool) {
	fv := &refForwardValue{name: expr.Src.Name}
	m.macro.exprToName[expr] = fv
	return fv, true
}
