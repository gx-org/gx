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
	"go/token"

	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/stdlib/math/grad/special"
)

// exprGrader computes the gradient of expression.
type exprGrader struct {
	*stmtGrader
	castNumbers bool
}

func (m *stmtGrader) newExprGrader(castNumbers bool) *exprGrader {
	return &exprGrader{
		stmtGrader:  m,
		castNumbers: castNumbers,
	}
}

func (m *exprGrader) gradExpr(src ir.Expr) (r *special.Expr, ok bool) {
	defer func() {
		if !ok || !m.castNumbers {
			return
		}
		r = r.AddCastIfRequired(src.Type())
	}()
	switch srcT := src.(type) {
	case *ir.ArrayLitExpr:
		return m.gradArrayLitExpr(srcT)
	case *ir.NumberCastExpr:
		return special.ZeroExpr(), true
	case *ir.ValueRef:
		return m.gradValueRef(srcT)
	case *ir.BinaryExpr:
		return m.gradBinaryExpr(srcT)
	case *ir.ParenExpr:
		return m.gradParenExpr(srcT)
	case *ir.FuncCallExpr:
		return m.gradCall(srcT)
	case *ir.SelectorExpr:
		return m.gradSelectorExpr(srcT)
	default:
		return nil, m.fetcher.Err().Appendf(src.Source(), "gradient of %T expression not supported", srcT)
	}
}

func (m *exprGrader) gradBinaryExpr(src *ir.BinaryExpr) (*special.Expr, bool) {
	u := special.NewFromIR(src.X)
	v := special.NewFromIR(src.Y)
	uGrad, xOk := m.gradExpr(src.X)
	vGrad, yOk := m.gradExpr(src.Y)
	if !xOk || !yOk {
		return nil, false
	}
	switch src.Src.Op {
	case token.ADD:
		return special.Add(uGrad, vGrad), true
	case token.SUB:
		return special.Sub(uGrad, vGrad), true
	case token.MUL:
		return special.Add(
			special.Mul(uGrad, v),
			special.Mul(u, vGrad),
		), true
	case token.QUO:
		return special.Quo(
			special.Sub(
				special.Mul(uGrad, v),
				special.Mul(u, vGrad),
			),
			special.Mul(v, v),
		), true
	default:
		return nil, m.fetcher.Err().Appendf(src.Source(), "gradient of binary expression %s not supported", src.Src.Op)
	}
}

func (m *exprGrader) gradParenExpr(src *ir.ParenExpr) (*special.Expr, bool) {
	expr, ok := m.gradExpr(src.X)
	if !ok {
		return expr, false
	}
	return special.Paren(expr), true
}

func (m *exprGrader) gradFieldStorage(src *ir.FieldStorage) (*special.Expr, bool) {
	if m.macro.wrt.same(src.Field) {
		return special.OneExpr(src.Source()), true
	}
	return special.ZeroExpr(), true
}

func (m *exprGrader) gradValueRef(src *ir.ValueRef) (*special.Expr, bool) {
	fieldStorage, isField := src.Stor.(*ir.FieldStorage)
	if isField {
		return m.gradFieldStorage(fieldStorage)
	}
	gIdent := m.macro.gradIdent(src.Stor.NameDef())
	return special.New(gIdent), true
}

func (m *exprGrader) gradArrayLitExpr(src *ir.ArrayLitExpr) (*special.Expr, bool) {
	allZero := true
	var acc *special.Expr = special.ZeroExpr()
	for _, expr := range src.Values() {
		gExpr, ok := m.gradExpr(expr)
		if !ok {
			return nil, false
		}
		if gExpr.IsZero() {
			continue
		}
		allZero = false
		acc = special.Add(acc, gExpr)
	}
	if allZero {
		return special.ZeroExpr(), true
	}
	return acc, true
}

func (m *exprGrader) gradSelectorExpr(src *ir.SelectorExpr) (*special.Expr, bool) {
	fieldStorage, isField := src.Stor.(*ir.FieldStorage)
	if isField {
		return m.gradFieldStorage(fieldStorage)
	}
	return special.ZeroExpr(), true
}
