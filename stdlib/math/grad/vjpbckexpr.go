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

// exprBackwardVJP decomposes expressions into elementary assignment statements.
type exprBackwardVJP struct {
	*stmtVJP
}

func (m *stmtVJP) newExprBackwardVJP() *exprBackwardVJP {
	return &exprBackwardVJP{stmtVJP: m}
}

func (m *exprBackwardVJP) backward(expr ir.Expr) (*gradExprResult, bool) {
	switch exprT := expr.(type) {
	case *ir.NumberCastExpr:
		return m.vjpNumberCastExpr(exprT)
	case *ir.ValueRef:
		return m.vjpValueRef(exprT)
	default:
		return nil, m.fetcher.Err().Appendf(expr.Source(), "gradient of %T expression not supported", exprT)
	}
}

func (m *exprBackwardVJP) vjpNumberCastExpr(expr *ir.NumberCastExpr) (*gradExprResult, bool) {
	src := expr.X.Source().(ast.Expr)
	return zeroValueOf(src), true
}

func (m *exprBackwardVJP) gradFieldStorage(expr *ir.ValueRef, stor *ir.FieldStorage) (*gradExprResult, bool) {
	if m.macro.wrt.same(stor.Field) {
		return oneValueOf(expr.Source()), true
	}
	return zeroValueOf(expr.Source()), true
}

func (m *exprBackwardVJP) vjpValueRef(expr *ir.ValueRef) (*gradExprResult, bool) {
	fieldStorage, isField := expr.Stor.(*ir.FieldStorage)
	if isField {
		return m.gradFieldStorage(expr, fieldStorage)
	}
	return &gradExprResult{expr: gradIdent(expr.Stor.NameDef())}, true
}
