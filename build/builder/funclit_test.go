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

func TestFunctionLiteralWithWant(t *testing.T) {
	funcLitType := func() *ir.FuncType {
		return irh.FuncType(
			nil, nil,
			irh.Fields(),
			irh.Fields(ir.Int32Type()),
		)
	}
	funcLitDef := &ir.FuncLit{
		FType: funcLitType(),
		Body: irh.SingleReturn(
			irh.IntNumberAs(10, ir.Int32Type()),
		),
	}
	testbuild.Run(t,
		testbuild.Decl{
			Src: `
func f() func() int32 {
	return func() int32 {
		return 10
	}
}

func g() int32 {
	fn := f()
	return fn()
}
`,
			Want: []ir.Node{
				&ir.FuncDecl{
					FType: irh.FuncType(
						nil, nil,
						irh.Fields(),
						irh.Fields(funcLitType()),
					),
					Body: irh.SingleReturn(funcLitDef),
				},
				&ir.FuncDecl{
					FType: irh.FuncType(
						nil, nil,
						irh.Fields(),
						irh.Fields(ir.Int32Type()),
					),
					Body: &ir.BlockStmt{List: []ir.Stmt{
						&ir.AssignExprStmt{List: []*ir.AssignExpr{&ir.AssignExpr{
							Storage: irh.LocalVar("fn", funcLitType()),
							X: &ir.CallExpr{
								Callee: irh.FuncDeclCallee("f"),
							},
						}}},
						&ir.ReturnStmt{Results: []ir.Expr{
							&ir.CallExpr{
								Callee: &ir.FuncValExpr{
									X: irh.ValueRef(irh.LocalVar("fn", funcLitType())),
									F: &ir.FuncLit{
										FType: funcLitType(),
									},
									T: funcLitType(),
								},
							},
						}},
					}},
				},
			},
		},
	)
}

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
		testbuild.Decl{
			Src: `
func f(x float32) (float32, func() int32) {
	fn := func() int32 {
		return 10
	}
	return x, fn
}
`,
		},
		testbuild.Decl{
			Src: `
func f(x float32) (float32, func() int32) {
	fn := func() int32 {
		return 10
	}
	return x, fn
}

func g() (float32, int32) {
	x, fn := f(10.0)
	return x, fn()
}
`,
		},
		testbuild.Decl{
			Src: `
func f(a float32) func() float32 {
	return func() float32 {
		return a
	}
}

func g() float32 {
	fn := f(10)
	a := fn()
	return a
}
`,
		},
		testbuild.Decl{
			Src: `
func f(a float32) func() float32 {
	return func() float32 {
		return a
	}
}

func g() float32 {
	fn := f(10)
	b := fn()
	return b
}
`,
		},
		testbuild.Decl{
			Src: `
func f(x float32) float32 {
	return x
}

func F(x float32) func() float32 {
	return func() float32 {
		return f(x)
	}
}
			`,
		},
		testbuild.Decl{
			Src: `
func F() int32 {
	fn := func(x, y int32) int32 {
		return x + y + 10
	}
	return fn(10, 5)
}
`,
		},
		testbuild.Decl{
			Src: `
func pair() (func() float32, func() float32) {
	f := func() float32{
		return 2
	}
	return f, f
}

func F() (float32, float32) {
	f1, f2 := pair()
	return f1(), f2()
}
`,
		},
	)
}

func TestFunctionLiteralErrors(t *testing.T) {
	testbuild.Run(t,
		testbuild.Decl{
			Src: `
func g(int32) int32

func f() int32 {
	fn := func() int32 {
		a := g()
		return a
	}
	return fn()
}
`,
			Err: "not enough arguments in call to g",
		},
	)
}
