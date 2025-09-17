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

func (m *exprForwardVJP) assignElementary(expr ir.Expr, elementary ast.Expr) *ast.Ident {
	name := &ast.Ident{Name: fmt.Sprintf("__fwd%d", m.numForward)}
	m.numForward++
	m.macro.exprToName[expr] = name.Name
	casted := addCastIfRequired(elementary, expr.Type())
	m.appendStmt(&ast.AssignStmt{
		Tok: token.DEFINE,
		Lhs: []ast.Expr{name},
		Rhs: []ast.Expr{casted},
	})
	return name
}

func (m *exprForwardVJP) forward(expr ir.Expr) (*ast.Ident, bool) {
	switch exprT := expr.(type) {
	case *ir.NumberCastExpr:
		return m.vjpNumberCastExpr(exprT)
	case *ir.ValueRef:
		return m.vjpValueRef(exprT)
	default:
		return nil, m.fetcher.Err().Appendf(expr.Source(), "gradient of %T expression not supported", exprT)
	}
}

func (m *exprForwardVJP) vjpNumberCastExpr(expr *ir.NumberCastExpr) (*ast.Ident, bool) {
	src := expr.X.Source().(ast.Expr)
	return m.assignElementary(expr, src), true
}

func (m *exprForwardVJP) vjpValueRef(expr *ir.ValueRef) (*ast.Ident, bool) {
	return expr.Stor.NameDef(), true
}
