package builder_test

import (
	"testing"

	"github.com/gx-org/gx/build/ir"
	irh "github.com/gx-org/gx/build/ir/irhelper"
)

func TestCallOperators(t *testing.T) {
	testAll(t,
		irDeclTest{
			src: `
func cast[S interface{float64}](S) uint64

func f() uint64 {
	v := cast[float64](1.0)
	return v
}
`,
		},
	)
}

func TestCallSliceArgs(t *testing.T) {
	testAll(t,
		irDeclTest{
			src: `
var A, B intlen

func f[T int32]([A][B][1]float32, []intidx) T

func fail(x [A][B][1]float32) float32 {
	return float32(f[int32](x, []intidx{0}))
}
`,
			want: []ir.Node{
				&ir.FuncDecl{
					FType: irh.CompEvalFuncType(
						irh.Fields(),
						irh.Fields(ir.Float32Type()),
					),
					Body: irh.Block(
						&ir.ReturnStmt{
							Results: []ir.Expr{
								irh.IntNumberAs(2, ir.Float32Type()),
							},
						},
					)},
			},
		},
	)
}
