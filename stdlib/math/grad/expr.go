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
	x    ir.AssignableExpr
}

func gradExpr(fetcher ir.Fetcher, src ir.Expr, argName string) (*gradExprResult, bool) {
	switch srcT := src.(type) {
	case *ir.ArrayLitExpr:
		return gradArrayLitExpr(fetcher, srcT, argName)
	case *ir.NumberCastExpr:
		return zeroValueOf(fetcher, src.Source(), src.Type())
	case *ir.ValueRef:
		return gradValueRef(fetcher, srcT, argName)
	case *ir.BinaryExpr:
		return gradBinaryExpr(fetcher, srcT, argName)
	default:
		return nil, fetcher.Err().Appendf(src.Source(), "gradient of %T expression not supported", srcT)
	}
}

func gradBinaryExpr(fetcher ir.Fetcher, src *ir.BinaryExpr, argName string) (*gradExprResult, bool) {
	u := src.X
	v := src.Y
	uGrad, xOk := gradExpr(fetcher, u, argName)
	vGrad, yOk := gradExpr(fetcher, v, argName)
	switch src.Src.Op {
	case token.ADD, token.SUB:
		return &gradExprResult{
			x: &ir.BinaryExpr{
				Src: src.Src,
				Typ: src.Typ,
				X:   uGrad.x,
				Y:   vGrad.x,
			},
		}, xOk && yOk
	case token.MUL:
		return &gradExprResult{
			x: &ir.BinaryExpr{
				Src: &ast.BinaryExpr{Op: token.ADD},
				X: &ir.BinaryExpr{
					Src: &ast.BinaryExpr{Op: token.MUL},
					X:   uGrad.x,
					Y:   v,
				},
				Y: &ir.BinaryExpr{
					Src: &ast.BinaryExpr{Op: token.MUL},
					X:   u,
					Y:   vGrad.x,
				},
			},
		}, xOk && yOk
	default:
		return nil, fetcher.Err().Appendf(src.Source(), "gradient of binary expression %s not supported", src.Src.Op)
	}
}

func gradValueRef(fetcher ir.Fetcher, src *ir.ValueRef, argName string) (*gradExprResult, bool) {
	if src.Src.Name != argName {
		// The ident does not correspond to the variable
		// for which we are differentiating: return zero.
		return nil, true
	}
	return oneValueOf(fetcher, src.Source(), src.Type())
}

func gradArrayLitExpr(fetcher ir.Fetcher, src *ir.ArrayLitExpr, argName string) (*gradExprResult, bool) {
	allZero := true
	gValues := make([]ir.AssignableExpr, len(src.Values()))
	for i, expr := range src.Values() {
		gExpr, ok := gradExpr(fetcher, expr, argName)
		if !ok {
			return nil, false
		}
		gValues[i] = gExpr.x
		if gExpr.kind != zeroSpecial {
			allZero = false
			continue
		}
		gExpr, ok = zeroValueOf(fetcher, expr.Source(), expr.Type())
		if !ok {
			return nil, false
		}
		gValues[i] = gExpr.x
	}
	if allZero {
		return zeroValueOf(fetcher, src.Source(), src.Type())
	}
	return &gradExprResult{
		kind: notSpecial,
		x:    src.NewFromValues(gValues),
	}, true
}

func zeroValueOf(fetcher ir.Fetcher, node ast.Node, typ ir.Type) (*gradExprResult, bool) {
	zero, ok := typ.(ir.Zeroer)
	if !ok {
		return nil, fetcher.Err().Appendf(node, "zero expression of %T not supported", typ)
	}
	return &gradExprResult{
		kind: zeroSpecial,
		x:    zero.Zero(),
	}, true
}

var one = &ir.NumberInt{Src: &ast.BasicLit{Value: "1"}, Val: big.NewInt(1)}

func oneValueOf(fetcher ir.Fetcher, node ast.Node, typ ir.Type) (*gradExprResult, bool) {
	return &gradExprResult{
		kind: oneSpecial,
		x: &ir.CastExpr{
			X: &ir.NumberCastExpr{
				X:   one,
				Typ: ir.TypeFromKind(typ.Kind()),
			},
			Typ: typ,
		},
	}, true
}
