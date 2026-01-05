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

func TestCast(t *testing.T) {
	testbuild.Run(t,
		testbuild.Expr{
			Src: `[2]float32(1.0)`,
			Want: &ir.CastExpr{
				Typ: irh.ArrayType(ir.Float32Type(), 2),
				X:   irh.FloatNumberAs(1, ir.Float32Type()),
			},
			WantType: "[2]float32",
		},
		testbuild.Expr{
			Src: `[2]float32(1)`,
			Want: &ir.CastExpr{
				Typ: irh.ArrayType(ir.Float32Type(), 2),
				X:   irh.IntNumberAs(1, ir.Float32Type()),
			},
			WantType: "[2]float32",
		},
		testbuild.Expr{
			Src: `[2]float32([2]int64{3, 4})`,
			Want: &ir.CastExpr{
				X: &ir.ArrayLitExpr{
					Typ: irh.ArrayType(ir.Int64Type(), 2),
					Elts: []ir.AssignableExpr{
						irh.IntNumberAs(3, ir.Int64Type()),
						irh.IntNumberAs(4, ir.Int64Type()),
					},
				},
				Typ: irh.ArrayType(ir.Float32Type(), 2),
			},
			WantType: "[2]float32",
		},
		testbuild.Expr{
			Src: `([2]float32)([2]int64{3, 4})`,
			Want: &ir.CastExpr{
				X: &ir.ArrayLitExpr{
					Typ: irh.ArrayType(ir.Int64Type(), 2),
					Elts: []ir.AssignableExpr{
						irh.IntNumberAs(3, ir.Int64Type()),
						irh.IntNumberAs(4, ir.Int64Type()),
					},
				},
				Typ: irh.ArrayType(ir.Float32Type(), 2),
			},
			WantType: "[2]float32",
		},
		testbuild.Expr{
			Src: `[___]float32([2]int64{3, 4})`,
			Want: &ir.CastExpr{
				X: &ir.ArrayLitExpr{
					Typ: irh.ArrayType(ir.Int64Type(), 2),
					Elts: []ir.AssignableExpr{
						irh.IntNumberAs(3, ir.Int64Type()),
						irh.IntNumberAs(4, ir.Int64Type()),
					},
				},
				Typ: irh.ArrayType(ir.Float32Type(), &ir.RankInfer{
					Rnk: &ir.Rank{Ax: []ir.AxisLengths{irh.Axis(2)}},
				}),
			},
			WantType: "[2]float32",
		},
		testbuild.Expr{
			Src: `[___]float32([2]bool{true, false})`,
			Want: &ir.CastExpr{
				X: &ir.ArrayLitExpr{
					Typ: irh.ArrayType(ir.BoolType(), 2),
					Elts: []ir.AssignableExpr{
						&ir.ValueRef{
							Stor: ir.TrueStorage(),
						},
						&ir.ValueRef{
							Stor: ir.FalseStorage(),
						},
					},
				},
				Typ: irh.ArrayType(ir.Float32Type(), &ir.RankInfer{
					Rnk: &ir.Rank{Ax: []ir.AxisLengths{irh.Axis(2)}},
				}),
			},
			WantType: "[2]float32",
		},
	)
}

func TestCastStaticVar(t *testing.T) {
	aVarDecl := irh.VarSpec("a")
	xFunc := &ir.FuncBuiltin{
		Src: &ast.FuncDecl{Name: &ast.Ident{Name: "x"}},
		FType: irh.FuncType(
			nil, nil,
			irh.Fields(ir.Float32Type()),
			irh.Fields(ir.Float32Type()),
		),
	}
	testbuild.Run(t,
		testbuild.Decl{
			Src: `
var a intlen

func x(float32) float32

func f() float32 {
	return x(float32(a))
}
`,
			Want: []ir.IR{
				aVarDecl,
				xFunc,
				&ir.FuncDecl{
					Src: &ast.FuncDecl{Name: &ast.Ident{Name: "f"}},
					FType: irh.FuncType(
						nil, nil,
						irh.Fields(),
						irh.Fields(ir.Float32Type()),
					),
					Body: irh.SingleReturn(&ir.FuncCallExpr{
						Args: []ir.AssignableExpr{&ir.CastExpr{
							X:   irh.ValueRef(aVarDecl.Exprs[0]),
							Typ: ir.Float32Type(),
						}},
						Callee: ir.NewFuncValExpr(xFunc, xFunc),
					}),
				},
			},
		},
		testbuild.Decl{
			Src: `
func f() [2][3]float64 {
	return ([2][3]float64)([6]float32{1, 2, 3, 4, 5, 6})
}
`,
			Want: []ir.IR{
				&ir.FuncDecl{
					Src: &ast.FuncDecl{Name: &ast.Ident{Name: "f"}},
					FType: irh.FuncType(
						nil, nil,
						irh.Fields(),
						irh.Fields(irh.ArrayType(ir.Float64Type(), 2, 3)),
					),
					Body: irh.SingleReturn(&ir.CastExpr{
						Typ: irh.ArrayType(ir.Float64Type(), 2, 3),
						X: &ir.ArrayLitExpr{
							Typ: irh.ArrayType(ir.Float32Type(), 6),
							Elts: []ir.AssignableExpr{
								irh.IntNumberAs(1, ir.Float32Type()),
								irh.IntNumberAs(2, ir.Float32Type()),
								irh.IntNumberAs(3, ir.Float32Type()),
								irh.IntNumberAs(4, ir.Float32Type()),
								irh.IntNumberAs(5, ir.Float32Type()),
								irh.IntNumberAs(6, ir.Float32Type()),
							},
						},
					}),
				},
			},
		},
		testbuild.Decl{
			Src: `
var a intlen

func newArray() [a]float32
func id([a]float32) float32

func f() float32 {
	return id(newArray())
}
`,
		},
		testbuild.Decl{
			Src: `
var a intlen

func newArray() [a]int32
func id([a]float32) float32

func f() float32 {
	return id(([a]float32)(newArray()))
}
`,
		},
	)
}

func TestCastToGenerics(t *testing.T) {
	testbuild.Run(t,
		testbuild.Decl{
			Src: `
type someInts interface { 
	int32 | int64 
}

func f[T someInts](x bool) T {
	return T(x) 
}
`,
		},
		testbuild.Decl{
			Src: `
type Ints interface { 
	int32 | int64 
}

type Floats interface {
	float32 | float64
}

func ToInts[U Ints, T Floats](a [2][3]T) [2][3]U {
    return [2][3]U(a)
}
`,
		},
		testbuild.Decl{
			Src: `
type Ints struct { 
	i int32 
}

type Floats interface {
	float32 | float64
}

func ToInts[T Floats](a T) Ints {
    return Ints(a)
}
`,
			Err: "cannot convert T to test.Ints",
		},
	)
}
