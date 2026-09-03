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
	iStorage := irh.LocalVar("i", ir.IntType())
	testbuild.Run(t,
		testbuild.Decl{
			Src: `
var L int
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
	)
}

func TestUnrollForLoop(t *testing.T) {
	xStorage := irh.LocalVar("x", ir.Int32Type())
	xAssign := &ir.AssignExpr{
		Storage: xStorage,
		X: &ir.CastExpr{
			X:   irh.IntNumberAs(0, ir.Int32Type()),
			Typ: ir.Int32Type(),
		},
	}
	iStorage := irh.LocalVar("i", ir.IntType())
	vStorage := irh.LocalVar("v", ir.Int32Type())
	sliceType := &ir.SliceType{
		BaseType: ir.BaseType[ast.Expr]{Src: &ast.ArrayType{}},
		DType:    ir.TypeExpr(nil, ir.Int32Type()),
		Rank:     1,
	}
	listFunc := &ir.FuncDecl{
		FType: irh.UnrollFuncType(
			irh.Fields(),
			irh.Fields(sliceType),
		),
		Body: irh.Block(
			&ir.ReturnStmt{Results: []ir.Expr{
				&ir.SliceLitExpr{
					Typ: sliceType,
					Elts: []ir.Expr{
						irh.IntNumberAs(7, ir.Int32Type()),
						irh.IntNumberAs(8, ir.Int32Type()),
					},
				},
			}},
		),
	}
	testbuild.Run(t,
		testbuild.Decl{
			Src: `
//gx:unroll
func list() []int32 {
	return []int32{7, 8}
}

func f() int32 {
	x := int32(0)
	for i := range list() {
		x += int32(i)
	}
	return x
}
`,
			Want: []ir.IR{
				listFunc,
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
						&ir.UnrollStmt{
							Range: &ir.RangeStmt{
								Key: iStorage,
								X: &ir.FuncCallExpr{
									Callee: irh.FuncDeclCallee("list", listFunc.FType),
								},
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
							Source: `{
	x += int32(0)
}
{
	x += int32(1)
}
`,
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
//gx:unroll
func list() []int32 {
	return []int32{7, 8}
}

func f() int32 {
	x := int32(0)
	for _, v := range list() {
		x += v
	}
	return x
}
`,
			Want: []ir.IR{
				listFunc,
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
						&ir.UnrollStmt{
							Range: &ir.RangeStmt{
								Key:   irh.LocalVar("_", ir.IntType()),
								Value: vStorage,
								X: &ir.FuncCallExpr{
									Callee: irh.FuncDeclCallee("list", listFunc.FType),
								},
								Body: irh.Block(
									&ir.AssignExprStmt{List: []*ir.AssignExpr{
										&ir.AssignExpr{
											Storage: xStorage,
											X: &ir.BinaryExpr{
												X:   irh.Ident(xAssign),
												Y:   irh.Ident(vStorage),
												Typ: ir.Int32Type(),
											},
										}}}),
							},
							Source: `{
	x += 7
}
{
	x += 8
}
`,
						},
						&ir.ReturnStmt{Results: []ir.Expr{
							irh.Ident(xAssign),
						}},
					),
				},
			},
		},
	)
}
