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
	"go/ast"
	"testing"

	"github.com/gx-org/gx/build/builder/testbuild"
	"github.com/gx-org/gx/build/ir"
	irh "github.com/gx-org/gx/build/ir/irhelper"
)

func TestVarArgs(t *testing.T) {
	varargsType := &ir.VarArgsType{
		Typ: &ir.SliceType{
			BaseType: ir.BaseType[ast.Expr]{Src: &ast.Ident{}},
			DType:    ir.TypeExpr(nil, ir.Int32Type()),
			Rank:     1,
		},
	}
	field := irh.Field("a", varargsType, nil)
	ftype := irh.FuncType(nil, nil,
		irh.Fields(field),
		irh.Fields(ir.Int32Type()),
	)
	ftype.VarArgs = varargsType
	testbuild.Run(t,
		testbuild.Decl{
			Src: `
func f(a ...int32) int32
`,
			Want: []ir.IR{
				&ir.FuncBuiltin{
					FType: ftype,
				},
			},
		},
		testbuild.Decl{
			Src: `
func f(a ...int32) int32

func g() int32 {
	return f()
}
`,
		},
		testbuild.Decl{
			Src: `
func f(a ...int32) int32

func g() int32 {
	return f(1)
}
`,
		},
		testbuild.Decl{
			Src: `
func f(a ...int32) int32

func g() int32 {
	return f(1, 2, 3)
}
`,
		},
		testbuild.Decl{
			Src: `
func f(...int32) []int32

func g() []int32 {
	a := []int32{1, 2, 3}
	return f(a...)
}
`,
		},
		testbuild.Decl{
			Src: `
func f(...int32) []int32

func g(a ...int32) []int32 {
	return f(a...)
}
`,
		},
		testbuild.Decl{
			Src: `
func f(...int32) []int32

func g(a ...int32) []int32 {
	return f(f(a...)...)
}
`,
		},
		testbuild.Decl{
			Src: `
func f(...int32) []int32

func g() []int32 {
	return f()
}
`,
		},
		testbuild.Decl{
			Src: `
func f(string, ...int32) []int32

func g() []int32 {
	return f("a")
}
`,
		},
		testbuild.Decl{
			Src: `
func f1(string, ...int32) []int32

func g() []int32 {
	return f1("a")
}
`,
		},
		testbuild.Decl{
			Src: `
func f1(string, ...int32) []int32

func f2([]int32) []int32

func g() []int32 {
	return f2(f1("a"))
}
`,
		},
	)
}

func TestGenericVarArgs(t *testing.T) {
	testbuild.Run(t,
		testbuild.Decl{
			Src: `
func g[T any](a ...T) T

func f() int32 {
	return g(int32(1), int32(2))
}
`,
		},
		testbuild.Decl{
			Src: `
func g[T any](a ...T) T

func f() int32 {
	return g(int32(1), int64(2)) // ERROR type int64 does not match type int32 for T
}
`,
		},
		testbuild.Decl{
			Src: `
func g[shapes [][]intlen](a ...[unpack(varargsindex(shapes))]int32) int32

func f() int32 {
	return g([2][3]int32{}, [4][5]int32{})
}
`,
		},
		testbuild.Decl{
			Src: `
type ints interface {
	int32 | int64
}

func g[Axis intidx, Shapes [][]intlen, DType ints](x ...[unpack(varargsindex(Shapes))]DType) DType

func f() int32 {
	return g[0]([2][3]int32{}, [4][5]int32{})
}
`,
		},
		testbuild.Decl{
			Src: `
func Concat[Lens [][]intlen](x ...[unpack(varargsindex(Lens))]float32) [unpack(Lens[0])]float32

func f[Size intlen](x [Size]float32) [Size]float32 {
	return Concat(x, [1]float32{})
}
`,
		},
	)
}

func TestVarArgsErrors(t *testing.T) {
	testbuild.Run(t,
		testbuild.Decl{
			Src: `
func f(a,b ...int32) int32 // ERROR can only use ... with final parameter
`,
		},
		testbuild.Decl{
			Src: `
func f(a ...int32, b ...int32) int32 // ERROR can only use ... with final parameter
`,
		},
		testbuild.Decl{
			Src: `
func f(...int32) []int32

func g() []int32 {
	a := []int32{1, 2, 3}
	return f(a) // ERROR cannot use type []int32 as int32 in argument to f
}
`,
		},
		testbuild.Decl{
			Src: `
func f(...int32) []int32

func g() []int32 {
	a := int32(1)
	return f(a...) // ERROR cannot unpack type int32
}
`,
		},
	)
}
