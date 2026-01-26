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
		Slice: &ir.SliceType{
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
func f(a ...intlen) [a]int32
`,
		},
	)
}

func TestVarArgsErrors(t *testing.T) {
	testbuild.Run(t,
		testbuild.Decl{
			Src: `
func f(a,b ...int32) int32
`,
			Err: "can only use ... with final parameter",
		},
		testbuild.Decl{
			Src: `
func f(a ...int32, b ...int32) int32
`,
			Err: "can only use ... with final parameter",
		},
	)
}
