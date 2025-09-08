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

	"github.com/gx-org/gx/build/ir"
)

// exprVJP decomposes expressions into elementary assignment statements.
type exprVJP struct {
	*stmtVJP
}

func (m *stmtVJP) newExprVJP() *exprVJP {
	return &exprVJP{stmtVJP: m}
}

func (m *exprVJP) process(expr ir.Expr) bool {
	switch exprT := expr.(type) {
	case *ir.NumberCastExpr:
		return m.vjpNumberCastExpr(exprT)
	case *ir.ValueRef:
		return m.vjpValueRef(exprT)
	default:
		return m.fetcher.Err().Appendf(expr.Source(), "gradient of %T expression not supported", exprT)
	}
}

func (m *exprVJP) vjpNumberCastExpr(expr *ir.NumberCastExpr) bool {
	src := expr.X.Source().(ast.Expr)
	grad := zeroValueOf(src)
	m.assignElementary(expr, src, grad)
	return true
}

func (m *exprVJP) gradFieldStorage(expr *ir.ValueRef, stor *ir.FieldStorage) bool {
	var grad *gradExprResult
	if m.macro.wrt.same(stor.Field) {
		grad = oneValueOf(expr.Source())
	} else {
		grad = zeroValueOf(expr.Source())
	}
	m.assignGradAs(expr, stor.NameDef().Name, grad)
	return true
}

func (m *exprVJP) vjpValueRef(expr *ir.ValueRef) bool {
	fieldStorage, isField := expr.Stor.(*ir.FieldStorage)
	if isField {
		return m.gradFieldStorage(expr, fieldStorage)
	}
	gIdent := gradIdent(expr.Stor.NameDef())
	m.assignGradAs(expr, expr.Stor.NameDef().Name, &gradExprResult{
		expr: gIdent,
	})
	return true
}
