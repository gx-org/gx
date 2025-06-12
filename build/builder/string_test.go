package builder_test

import (
	"testing"

	"github.com/gx-org/gx/build/builder/testbuild"
)

func TestStringLiteral(t *testing.T) {
	testbuild.Run(t,
		testbuild.DeclTest{
			Src: `
func str(s string) string

func f() string {
	return str("Bonjour")
}
`,
		},
	)
}
