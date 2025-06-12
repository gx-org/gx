package builder_test

import (
	"testing"

	"github.com/gx-org/gx/build/builder/testbuild"
)

func TestCallOperators(t *testing.T) {
	testbuild.Run(t,
		testbuild.DeclTest{
			Src: `
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
	testbuild.Run(t,
		testbuild.DeclTest{
			Src: `
var A, B intlen

func f[T int32]([A][B][1]float32, []intidx) T

func fail(x [A][B][1]float32) float32 {
	return float32(f[int32](x, []intidx{0}))
}
`,
		},
	)
}
