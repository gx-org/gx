package builder_test

import (
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
			want: []ir.Node{
				aVarDecl,
			},
		},
	)
}
