package builder_test

import (
	"testing"

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
	testAll(t,
		irDeclTest{
			src: `var a intlen`,
			want: []ir.Node{
				irh.VarSpec("a"),
			},
		},
		irDeclTest{
			src: `var a, b intlen`,
			want: []ir.Node{
				irh.VarSpec("a", "b"),
			},
		},
		irDeclTest{
			src: `
var a, b intlen
var c, d intlen
			`,
			want: []ir.Node{
				irh.VarSpec("a", "b"),
				irh.VarSpec("c", "d"),
			},
		},
		irDeclTest{
			src: `
var a intlen

func f(x [a]int32) [a]int32 {
	return x
}
			`,
			want: []ir.Node{
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
		irDeclTest{
			src: `
var a intlen

func f() [a]int32 {
	x := [a]int32{}
	return x
}
			`,
			want: []ir.Node{
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
	)
}
