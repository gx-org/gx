package builder_test

import (
	"go/ast"
	"testing"

	"github.com/gx-org/gx/build/builder/testbuild"
	"github.com/gx-org/gx/build/ir"
	irh "github.com/gx-org/gx/build/ir/irhelper"
)

func TestArrayLit(t *testing.T) {
	testbuild.Run(t,
		testbuild.ExprTest{
			Src: `[2]float32{3, 4}`,
			Want: &ir.ArrayLitExpr{
				Typ: irh.ArrayType(ir.Float32Type(), 2),
				Elts: []ir.AssignableExpr{
					irh.IntNumberAs(3, ir.Float32Type()),
					irh.IntNumberAs(4, ir.Float32Type()),
				},
			},
			WantType: "[2]float32",
		},
		testbuild.ExprTest{
			Src: `[_]float32{3, 4}`,
			Want: &ir.ArrayLitExpr{
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
			WantType: "[2]float32",
		},
		testbuild.ExprTest{
			Src: `[2][3]float32{{1, 2, 3}, {4, 5, 6}}`,
			Want: &ir.ArrayLitExpr{
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
			WantType: "[2][3]float32",
		},
		testbuild.ExprTest{
			Src: `[2]float32{}`,
			Want: &ir.ArrayLitExpr{
				Typ: irh.ArrayType(ir.Float32Type(), 2),
			},
			WantType: "[2]float32",
		},
		testbuild.ExprTest{
			Src: `[...]float32{}`,
			Err: "cannot infer rank: empty literal",
		},
		testbuild.ExprTest{
			Src: `[...]int32{1, 2, 3}`,
			Want: &ir.ArrayLitExpr{
				Typ: irh.InferArrayType(ir.Int32Type(), 3),
				Elts: []ir.AssignableExpr{
					irh.IntNumberAs(1, ir.Int32Type()),
					irh.IntNumberAs(2, ir.Int32Type()),
					irh.IntNumberAs(3, ir.Int32Type()),
				},
			},
			WantType: "[3]int32",
		},
		testbuild.ExprTest{
			Src: `[...]int32{{1, 2, 3}, {4, 5, 6}}`,
			Want: &ir.ArrayLitExpr{
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
			WantType: "[2][3]int32",
		},
		testbuild.ExprTest{
			Src: `[1][1]int32{{2}}`,
			Want: &ir.ArrayLitExpr{
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
