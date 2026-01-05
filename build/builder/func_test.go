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

func TestCompEval(t *testing.T) {
	testbuild.Run(t,
		testbuild.Decl{
			Src: `
//gx:compeval
func returnTwo() float32 {
	return 2
}
`,
			Want: []ir.IR{
				&ir.FuncDecl{
					FType: irh.CompEvalFuncType(
						irh.Fields(),
						irh.Fields(ir.Float32Type()),
					),
					Body: irh.Block(
						&ir.ReturnStmt{
							Results: []ir.Expr{
								irh.IntNumberAs(2, ir.Float32Type()),
							},
						},
					)},
			},
		},
	)
}

func TestBuiltin(t *testing.T) {
	testbuild.Run(t,
		testbuild.Decl{
			Src: `func returnTwo() float32`,
			Want: []ir.IR{
				&ir.FuncBuiltin{
					FType: irh.FuncType(
						nil, nil,
						irh.Fields(),
						irh.Fields(ir.Float32Type()),
					),
				},
			},
		},
		testbuild.Decl{
			Src: `func returnTwo(x int32) x`,
			Err: "x undefined",
		},
	)
}

func TestBuiltinMethods(t *testing.T) {
	typeA := &ir.NamedType{
		File:       wantFile,
		Src:        &ast.TypeSpec{Name: irh.Ident("A")},
		Underlying: ir.TypeExpr(nil, ir.Uint32Type()),
	}
	funF := &ir.FuncBuiltin{
		FType: irh.FuncType(
			nil,
			irh.Fields(typeA),
			irh.Fields(ir.Uint32Type()),
			irh.Fields(ir.Uint32Type()),
		),
	}
	typeA.Methods = []ir.PkgFunc{funF}
	testbuild.Run(t,
		testbuild.Decl{
			Src: `
type A uint32
func (A) F(uint32) uint32
`,
			Want: []ir.IR{
				typeA,
			},
		},
	)
}

func TestFuncDecl(t *testing.T) {
	aField := irh.Field("a", ir.Int32Type(), nil)
	returnTwoFunc := &ir.FuncDecl{
		Src: &ast.FuncDecl{Name: &ast.Ident{Name: "returnTwo"}},
		FType: irh.FuncType(
			nil, nil,
			irh.Fields(),
			irh.Fields(ir.Float32Type()),
		),
		Body: irh.Block(
			&ir.ReturnStmt{
				Results: []ir.Expr{
					irh.IntNumberAs(2, ir.Float32Type()),
				},
			},
		)}
	testbuild.Run(t,
		testbuild.Decl{
			Src: `
func returnTwo() float32 {
	return 2
}
`,
			Want: []ir.IR{returnTwoFunc},
		},
		testbuild.Decl{
			Src: `
func returnTwo() float32 {
	return 2
}

func call() float32 {
	return returnTwo()
}
`,
			Want: []ir.IR{
				returnTwoFunc,
				&ir.FuncDecl{
					FType: irh.FuncType(nil, nil, irh.Fields(), irh.Fields(ir.Float32Type())),
					Body: irh.Block(
						&ir.ReturnStmt{
							Results: []ir.Expr{&ir.FuncCallExpr{
								Callee: irh.FuncExpr(returnTwoFunc),
							}},
						},
					),
				},
			},
		},
		testbuild.Decl{
			Src: `
func withArgs(a int32) int32 {
	return a
}
`,
			Want: []ir.IR{
				&ir.FuncDecl{
					FType: irh.FuncType(
						nil, nil,
						irh.Fields(aField),
						irh.Fields(ir.Int32Type()),
					),
					Body: irh.Block(
						&ir.ReturnStmt{
							Results: []ir.Expr{
								irh.ValueRef(aField.Storage()),
							},
						},
					)},
			},
		},
		testbuild.Decl{
			Src: `
func namedReturn() (a int32) {
	a = 2
	return
}
`,
			Want: []ir.IR{
				&ir.FuncDecl{
					FType: irh.FuncType(
						nil, nil, irh.Fields(),
						irh.Fields("a", ir.Int32Type()),
					),
					Body: irh.Block(
						&ir.AssignExprStmt{List: []*ir.AssignExpr{
							{
								Storage: irh.Fields("a", ir.Int32Type()).Fields()[0].Storage(),
								X:       irh.IntNumberAs(2, ir.Int32Type()),
							},
						}},
						&ir.ReturnStmt{Src: &ast.ReturnStmt{}},
					)},
			},
		},
	)
	returnTupleFunc := &ir.FuncDecl{
		Src: &ast.FuncDecl{Name: &ast.Ident{Name: "returnTuple"}},
		FType: irh.FuncType(
			nil, nil,
			irh.Fields(),
			irh.Fields(ir.Float32Type(), ir.Int32Type()),
		),
		Body: irh.Block(
			&ir.ReturnStmt{
				Results: []ir.Expr{
					irh.IntNumberAs(2, ir.Float32Type()),
					irh.IntNumberAs(3, ir.Int32Type()),
				},
			},
		)}
	testbuild.Run(t,
		testbuild.Decl{
			Src: `
func returnTuple() (float32, int32) {
	return 2, 3
}

func call() (float32, int32) {
	return returnTuple()
}
`,
			Want: []ir.IR{
				returnTupleFunc,
				&ir.FuncDecl{
					FType: irh.FuncType(
						nil, nil,
						irh.Fields(),
						irh.Fields(ir.Float32Type(), ir.Int32Type()),
					),
					Body: irh.Block(
						&ir.ReturnStmt{
							Results: []ir.Expr{&ir.FuncCallExpr{
								Callee: irh.FuncExpr(returnTupleFunc),
							}},
						},
					),
				},
			},
		},
	)
}

func TestCallWithLiterals(t *testing.T) {
	oneByOneLiteral := &ir.ArrayLitExpr{
		Typ: irh.ArrayType(ir.Float32Type(), 1, 1),
		Elts: []ir.Expr{
			&ir.ArrayLitExpr{
				Typ:  irh.ArrayType(ir.Float32Type(), 1),
				Elts: []ir.Expr{irh.IntNumberAs(2, ir.Float32Type())},
			},
		},
	}
	fDef := &ir.FuncBuiltin{
		Src: &ast.FuncDecl{Name: &ast.Ident{Name: "f"}},
		FType: irh.FuncType(
			nil, nil,
			irh.Fields(oneByOneLiteral.Typ),
			irh.Fields(ir.Float32Type()),
		),
	}
	testbuild.Run(t,
		testbuild.Decl{
			Src: `
func f([1][1]float32) float32

func call() float32 {
	return f([1][1]float32{{2}})
}
`,
			Want: []ir.IR{
				fDef,
				&ir.FuncDecl{
					FType: irh.FuncType(nil, nil, irh.Fields(), irh.Fields(ir.Float32Type())),
					Body: irh.Block(
						&ir.ReturnStmt{
							Results: []ir.Expr{&ir.FuncCallExpr{
								Callee: irh.FuncExpr(fDef),
								Args:   []ir.Expr{oneByOneLiteral},
							}},
						},
					),
				},
			},
		},
	)
}

func TestUndefinedCallee(t *testing.T) {
	testbuild.Run(t,
		testbuild.Decl{
			Src: `
func f() float32 {
	return g()()
}
`,
			Err: "undefined: g",
		},
		testbuild.Decl{
			Src: `
func f() float32 {
	a := g()()
	return a
}
`,
			Err: "undefined: g",
		},
	)

}
