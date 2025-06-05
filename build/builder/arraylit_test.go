package builder_test

import (
	"go/ast"
	"testing"

	"github.com/gx-org/gx/build/ir"
	irh "github.com/gx-org/gx/build/ir/irhelper"
)

func TestArrayLit(t *testing.T) {
	testAll(t,
		irExprTest{
			src: `[2]float32{3, 4}`,
			want: &ir.ArrayLitExpr{
				Typ: irh.ArrayType(ir.Float32Type(), 2),
				Elts: []ir.AssignableExpr{
					irh.IntNumberAs(3, ir.Float32Type()),
					irh.IntNumberAs(4, ir.Float32Type()),
				},
			},
			wantType: "[2]float32",
		},
		irExprTest{
			src: `[_]float32{3, 4}`,
			want: &ir.ArrayLitExpr{
				Typ: irh.ArrayType(
					ir.Float32Type(),
					&ir.AxisInfer{
						Src: &ast.Ident{Name: "_"},
						X:   irh.Axis(2),
					},
				),
				Elts: []ir.AssignableExpr{
					irh.IntNumberAs(3, ir.Float32Type()),
					irh.IntNumberAs(4, ir.Float32Type()),
				},
			},
			wantType: "[2]float32",
		},
		irExprTest{
			src: `[2][3]float32{{1, 2, 3}, {4, 5, 6}}`,
			want: &ir.ArrayLitExpr{
				Typ: irh.ArrayType(ir.Float32Type(), 2, 3),
				Elts: []ir.AssignableExpr{
					&ir.ArrayLitExpr{
						Typ: irh.ArrayType(ir.Float32Type(), 3),
						Elts: []ir.AssignableExpr{
							irh.IntNumberAs(1, ir.Float32Type()),
							irh.IntNumberAs(2, ir.Float32Type()),
							irh.IntNumberAs(3, ir.Float32Type()),
						},
					},
					&ir.ArrayLitExpr{
						Typ: irh.ArrayType(ir.Float32Type(), 3),
						Elts: []ir.AssignableExpr{
							irh.IntNumberAs(4, ir.Float32Type()),
							irh.IntNumberAs(5, ir.Float32Type()),
							irh.IntNumberAs(6, ir.Float32Type()),
						},
					},
				},
			},
			wantType: "[2][3]float32",
		},
		irExprTest{
			src: `[2]float32{}`,
			want: &ir.ArrayLitExpr{
				Typ: irh.ArrayType(ir.Float32Type(), 2),
			},
			wantType: "[2]float32",
		},
		irExprTest{
			src: `[...]float32{}`,
			err: "cannot infer rank: empty literal",
		},
		irExprTest{
			src: `[...]int32{1, 2, 3}`,
			want: &ir.ArrayLitExpr{
				Typ: irh.InferArrayType(ir.Int32Type(), 3),
				Elts: []ir.AssignableExpr{
					irh.IntNumberAs(1, ir.Int32Type()),
					irh.IntNumberAs(2, ir.Int32Type()),
					irh.IntNumberAs(3, ir.Int32Type()),
				},
			},
			wantType: "[3]int32",
		},
		irExprTest{
			src: `[...]int32{{1, 2, 3}, {4, 5, 6}}`,
			want: &ir.ArrayLitExpr{
				Typ: irh.InferArrayType(ir.Int32Type(), 2, 3),
				Elts: []ir.AssignableExpr{
					&ir.ArrayLitExpr{
						Typ: irh.InferArrayType(ir.Int32Type(), 3),
						Elts: []ir.AssignableExpr{
							irh.IntNumberAs(1, ir.Int32Type()),
							irh.IntNumberAs(2, ir.Int32Type()),
							irh.IntNumberAs(3, ir.Int32Type()),
						},
					},
					&ir.ArrayLitExpr{
						Typ: irh.InferArrayType(ir.Int32Type(), 3),
						Elts: []ir.AssignableExpr{
							irh.IntNumberAs(4, ir.Int32Type()),
							irh.IntNumberAs(5, ir.Int32Type()),
							irh.IntNumberAs(6, ir.Int32Type()),
						},
					},
				},
			},
			wantType: "[2][3]int32",
		},
		irExprTest{
			src: `[1][1]int32{{2}}`,
			want: &ir.ArrayLitExpr{
				Typ: irh.ArrayType(ir.Int32Type(), 1, 1),
				Elts: []ir.AssignableExpr{
					&ir.ArrayLitExpr{
						Typ:  irh.ArrayType(ir.Int32Type(), 1),
						Elts: []ir.AssignableExpr{irh.IntNumberAs(2, ir.Int32Type())},
					},
				},
			},
		},
	)
}
