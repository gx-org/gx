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
	"testing"

	"github.com/gx-org/gx/build/builder/testbuild"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/build/ir/irhelper"
)

func TestSlice(t *testing.T) {
	testbuild.Run(t,
		testbuild.Expr{
			Src: `[]int32{1, 2, 3}`,
			Want: &ir.SliceLitExpr{
				Typ: irhelper.SliceType(ir.TypeExpr(nil, ir.Int32Type()), 1),
				Elts: []ir.Expr{
					irhelper.IntNumberAs(1, ir.Int32Type()),
					irhelper.IntNumberAs(2, ir.Int32Type()),
					irhelper.IntNumberAs(3, ir.Int32Type()),
				},
			},
		},
		testbuild.Decl{
			Src: `
func g(a int32) int32

func f(s []int32) int32 {
	return g(s[0])
}
`,
		},
		testbuild.Decl{
			Src: `
func f() [][2]int32 {
	a := [][2]int32{}
	for il := range 3 {
		i := int32(il)
		a = append(a, [2]int32{i, i * 2})
	}
	return a
}
`,
		},
		testbuild.Decl{
			Src: `
func f() [5][4][3][2]int32 {
	a := [5][4][3][2]int32{}
	for il := range 5 {
		i := int32(il)
		for jl := range 4 {
			j := int32(jl)
			a = set(a, [3][2]int32{}+(i+1)*(j+1), [...]int32{i, j})
		}
	}
	return a
}
`,
		},
	)
}
