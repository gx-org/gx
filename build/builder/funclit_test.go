package builder_test

import (
	"testing"

	"github.com/gx-org/gx/build/builder/testbuild"
)

func TestFunctionLiteral(t *testing.T) {
	testbuild.Run(t,
		testbuild.Decl{
			Src: `
func f() int32 {
	fn := func() int32 {
		return 10
	}
	return fn()
}
`,
		},
	)
}
