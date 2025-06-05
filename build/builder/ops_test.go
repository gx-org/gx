package builder_test

import (
	"testing"

	"github.com/gx-org/gx/build/ir"
	irh "github.com/gx-org/gx/build/ir/irhelper"
)

func TestBinaryOp(t *testing.T) {
	testAll(t,
		irExprTest{
			src: `[2]int64{3, 4} > 2`,
			want: &ir.BinaryExpr{
				X: &ir.ArrayLitExpr{
					Typ: irh.ArrayType(ir.Int64Type(), 2),
					Elts: []ir.AssignableExpr{
						irh.IntNumberAs(3, ir.Int64Type()),
						irh.IntNumberAs(4, ir.Int64Type()),
					},
				},
				Y:   irh.IntNumberAs(2, ir.Int64Type()),
				Typ: irh.ArrayType(ir.BoolType(), 2),
			},
			wantType: "[2]bool",
		},
	)
}
