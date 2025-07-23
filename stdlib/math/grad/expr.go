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
	"math/big"

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
	expr ir.AssignableExpr
}

func (m *gradMacro) gradExpr(fetcher ir.Fetcher, src ir.Expr) (*gradExprResult, bool) {
	switch srcT := src.(type) {
	case *ir.ArrayLitExpr:
		return m.gradArrayLitExpr(fetcher, srcT)
	case *ir.NumberCastExpr:
		return zeroValueOf(fetcher, src.Source(), src.Type())
	case *ir.ValueRef:
		return m.gradValueRef(fetcher, srcT)
	case *ir.BinaryExpr:
		return m.gradBinaryExpr(fetcher, srcT)
	case *ir.ParenExpr:
		expr, ok := m.gradExpr(fetcher, srcT.X)
		if !ok {
			return expr, false
		}
		if expr.kind != notSpecial {
			return expr, true
		}
		return &gradExprResult{
			expr: &ir.ParenExpr{X: expr.expr},
		}, true
	case *ir.CallExpr:
		return m.gradCall(fetcher, srcT)
	default:
		return nil, fetcher.Err().Appendf(src.Source(), "gradient of %T expression not supported", srcT)
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
		expr: &ir.BinaryExpr{
			Src: &ast.BinaryExpr{Op: token.ADD},
			Typ: x.expr.Type(),
			X:   x.expr,
			Y:   y.expr,
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
		expr: &ir.BinaryExpr{
			Src: &ast.BinaryExpr{Op: token.MUL},
			Typ: x.expr.Type(),
			X:   x.expr,
			Y:   y.expr,
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
			expr: &ir.UnaryExpr{
				Src: &ast.UnaryExpr{Op: token.SUB},
				X:   y.expr,
			},
		}
	}
	return &gradExprResult{
		expr: &ir.BinaryExpr{
			Src: &ast.BinaryExpr{Op: token.SUB},
			Typ: x.expr.Type(),
			X:   x.expr,
			Y:   y.expr,
		},
	}
}

func buildQuo(x, y *gradExprResult) *gradExprResult {
	if y.kind == oneSpecial {
		return x
	}
	return &gradExprResult{
		expr: &ir.BinaryExpr{
			Src: &ast.BinaryExpr{Op: token.QUO},
			Typ: x.expr.Type(),
			X:   x.expr,
			Y:   y.expr,
		},
	}
}

func (m *gradMacro) gradBinaryExpr(fetcher ir.Fetcher, src *ir.BinaryExpr) (*gradExprResult, bool) {
	u := &gradExprResult{expr: src.X}
	v := &gradExprResult{expr: src.Y}
	uGrad, xOk := m.gradExpr(fetcher, u.expr)
	vGrad, yOk := m.gradExpr(fetcher, v.expr)
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
		return nil, fetcher.Err().Appendf(src.Source(), "gradient of binary expression %s not supported", src.Src.Op)
	}
}

func (m *gradMacro) gradValueRef(fetcher ir.Fetcher, src *ir.ValueRef) (*gradExprResult, bool) {
	if m.wrt.Same(src.Stor) {
		return oneValueOf(fetcher, src.Source(), src.Type())
	}
	gIdent := gradIdent(src.Stor.NameDef())
	gStore := &ir.LocalVarStorage{
		Src: gIdent,
		Typ: src.Type(),
	}
	return &gradExprResult{expr: &ir.ValueRef{
		Src:  gIdent,
		Stor: gStore,
	}}, true
}

func (m *gradMacro) gradArrayLitExpr(fetcher ir.Fetcher, src *ir.ArrayLitExpr) (*gradExprResult, bool) {
	allZero := true
	gValues := make([]ir.AssignableExpr, len(src.Values()))
	for i, expr := range src.Values() {
		gExpr, ok := m.gradExpr(fetcher, expr)
		if !ok {
			return nil, false
		}
		gValues[i] = gExpr.expr
		if gExpr.kind != zeroSpecial {
			allZero = false
			continue
		}
		gExpr, ok = zeroValueOf(fetcher, expr.Source(), expr.Type())
		if !ok {
			return nil, false
		}
		gValues[i] = gExpr.expr
	}
	if allZero {
		return zeroValueOf(fetcher, src.Source(), src.Type())
	}
	return &gradExprResult{
		kind: notSpecial,
		expr: src.NewFromValues(gValues),
	}, true
}

func zeroValueOf(fetcher ir.Fetcher, node ast.Node, typ ir.Type) (*gradExprResult, bool) {
	zero, ok := typ.(ir.Zeroer)
	if !ok {
		return nil, fetcher.Err().Appendf(node, "zero expression of %T not supported", typ)
	}
	return &gradExprResult{
		kind: zeroSpecial,
		expr: zero.Zero(),
	}, true
}

var one = &ir.NumberInt{Src: &ast.BasicLit{Value: "1"}, Val: big.NewInt(1)}

func oneValueOf(fetcher ir.Fetcher, node ast.Node, typ ir.Type) (*gradExprResult, bool) {
	return &gradExprResult{
		kind: oneSpecial,
		expr: &ir.CastExpr{
			Src: &ast.CallExpr{},
			X: &ir.NumberCastExpr{
				X:   one,
				Typ: ir.TypeFromKind(typ.Kind()),
			},
			Typ: typ,
		},
	}, true
}
