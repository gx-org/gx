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
)

// exprForwardVJP decomposes expressions into elementary assignment statements.
type exprForwardVJP struct {
	*stmtVJP
}

func (m *stmtVJP) newExprForwardVJP() *exprForwardVJP {
	return &exprForwardVJP{stmtVJP: m}
}

func (m *exprForwardVJP) assignAs(expr ir.Expr) (*ast.Ident, bool) {
	src, isExpr := expr.Source().(ast.Expr)
	if !isExpr {
		m.fetcher.Err().Appendf(expr.Source(), "%T not an AST expression", expr.Source())
	}
	return m.assignElementary(expr, src), true
}

func (m *exprForwardVJP) nextForwardName() *ast.Ident {
	name := m.macro.unames.NameNumbered("fwd")
	return &ast.Ident{Name: name}
}

func (m *exprForwardVJP) assignFuncCall(expr ir.Expr, calleeName string, call *ast.CallExpr) *ast.Ident {
	name := m.nextForwardName()
	vjpFuncName := m.macro.unames.Name(fmt.Sprintf("%sVJP", calleeName))
	m.macro.exprToName[expr] = forwardValues{
		forwards: []string{name.Name},
		vjp:      vjpFuncName,
	}
	m.appendMainStmt(&ast.AssignStmt{
		Tok: token.DEFINE,
		Lhs: []ast.Expr{name, &ast.Ident{Name: vjpFuncName}},
		Rhs: []ast.Expr{call},
	})
	return name
}

func (m *exprForwardVJP) assignElementary(expr ir.Expr, elementary ast.Expr) *ast.Ident {
	name := m.nextForwardName()
	m.macro.exprToName[expr] = m.macro.exprToName[expr].add(name.Name)
	casted := addCastIfRequired(elementary, expr.Type())
	m.appendMainStmt(&ast.AssignStmt{
		Tok: token.DEFINE,
		Lhs: []ast.Expr{name},
		Rhs: []ast.Expr{casted},
	})
	return name
}

func asExpr(vars []*ast.Ident) []ast.Expr {
	exprs := make([]ast.Expr, len(vars))
	for i, vr := range vars {
		exprs[i] = vr
	}
	return exprs
}

func (m *exprForwardVJP) assignForwardFuncCall(expr ir.Expr, call ast.Expr, numResults int) []*ast.Ident {
	// Build all the variables to receive values.
	fwdVars := make([]*ast.Ident, numResults)
	for i := range numResults {
		fwdVars[i] = m.nextForwardName()
	}
	funcFwdName := m.nextForwardName()
	fwdVars = append(fwdVars, funcFwdName)
	m.appendMainStmt(&ast.AssignStmt{
		Tok: token.DEFINE,
		Lhs: asExpr(fwdVars),
		Rhs: []ast.Expr{call},
	})
	return fwdVars
}

func (m *exprForwardVJP) forward(expr ir.Expr) (*ast.Ident, bool) {
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

func (m *exprForwardVJP) binaryExpr(expr *ir.BinaryExpr) (*ast.Ident, bool) {
	x, xOk := m.forward(expr.X)
	y, yOk := m.forward(expr.Y)
	return m.assignElementary(expr, &ast.BinaryExpr{
		Op: expr.Src.Op,
		X:  x,
		Y:  y,
	}), xOk && yOk
}

func (m *exprForwardVJP) callExpr(expr *ir.CallExpr) (*ast.Ident, bool) {
	if len(expr.Args) == 0 {
		return m.assignAs(expr)
	}
	args := make([]ast.Expr, len(expr.Args))
	for i, argI := range expr.Args {
		var ok bool
		args[i], ok = m.forward(argI)
		if !ok {
			return nil, false
		}
	}
	calleeT, ok := expr.Callee.(*ir.FuncValExpr)
	if !ok {
		return nil, m.fetcher.Err().AppendInternalf(expr.Source(), "callee type %T not supported", expr.Callee)
	}
	fnName, buildVJPCall, ok := m.macro.vjpFunc(m.fetcher, calleeT, m.macro.wrt.name())
	if !ok {
		return nil, false
	}
	call := &ast.CallExpr{
		Fun:  buildVJPCall,
		Args: args,
	}
	return m.assignFuncCall(expr, fnName, call), true
}

func (m *exprForwardVJP) numberCastExpr(expr *ir.NumberCastExpr) (*ast.Ident, bool) {
	src := expr.X.Source().(ast.Expr)
	return m.assignElementary(expr, src), true
}

func (m *exprForwardVJP) valueRef(expr *ir.ValueRef) (*ast.Ident, bool) {
	m.macro.exprToName[expr] = m.macro.exprToName[expr].add(expr.Src.Name)
	return expr.Stor.NameDef(), true
}
