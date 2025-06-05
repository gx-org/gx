package builder_test

import (
	"testing"

	"github.com/gx-org/gx/build/ir"
	irh "github.com/gx-org/gx/build/ir/irhelper"
)

func TestIndex(t *testing.T) {
	testAll(t,
		irExprTest{
			src: `[2]float32{3, 4}[1]`,
			want: &ir.IndexExpr{
				X: &ir.ArrayLitExpr{
					Typ: irh.ArrayType(ir.Float32Type(), 2),
					Elts: []ir.AssignableExpr{
						irh.IntNumberAs(3, ir.Float32Type()),
						irh.IntNumberAs(4, ir.Float32Type()),
					},
				},
				Index: irh.IntNumberAs(1, ir.Int64Type()),
				Typ:   ir.Float32Type(),
			},
			wantType: "float32",
		},
	)
}
