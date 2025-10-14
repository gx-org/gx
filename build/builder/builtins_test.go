package builder_test

import (
	"testing"

	"github.com/gx-org/gx/build/builder/testbuild"
)

func TestBuiltinFuncs(t *testing.T) {
	testbuild.Run(t,
		testbuild.Decl{
			Src: `
func f() [2][2]int32 {
	a := [2][2]int32{}
	a = set(a, [2]int32{1, 2}, [1]int32{0})
	a = set(a, [2]int32{3, 4}, [1]int32{1})
	return a
}
`,
		},
	)
}
