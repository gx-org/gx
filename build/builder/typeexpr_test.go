package builder_test

import (
	"fmt"
	"testing"

	gxfmt "github.com/gx-org/gx/base/fmt"
	"github.com/gx-org/gx/build/builder"
)

func TestResolveType(t *testing.T) {
	tests := []struct {
		code string

		typeOk   bool
		typeName string
	}{
		{"true", true, "bool"},
		{"false", true, "bool"},
		{"123", true, "number"},
		{"123.0", true, "number"},
		{`"a"`, true, "string"},
		{"int32(1)", true, "int32"},
		{"float64(-1)", true, "float64"},

		{"[1]int32{1}", true, "[1]int32"},
		{"[_]int32{1}", true, "[1]int32"},
		{"[...]int32{1}", true, "[1]int32"},
		{"[...]int32{{1, 2}, {3, 4}}", true, "[2][2]int32"},
		{"[...]int32{1, 2, 3}[1]", true, "int32"},
		{"[...]int32{{1, 2}, {3, 4}}[1]", true, "[2]int32"},
		{"[]int32{1, 2, 3}", true, "[]int32"},

		{"struct{}{}", true, "struct"},
		{"struct{x int32}{x: 1}", true, "struct"},

		{"func() bool {}", true, "func() bool"},
		{"func() bool {}()", true, "bool"},
		{"func(int32) bool {}", true, "func(int32) bool"},
		{"func(int32) bool {}(1)", true, "bool"},
		{"func(interface{bool}) bool {}", true, "func(interface { bool }) bool"},
		{"func(interface{bfloat16|float32|float64}) bool {}", true, "func(interface { bfloat16|float32|float64 }) bool"},

		// Dynamic cast
		{"[2]int32{1, 2}.([3]int32)", true, "[3]int32"},

		// Binary ops
		{"int32(1) + int32(1)", true, "int32"},
		{"float32(1) + float32(1)", true, "float32"},
		{"[_]int32{1, 2} + int32(1)", true, "[2]int32"},
		{"int32(1) + [_]int32{1, 2}", true, "[2]int32"},
		{"float32(1) + 1", true, "float32"},
		{"intlen(1) + 1", true, "intlen"},

		{"1 > 2", true, "bool"},
		{"int32(1) > 2", true, "bool"},
		{"[_]int32{1, 2} >= 2", true, "[2]bool"},
		{"[_]int32{1, 2} >= [_]int32{1, 2}", true, "[2]bool"},
		{"2 >= [_]int32{1, 2}", true, "[2]bool"},

		// Failure cases
		{"x", false, "invalid"},
		{"x[0]", false, "invalid"},
		{"x()", false, "invalid"},
		{"x(1)", false, "invalid"},

		{"func(int32) bool {}(true)", false, "bool"},
	}

	for i, test := range tests {
		t.Run(fmt.Sprintf("Test%d", i), func(t *testing.T) {
			bld := builder.New(nil)
			pkg := bld.NewIncrementalPackage("test")
			node, err := pkg.BuildExpr(test.code)
			if err != nil {
				if test.typeOk {
					t.Errorf("test %d: %s\nunexpected error:\n%+v", i, test.code, err)
				}
				return
			}
			got := gxfmt.String(node.Type())
			if got != test.typeName {
				t.Errorf("test %d: %s\nincorrect type: got %q want %q", i, test.code, got, test.typeName)
			}
		})
	}
}
