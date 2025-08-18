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

type specialValue int

const (
	notSpecial = iota
	zeroSpecial
	oneSpecial
)

type gradExprResult struct {
	kind specialValue
	expr ast.Expr
}

func (r *gradExprResult) addCastIfRequired(typ ir.Type) (*gradExprResult, bool) {
	basic, isBasic := r.expr.(*ast.BasicLit)
	if !isBasic {
		return r, true
	}
	if basic.Kind != token.INT && basic.Kind != token.FLOAT {
		return r, true
	}
	return &gradExprResult{
		kind: r.kind,
		expr: &ast.CallExpr{
			Fun:  typ.Source().(ast.Expr),
			Args: []ast.Expr{r.expr},
		},
	}, true
}

func (r *gradExprResult) Print() {
	ast.Print(token.NewFileSet(), r.expr)
}

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

func (m *exprGrader) gradExpr(src ir.Expr) (r *gradExprResult, ok bool) {
	defer func() {
		if !ok || !m.castNumbers {
			return
		}
		r, ok = r.addCastIfRequired(src.Type())
	}()
	switch srcT := src.(type) {
	case *ir.ArrayLitExpr:
		return m.gradArrayLitExpr(srcT)
	case *ir.NumberCastExpr:
		return zeroValueOf(src.Source()), true
	case *ir.ValueRef:
		return m.gradValueRef(srcT)
	case *ir.BinaryExpr:
		return m.gradBinaryExpr(srcT)
	case *ir.ParenExpr:
		return m.gradParenExpr(srcT)
	case *ir.CallExpr:
		return m.gradCall(srcT)
	case *ir.SelectorExpr:
		return m.gradSelectorExpr(srcT)
	default:
		return nil, m.fetcher.Err().Appendf(src.Source(), "gradient of %T expression not supported", srcT)
	}
}

func buildAdd(x, y *gradExprResult) *gradExprResult {
	if x.kind == zeroSpecial {
		return y
	}
	if y.kind == zeroSpecial {
		return x
	}
	return &gradExprResult{
		expr: &ast.BinaryExpr{
			Op: token.ADD,
			X:  x.expr,
			Y:  y.expr,
		},
	}
}

func buildMul(x, y *gradExprResult) *gradExprResult {
	// Multiplication by 0.
	if x.kind == zeroSpecial {
		return x
	}
	if y.kind == zeroSpecial {
		return y
	}
	// Multiplication by 1.
	if x.kind == oneSpecial {
		return y
	}
	if y.kind == oneSpecial {
		return x
	}
	// All other cases.
	return &gradExprResult{
		expr: &ast.BinaryExpr{
			Op: token.MUL,
			X:  x.expr,
			Y:  y.expr,
		},
	}
}

func buildSub(x, y *gradExprResult) *gradExprResult {
	// Substraction by 0.
	if y.kind == zeroSpecial {
		return x
	}
	// Substraction from 0.
	if x.kind == zeroSpecial {
		return &gradExprResult{
			expr: &ast.UnaryExpr{
				Op: token.SUB,
				X:  y.expr,
			},
		}
	}
	return &gradExprResult{
		expr: &ast.BinaryExpr{
			Op: token.SUB,
			X:  x.expr,
			Y:  y.expr,
		},
	}
}

func buildQuo(x, y *gradExprResult) *gradExprResult {
	if y.kind == oneSpecial {
		return x
	}
	return &gradExprResult{
		expr: &ast.BinaryExpr{
			Op: token.QUO,
			X:  x.expr,
			Y:  y.expr,
		},
	}
}

func (m *exprGrader) gradBinaryExpr(src *ir.BinaryExpr) (*gradExprResult, bool) {
	u := &gradExprResult{expr: src.X.Source().(ast.Expr)}
	v := &gradExprResult{expr: src.Y.Source().(ast.Expr)}
	uGrad, xOk := m.gradExpr(src.X)
	vGrad, yOk := m.gradExpr(src.Y)
	if !xOk || !yOk {
		return nil, false
	}
	switch src.Src.Op {
	case token.ADD:
		return buildAdd(uGrad, vGrad), true
	case token.SUB:
		return buildSub(uGrad, vGrad), true
	case token.MUL:
		return buildAdd(
			buildMul(uGrad, v),
			buildMul(u, vGrad),
		), true
	case token.QUO:
		return buildQuo(
			buildSub(
				buildMul(uGrad, v),
				buildMul(u, vGrad),
			),
			buildMul(v, v),
		), true
	default:
		return nil, m.fetcher.Err().Appendf(src.Source(), "gradient of binary expression %s not supported", src.Src.Op)
	}
}

func (m *exprGrader) gradParenExpr(src *ir.ParenExpr) (*gradExprResult, bool) {
	expr, ok := m.gradExpr(src.X)
	if !ok {
		return expr, false
	}
	if expr.kind != notSpecial {
		return expr, true
	}
	return &gradExprResult{
		expr: &ast.ParenExpr{X: expr.expr},
	}, true
}

func (m *exprGrader) gradValueRef(src *ir.ValueRef) (*gradExprResult, bool) {
	if m.macro.wrt.Same(src.Stor) {
		return oneValueOf(src.Source()), true
	}
	if m.macro.isParam(src.Stor) {
		return zeroValueOf(src.Source()), true
	}
	gIdent := m.macro.gradIdent(src.Stor.NameDef())
	return &gradExprResult{expr: gIdent}, true
}

func (m *exprGrader) gradArrayLitExpr(src *ir.ArrayLitExpr) (*gradExprResult, bool) {
	allZero := true
	gValues := make([]ast.Expr, len(src.Values()))
	for i, expr := range src.Values() {
		gExpr, ok := m.gradExpr(expr)
		if !ok {
			return nil, false
		}
		gValues[i] = gExpr.expr
		if gExpr.kind != zeroSpecial {
			allZero = false
			continue
		}
		gExpr = zeroValueOf(expr.Source())
		gValues[i] = gExpr.expr
	}
	if allZero {
		return zeroValueOf(src.Source()), true
	}
	return &gradExprResult{
		kind: notSpecial,
		expr: &ast.CompositeLit{
			Type: src.Typ.Source().(ast.Expr),
			Elts: gValues,
		},
	}, true
}

func (m *exprGrader) gradSelectorExpr(src *ir.SelectorExpr) (*gradExprResult, bool) {
	return zeroValueOf(src.Src), true
}

var zero = &ast.BasicLit{Value: "0", Kind: token.INT}

func zeroValueOf(node ast.Node) *gradExprResult {
	return &gradExprResult{
		kind: zeroSpecial,
		expr: zero,
	}
}

var one = &ast.BasicLit{Value: "1", Kind: token.INT}

func oneValueOf(node ast.Node) *gradExprResult {
	return &gradExprResult{
		kind: oneSpecial,
		expr: one,
	}
}
