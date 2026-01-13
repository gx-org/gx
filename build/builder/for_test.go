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
	irh "github.com/gx-org/gx/build/ir/irhelper"
)

func TestForLoop(t *testing.T) {
	lVarDecl := irh.VarSpec("L")
	xStorage := irh.LocalVar("x", ir.Int32Type())
	xAssign := &ir.AssignExpr{
		Storage: xStorage,
		X: &ir.CastExpr{
			X:   irh.IntNumberAs(0, ir.Int32Type()),
			Typ: ir.Int32Type(),
		},
	}
	iStorage := irh.LocalVar("i", ir.IntLenType())
	testbuild.Run(t,
		testbuild.Decl{
			Src: `
var L intlen
func f() int32 {
	x := int32(0)
	for i := range L {
		x += int32(i)
	}
	return x
}
`,
			Want: []ir.IR{
				lVarDecl,
				&ir.FuncDecl{
					FType: irh.FuncType(
						nil, nil,
						irh.Fields(),
						irh.Fields(ir.Int32Type()),
					),
					Body: irh.Block(
						&ir.AssignExprStmt{List: []*ir.AssignExpr{
							xAssign,
						}},
						&ir.RangeStmt{
							Key: iStorage,
							X:   irh.Ident(lVarDecl.Exprs[0]),
							Body: irh.Block(
								&ir.AssignExprStmt{List: []*ir.AssignExpr{
									&ir.AssignExpr{
										Storage: xStorage,
										X: &ir.BinaryExpr{
											X: irh.Ident(xAssign),
											Y: &ir.CastExpr{
												X:   irh.Ident(iStorage),
												Typ: ir.Int32Type(),
											},
											Typ: ir.Int32Type(),
										},
									}}}),
						},
						&ir.ReturnStmt{Results: []ir.Expr{
							irh.Ident(xAssign),
						}},
					),
				},
			},
		},
		testbuild.Decl{
			Src: `
func f() int32 {
	a := [2]int32{2, 3}
	x := int32(0)
	for i := range a {
		x += a[i]
	}
	return x
}
`,
		},
		testbuild.Decl{
			Src: `
func f() int32 {
	a := [2]int32{2, 3}
	x := int32(0)
	for i, ai := range a {
		x += ai
	}
	return x
}
`,
		},
		testbuild.Decl{
			Src: `
func f() int64 {
	a := [3]float32{3, 5, 7}
	x := int64(100)
	for i := range a {
		x = x + i
	}
	return x
}
`,
		},
	)
}
