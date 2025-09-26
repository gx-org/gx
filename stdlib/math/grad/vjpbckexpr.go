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

	"github.com/gx-org/gx/build/ir"
)

// exprBackwardVJP decomposes expressions into elementary assignment statements.
type exprBackwardVJP struct {
	*stmtVJP
	stmts []ast.Stmt
}

func (m *stmtVJP) newExprBackwardVJP() *exprBackwardVJP {
	return &exprBackwardVJP{stmtVJP: m}
}

func (m *exprBackwardVJP) appendVJPStmt(stmt ast.Stmt) {
	m.stmts = append(m.stmts, stmt)
}

func (m *exprBackwardVJP) nextBackwardName() *ast.Ident {
	name := m.macro.unames.NameNumbered("bck")
	return &ast.Ident{Name: name}
}

func (m *exprBackwardVJP) assign(expr *gradExprResult) *gradExprResult {
	name := m.nextBackwardName()
	m.appendVJPStmt(&ast.AssignStmt{
		Tok: token.DEFINE,
		Lhs: []ast.Expr{name},
		Rhs: []ast.Expr{expr.expr},
	})
	return &gradExprResult{expr: name}
}

func (m *exprBackwardVJP) backward(bck *gradExprResult, expr ir.Expr) (vjp *gradExprResult, ok bool) {
	switch exprT := expr.(type) {
	case *ir.CallExpr:
		return m.callExpr(bck, exprT)
	case *ir.NumberCastExpr:
		return m.numberCastExpr(bck, exprT)
	case *ir.ValueRef:
		return m.valueRef(bck, exprT)
	default:
		return nil, m.fetcher.Err().Appendf(expr.Source(), "gradient of %T expression not supported", exprT)
	}
}

func (m *exprBackwardVJP) numberCastExpr(bck *gradExprResult, expr *ir.NumberCastExpr) (*gradExprResult, bool) {
	src := expr.X.Source().(ast.Expr)
	return zeroValueOf(src), true
}

func (m *exprBackwardVJP) gradFieldStorage(bck *gradExprResult, expr *ir.ValueRef, stor *ir.FieldStorage) (*gradExprResult, bool) {
	if m.macro.wrt.same(stor.Field) {
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
		fwdVals := m.macro.exprToName[arg]
		args[i] = &ast.Ident{Name: fwdVals.forwards[0]}
	}
	// Compute the derivative of its arguments.
	forwardValues := m.macro.exprToName[expr]
	selfCall := &ast.CallExpr{
		Fun:  &ast.Ident{Name: forwardValues.vjp},
		Args: args,
	}
	bckIdent := m.assign(buildMul(bck, &gradExprResult{expr: selfCall}))
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
	return m.assign(buildAdd(argBackwardIdents...)), true
}
