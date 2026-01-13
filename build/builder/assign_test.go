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

func TestAssign(t *testing.T) {
	aStorage := &ir.AssignExpr{
		Storage: irh.LocalVar("a", ir.Float32Type()),
		X: &ir.CastExpr{
			X:   irh.IntNumberAs(2, ir.Float32Type()),
			Typ: ir.Float32Type(),
		},
	}
	bStorage := &ir.AssignExpr{
		Storage: irh.LocalVar("b", ir.Float32Type()),
		X: &ir.CastExpr{
			X:   irh.IntNumberAs(3, ir.Float32Type()),
			Typ: ir.Float32Type(),
		},
	}
	cStorage := irh.LocalVar("c", ir.Float32Type())
	dStorage := irh.LocalVar("d", ir.Float32Type())
	assign := &ir.FuncDecl{
		Src: &ast.FuncDecl{Name: irh.IdentAST("assign")},
		FType: irh.FuncType(
			nil, nil,
			irh.Fields(),
			irh.Fields(ir.Float32Type(), ir.Float32Type()),
		),
		Body: irh.Block(
			&ir.AssignExprStmt{List: []*ir.AssignExpr{
				aStorage, bStorage,
			}},
			&ir.ReturnStmt{
				Results: []ir.Expr{
					irh.Ident(aStorage),
					irh.Ident(bStorage),
				},
			},
		)}
	callToAssign := &ir.FuncCallExpr{
		Callee: irh.FuncExpr(assign),
	}
	cAssignment := &ir.AssignCallResult{
		Storage:     cStorage,
		Call:        callToAssign,
		ResultIndex: 0,
	}
	dAssignment := &ir.AssignCallResult{
		Storage:     dStorage,
		Call:        callToAssign,
		ResultIndex: 1,
	}
	testbuild.Run(t,
		testbuild.Decl{
			Src: `
func assign() (float32, float32) {
	a, b := float32(2), float32(3)
	return a, b
}
`,
			Want: []ir.IR{assign},
		},
		testbuild.Decl{
			Src: `
func assign() (float32, float32) {
	a, b := float32(2), float32(3)
	return a, b
}

func callAssign() (float32, float32) {
	c, d := assign()
	return c, d
}
`,
			Want: []ir.IR{
				assign,
				&ir.FuncDecl{
					FType: irh.FuncType(
						nil, nil,
						irh.Fields(),
						irh.Fields(ir.Float32Type(), ir.Float32Type()),
					),
					Body: irh.Block(
						&ir.AssignCallStmt{
							List: []*ir.AssignCallResult{
								cAssignment,
								dAssignment,
							},
							Call: callToAssign,
						},
						&ir.ReturnStmt{
							Results: []ir.Expr{
								irh.Ident(cAssignment),
								irh.Ident(dAssignment),
							},
						},
					)},
			},
		},
		testbuild.Decl{
			Src: `
func id(int64) int64

func f() int64 {
	a, b := 2, 3
	c := a+b
	return id(c)
}
`,
		},
		testbuild.Decl{
			Src: `
func f() uint32 {
	a, b := 2, 3
	c := uint32(a+b)
	return c
}
`,
		},
		testbuild.Decl{
			Src: `
func g(uint32) uint32
func f() uint32 {
	a := uint32(2)
	a = a + 2
	return g(a)
}
`,
		},
		testbuild.Decl{
			Src: `
func f() int64 {
	true := 3
	return true
}
`,
		},
		testbuild.Decl{
			Src: `
type st struct {
	a float32
}

func a() st {
	return st{a:2}
}
`,
		},
		testbuild.Decl{
			Src: `
type st struct {
	a float32
}

func a() st {
	r := st{a:2}
	r.a = 3
	return r
}
`,
		},
		testbuild.Decl{
			Src: `
func f() int64 {
	x = 0
	return x
}
`,
			Err: "undefined: x",
		},
		testbuild.Decl{
			Src: `
type st struct {
	a float32
}

func a() st {
	r := st{a:2}
	r.a = int32(3)
	return r
}
`,
			Err: "cannot use int32 as float32",
		},
		testbuild.Decl{
			Src: `
func f() int64 {
	a := 2
	a := 3
	return a
}
`,
			Err: "no new variables on left side of :=",
		},
	)
}
