package builder_test

import (
	"testing"

	"github.com/gx-org/gx/build/builder/testbuild"
	"github.com/gx-org/gx/build/ir"
	irh "github.com/gx-org/gx/build/ir/irhelper"
)

func TestStaticVar(t *testing.T) {
	aVarDecl := irh.VarSpec("a")
	int32ArrayType := irh.ArrayType(
		ir.Int32Type(),
		&ir.AxisExpr{X: irh.ValueRef(aVarDecl.Exprs[0])},
	)
	xField := irh.Field("x", int32ArrayType, nil)
	testbuild.Run(t,
		testbuild.DeclTest{
			Src: `var a intlen`,
			Want: []ir.Node{
				irh.VarSpec("a"),
			},
		},
		testbuild.DeclTest{
			Src: `var a, b intlen`,
			Want: []ir.Node{
				irh.VarSpec("a", "b"),
			},
		},
		testbuild.DeclTest{
			Src: `
var a, b intlen
var c, d intlen
			`,
			Want: []ir.Node{
				irh.VarSpec("a", "b"),
				irh.VarSpec("c", "d"),
			},
		},
		testbuild.DeclTest{
			Src: `
var a intlen

func f(x [a]int32) [a]int32 {
	return x
}
			`,
			Want: []ir.Node{
				aVarDecl,
				&ir.FuncDecl{
					FType: irh.FuncType(
						nil, nil,
						irh.Fields(xField),
						irh.Fields(int32ArrayType),
					),
					Body: irh.SingleReturn(
						irh.ValueRef(xField.Storage()),
					),
				},
			},
		},
		testbuild.DeclTest{
			Src: `
var a intlen

func f() [a]int32 {
	x := [a]int32{}
	return x
}
			`,
		},
	)
}
