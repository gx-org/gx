package builder_test

import (
	"testing"

	"github.com/gx-org/gx/build/builder/testbuild"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/build/ir/irhelper"
)

func TestSlice(t *testing.T) {
	testbuild.Run(t,
		testbuild.ExprTest{
			Src: `[]int32{1, 2, 3}`,
			Want: &ir.SliceLitExpr{
				Typ: irhelper.SliceType(ir.AtomTypeExpr(ir.Int32Type()), 1),
				Elts: []ir.AssignableExpr{
					irhelper.IntNumberAs(1, ir.Int32Type()),
					irhelper.IntNumberAs(2, ir.Int32Type()),
					irhelper.IntNumberAs(3, ir.Int32Type()),
				},
			},
		},
	)
}
