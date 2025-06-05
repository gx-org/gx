package builder_test

import (
	"testing"

	"github.com/gx-org/gx/build/ir"
	irh "github.com/gx-org/gx/build/ir/irhelper"
)

func TestForLoop(t *testing.T) {
	testAll(t,
		irDeclTest{
			src: `
var L intlen
func f() int32 {
	x := int32(0)
	for i := range L {
		x += int32(i)
	}
	return x
}
`,
			want: []ir.Node{
				&ir.FuncDecl{
					FType: irh.FuncType(
						nil, nil,
						irh.Fields(),
						irh.Fields(ir.Int32Type()),
					),
				},
			},
		},
		irDeclTest{
			src: `
func f() int32 {
	a := [2]int32{2, 3}
	x := int32(0)
	for i := range a {
		x += a[i]
	}
	return x
}
`,
			want: []ir.Node{
				&ir.FuncDecl{
					FType: irh.FuncType(
						nil, nil,
						irh.Fields(),
						irh.Fields(ir.Int32Type()),
					),
				},
			},
		},
		irDeclTest{
			src: `
func f() int32 {
	a := [2]int32{2, 3}
	x := int32(0)
	for i, ai := range a {
		x += ai
	}
	return x
}
`,
			want: []ir.Node{
				&ir.FuncDecl{
					FType: irh.FuncType(
						nil, nil,
						irh.Fields(),
						irh.Fields(ir.Int32Type()),
					),
				},
			},
		},
	)
}
