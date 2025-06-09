package builder_test

import (
	"go/ast"
	"testing"

	"github.com/gx-org/gx/build/ir"
	irh "github.com/gx-org/gx/build/ir/irhelper"
)

func TestCast(t *testing.T) {
	testAll(t,
		irExprTest{
			src: `[2]float32([2]int64{3, 4})`,
			want: &ir.CastExpr{
				X: &ir.ArrayLitExpr{
					Typ: irh.ArrayType(ir.Int64Type(), 2),
					Elts: []ir.AssignableExpr{
						irh.IntNumberAs(3, ir.Int64Type()),
						irh.IntNumberAs(4, ir.Int64Type()),
					},
				},
				Typ: irh.ArrayType(ir.Float32Type(), 2),
			},
			wantType: "[2]float32",
		},
		irExprTest{
			src: `([2]float32)([2]int64{3, 4})`,
			want: &ir.CastExpr{
				X: &ir.ArrayLitExpr{
					Typ: irh.ArrayType(ir.Int64Type(), 2),
					Elts: []ir.AssignableExpr{
						irh.IntNumberAs(3, ir.Int64Type()),
						irh.IntNumberAs(4, ir.Int64Type()),
					},
				},
				Typ: irh.ArrayType(ir.Float32Type(), 2),
			},
			wantType: "[2]float32",
		},
		irExprTest{
			src: `[___]float32([2]int64{3, 4})`,
			want: &ir.CastExpr{
				X: &ir.ArrayLitExpr{
					Typ: irh.ArrayType(ir.Int64Type(), 2),
					Elts: []ir.AssignableExpr{
						irh.IntNumberAs(3, ir.Int64Type()),
						irh.IntNumberAs(4, ir.Int64Type()),
					},
				},
				Typ: irh.ArrayType(ir.Float32Type(), &ir.RankInfer{
					Rnk: &ir.Rank{Ax: []ir.AxisLengths{irh.Axis(2)}},
				}),
			},
			wantType: "[2]float32",
		},
	)
}

func TestCastStaticVar(t *testing.T) {
	aVarDecl := irh.VarSpec("a")
	xFunc := &ir.FuncBuiltin{
		Src: &ast.FuncDecl{Name: &ast.Ident{Name: "x"}},
		FType: irh.FuncType(
			nil, nil,
			irh.Fields(ir.Float32Type()),
			irh.Fields(ir.Float32Type()),
		),
	}
	testAll(t,
		irDeclTest{
			src: `
var a intlen

func x(float32) float32

func f() float32 {
	return x(float32(a))
}
`,
			want: []ir.Node{
				aVarDecl,
				xFunc,
				&ir.FuncDecl{
					Src: &ast.FuncDecl{Name: &ast.Ident{Name: "f"}},
					FType: irh.FuncType(
						nil, nil,
						irh.Fields(),
						irh.Fields(ir.Float32Type()),
					),
					Body: irh.SingleReturn(&ir.CallExpr{
						Args: []ir.AssignableExpr{&ir.CastExpr{
							X:   irh.ValueRef(aVarDecl.Exprs[0]),
							Typ: ir.Float32Type(),
						}},
						Callee: &ir.FuncValExpr{
							X: xFunc,
							F: xFunc,
							T: xFunc.FuncType(),
						},
					}),
				},
			},
		},
		irDeclTest{
			src: `
var a intlen

func newArray() [a]int32
func id([a]float32) float32

func f() float32 {
	return id(([a]float32)(newArray()))
}
`,
		},
	)
}
