// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package builder_test

import (
	"fmt"
	"strings"
	"testing"

	gxfmt "github.com/gx-org/gx/base/fmt"
	"github.com/gx-org/gx/build/builder"
)

func TestResolveType(t *testing.T) {
	tests := []struct {
		code string
		typ  string
		err  string
	}{
		{code: "true", typ: "bool"},
		{code: "false", typ: "bool"},
		{code: "123", typ: "number"},
		{code: "123.0", typ: "number"},
		{code: `"a"`, typ: "string"},
		{code: "int32(1)", typ: "int32"},
		{code: "float64(-1)", typ: "float64"},

		{code: "[1]int32{1}", typ: "[1]int32"},
		{code: "[_]int32{1}", typ: "[1]int32"},
		{code: "[...]int32{1}", typ: "[1]int32"},
		{code: "[...]int32{{1, 2}, {3, 4}}", typ: "[2][2]int32"},
		{code: "[...]int32{1, 2, 3}[1]", typ: "int32"},
		{code: "[...]int32{{1, 2}, {3, 4}}[1]", typ: "[2]int32"},
		{code: "[]int32{1, 2, 3}", typ: "[]int32"},
		{code: `[...]float64{
			[...]float64{0, 1, 2, 3},
			[...]float64{10, 11, 12, 13},
			[...]float64{20, 21, 22, 23},
			[...]float64{30, 31, 32, 33},
		}`, typ: "[4][4]float64"},

		{code: "struct{}{}", typ: "struct"},
		{code: "struct{x int32}{x: 1}", typ: "struct"},

		{code: "func() bool {}", typ: "func() bool"},
		{code: "func() bool {}()", typ: "bool"},
		{code: "func(int32) bool {}", typ: "func(int32) bool"},
		{code: "func(int32) bool {}(1)", typ: "bool"},
		{code: "func(interface{bool}) bool {}", typ: "func(interface { bool }) bool"},
		{code: "func(interface{bfloat16|float32|float64}) bool {}", typ: "func(interface { bfloat16|float32|float64 }) bool"},

		// Dynamic cast
		{code: "[2]int32{1, 2}.([3]int32)", typ: "[3]int32"},

		// Binary ops
		{code: "int32(1) + int32(1)", typ: "int32"},
		{code: "float32(1) + float32(1)", typ: "float32"},
		{code: "[_]int32{1, 2} + int32(1)", typ: "[2]int32"},
		{code: "int32(1) + [_]int32{1, 2}", typ: "[2]int32"},
		{code: "float32(1) + 1", typ: "float32"},
		{code: "intlen(1) + 1", typ: "intlen"},

		{code: "1 > 2", typ: "bool"},
		{code: "int32(1) > 2", typ: "bool"},
		{code: "[_]int32{1, 2} >= 2", typ: "[2]bool"},
		{code: "[_]int32{1, 2} >= [_]int32{1, 2}", typ: "[2]bool"},
		{code: "2 >= [_]int32{1, 2}", typ: "[2]bool"},

		// Failure cases
		{code: "x", err: "undefined: x"},
		{code: "x[0]", err: "undefined: x"},
		{code: "x()", err: "undefined: x"},
		{code: "x(1)", err: "undefined: x"},
		{code: "[2]float", err: "undefined: float"},
		{code: "[2][3]float32{{1, 2, 3}}", err: "cannot assign"},
		{code: "func(int32) bool {}(true)", err: "cannot use bool as int32"},
	}

	for i, test := range tests {
		t.Run(fmt.Sprintf("Test%d", i), func(t *testing.T) {
			bld := builder.New(nil)
			pkg := bld.NewIncrementalPackage("test")
			node, err := pkg.BuildExpr(test.code)
			if test.typ == "" {
				if err == nil {
					t.Error("expected an error but got nil")
					return
				}
				// Check the error returned by the expression
				got := err.Error()
				if got == "" {
					got = "\n<no error>"
				}
				if !strings.Contains(got, test.err) {
					t.Errorf("got error:%s\nbut want an error containing %s", got, test.err)
					return
				}
				return
			}

			// Check the type of the expression.
			if err != nil {
				t.Errorf("unexpected error: %+v", err)
				return
			}
			got := gxfmt.String(node.Type())
			if got != test.typ {
				t.Errorf("test %d: %s\nincorrect type: got %q want %q", i, test.code, got, test.typ)
			}

		})
	}
}
