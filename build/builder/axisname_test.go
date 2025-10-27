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

func TestAxisName01(t *testing.T) {
	arrayType := irh.ArrayType(ir.Float32Type(), irh.AxisGroup("shape"))
	sliceLiteral := &ir.SliceLitExpr{
		Typ: ir.IntLenSliceType(),
		Elts: []ir.AssignableExpr{
			irh.IntNumberAs(2, ir.IntLenType()),
			irh.IntNumberAs(3, ir.IntLenType()),
		},
	}
	newArrayFunc := &ir.FuncBuiltin{
		Src: &ast.FuncDecl{Name: &ast.Ident{Name: "newArray"}},
		FType: irh.FuncType(
			nil, nil,
			irh.Fields("shape", ir.IntLenSliceType()),
			irh.Fields(arrayType),
		),
	}
	castFunc := &ir.FuncBuiltin{
		Src: &ast.FuncDecl{Name: &ast.Ident{Name: "cast"}},
		FType: irh.AxisLengths(irh.FuncType(
			nil, nil,
			irh.Fields(irh.ArrayType(ir.Float32Type(), irh.Axis("___S"))),
			irh.Fields(irh.ArrayType(ir.Float64Type(), irh.Axis("S___"))),
		), "___S"),
	}
	dimsField := irh.Field("dims", ir.IntLenSliceType(), nil)
	aStorage := &ir.LocalVarStorage{
		Src: &ast.Ident{Name: "a"},
		Typ: irh.ArrayType(ir.Float32Type(), irh.Axis("dims___")),
	}
	aAssignment := &ir.AssignExpr{
		Storage: aStorage,
		X: &ir.CallExpr{
			Args: []ir.AssignableExpr{irh.ValueRef(dimsField.Storage())},
			Callee: &ir.FuncValExpr{
				X: irh.ValueRef(newArrayFunc),
				F: newArrayFunc,
				T: irh.FuncType(
					nil, nil,
					irh.Fields("dims", irh.TypeRef(ir.IntLenSliceType())),
					irh.Fields(irh.TypeRef(
						irh.ArrayType(ir.Float32Type(), irh.Axis("dims___")),
					)),
				),
			},
		},
	}
	testbuild.Run(t,
		testbuild.Decl{
			Src: `
func newArray(shape []intlen) [shape___]float32
`,
			Want: []ir.Node{newArrayFunc},
		},
		testbuild.Decl{
			Src: `
func newArray(shape []intlen) [shape___]float32

func f(dims []intlen) [dims___]float32 {
	return newArray(dims)
}
`,
			Want: []ir.Node{
				newArrayFunc,
				&ir.FuncDecl{
					FType: irh.FuncType(
						nil, nil,
						irh.Fields(dimsField),
						irh.Fields(irh.TypeRef(irh.ArrayType(
							ir.Float32Type(),
							irh.Axis("dims___"),
						))),
					),
					Body: &ir.BlockStmt{List: []ir.Stmt{
						&ir.ReturnStmt{Results: []ir.Expr{
							&ir.CallExpr{
								Args: []ir.AssignableExpr{irh.ValueRef(dimsField.Storage())},
								Callee: &ir.FuncValExpr{
									X: irh.ValueRef(newArrayFunc),
									F: newArrayFunc,
									T: irh.FuncType(
										nil, nil,
										irh.Fields("dims", ir.IntLenSliceType()),
										irh.Fields(irh.ArrayType(
											ir.Float32Type(),
											irh.Axis("dims___")),
										),
									),
								},
							},
						}},
					}},
				},
			},
		},
		testbuild.Decl{
			Src: `
func newArray(shape []intlen) [shape___]float32

func f(dims []intlen) [dims___]float32 {
	a := newArray(dims)
	return a
}

`,
			Want: []ir.Node{
				newArrayFunc,
				&ir.FuncDecl{
					FType: irh.FuncType(
						nil, nil,
						irh.Fields(dimsField),
						irh.Fields(irh.ArrayType(
							ir.Float32Type(),
							irh.Axis("dims___"),
						)),
					),
					Body: &ir.BlockStmt{List: []ir.Stmt{
						&ir.AssignExprStmt{List: []*ir.AssignExpr{
							aAssignment,
						}},
						&ir.ReturnStmt{Results: []ir.Expr{
							irh.ValueRef(aAssignment),
						}},
					}},
				},
			},
		},
		testbuild.Decl{
			Src: `
func newArray(shape []intlen) [shape___]float32

func f() [2][3]float32 {
	return newArray([]intlen{2,3})
}
`,
			Want: []ir.Node{
				newArrayFunc,
				&ir.FuncDecl{
					FType: irh.FuncType(
						nil, nil,
						irh.Fields(),
						irh.Fields(irh.ArrayType(ir.Float32Type(), 2, 3)),
					),
					Body: &ir.BlockStmt{List: []ir.Stmt{
						&ir.ReturnStmt{Results: []ir.Expr{
							&ir.CallExpr{
								Args: []ir.AssignableExpr{sliceLiteral},
								Callee: &ir.FuncValExpr{
									X: irh.ValueRef(newArrayFunc),
									F: newArrayFunc,
									T: irh.FuncType(
										nil, nil,
										irh.Fields("dims", ir.IntLenSliceType()),
										irh.Fields(irh.ArrayType(ir.Float32Type(), 2, 3)),
									),
								},
							},
						}},
					}},
				},
			},
		},
		testbuild.Decl{
			Src: `
func newArray(shape []intlen) [shape___]float32

func f() [4][3]float32 {
	oneAxis := []intlen{4}
	twoAxis := append(oneAxis, 3)
	return newArray(twoAxis)
}
`,
		},
		testbuild.Decl{
			Src: `
var numClasses intlen

func newArray(shape []intlen) [shape___]float32

func f(x [___axes]float32) [axes___][numClasses]float32 {
	ax := append(axlengths(x), numClasses)
	return newArray(ax)
}
`,
		},
		testbuild.Decl{
			Src: `
func cast([___S]float32) [S___]float64

func f() [2]float64 {
	return cast([2]float32{3, 4})
}
`,
			Want: []ir.Node{
				castFunc,
				&ir.FuncDecl{
					FType: irh.FuncType(
						nil, nil,
						irh.Fields(),
						irh.Fields(irh.ArrayType(ir.Float64Type(), 2)),
					),
					Body: &ir.BlockStmt{List: []ir.Stmt{
						&ir.ReturnStmt{Results: []ir.Expr{
							&ir.CallExpr{
								Args: []ir.AssignableExpr{&ir.ArrayLitExpr{
									Typ: irh.ArrayType(ir.Float32Type(), 2),
									Elts: []ir.AssignableExpr{
										irh.IntNumberAs(3, ir.Float32Type()),
										irh.IntNumberAs(4, ir.Float32Type()),
									},
								}},
								Callee: &ir.FuncValExpr{
									X: irh.ValueRef(castFunc),
									F: castFunc,
									T: irh.FuncType(
										nil, nil,
										irh.Fields(irh.ArrayType(ir.Float32Type(), "___S")),
										irh.Fields(irh.ArrayType(ir.Float64Type(), 2)),
									),
								},
							},
						}},
					}},
				},
			},
		},
		testbuild.Decl{
			Src: `
func cast([___S]float32) [S___]float64

func f() float64 {
	return cast(float32(1))
}
`,
			Want: []ir.Node{
				castFunc,
				&ir.FuncDecl{
					FType: irh.FuncType(
						nil, nil,
						irh.Fields(),
						irh.Fields(ir.Float64Type()),
					),
					Body: &ir.BlockStmt{List: []ir.Stmt{
						&ir.ReturnStmt{Results: []ir.Expr{
							&ir.CallExpr{
								Args: []ir.AssignableExpr{&ir.CastExpr{
									X:   irh.IntNumberAs(1, ir.Float32Type()),
									Typ: ir.Float32Type(),
								}},
								Callee: &ir.FuncValExpr{
									X: irh.ValueRef(castFunc),
									F: castFunc,
									T: irh.FuncType(
										nil, nil,
										irh.Fields(irh.ArrayType(ir.Float32Type(), "___S")),
										irh.Fields(irh.ArrayType(ir.Float64Type())),
									),
								},
							},
						}},
					}},
				},
			},
		},
		testbuild.Decl{
			Src: `
func cast([___S]float32) [S___]float64

func f(a [2][2]float32) [2][2]float64 {
	return cast(a+a)
}
`,
		},
		testbuild.Decl{
			Src: `
type floats interface {
	float32 | float64
}

func g[T floats]([___S]T) [S___]T

func f() [2]float32 {
	return g[float32]([...]float32{
		0, 1,
	})
}
`,
		},
	)
}

func TestAxisName02(t *testing.T) {
	arrayType := irh.ArrayType(ir.Float32Type(), irh.AxisGroup("shape"))
	newArrayFunc := &ir.FuncBuiltin{
		Src: &ast.FuncDecl{Name: &ast.Ident{Name: "newArray"}},
		FType: irh.FuncType(
			nil, nil,
			irh.Fields("shape", ir.IntLenSliceType()),
			irh.Fields(arrayType),
		),
	}
	fDims := &ir.LocalVarStorage{
		Src: irh.Ident("fDims"),
		Typ: ir.IntLenSliceType(),
	}
	fDimsAssignment := &ir.AssignExpr{
		Storage: fDims,
		X: &ir.SliceLitExpr{
			Typ: ir.IntLenSliceType(),
			Elts: []ir.AssignableExpr{
				irh.IntNumberAs(2, ir.IntLenType()),
				irh.IntNumberAs(3, ir.IntLenType()),
			},
		},
	}
	testbuild.Run(t,
		testbuild.Decl{
			Src: `
func newArray(shape []intlen) [shape___]float32

func f() [2][3]float32  {
	fDims := []intlen{2, 3}
	return newArray(fDims)
}
`,
			Want: []ir.Node{
				newArrayFunc,
				&ir.FuncDecl{
					FType: irh.FuncType(
						nil, nil,
						irh.Fields(),
						irh.Fields(irh.ArrayType(ir.Float32Type(), 2, 3)),
					),
					Body: &ir.BlockStmt{List: []ir.Stmt{
						&ir.AssignExprStmt{
							List: []*ir.AssignExpr{fDimsAssignment},
						},
						&ir.ReturnStmt{Results: []ir.Expr{
							&ir.CallExpr{
								Args: []ir.AssignableExpr{irh.ValueRef(fDimsAssignment)},
								Callee: &ir.FuncValExpr{
									X: irh.ValueRef(newArrayFunc),
									F: newArrayFunc,
									T: newArrayFunc.FType,
								},
							},
						}},
					}},
				},
			},
		},
		testbuild.Decl{
			Src: `
var A, B intlen

func g([___M]float32) [M___]float32

func f(x [A][B]float32) [A][B]float32 {
	return g(x) 
}
`,
		},
		testbuild.Decl{
			Src: `
var A, B intlen

func h(x [___]int32, shape []intlen) [shape___]int32

func f(a [A]int32) [A][B][1]int32 {
	return h(a, []intlen{A, B, 1})
}
`,
		},
		testbuild.Decl{
			Src: `
var A, B intlen

func h(x [1][A]int32, shape []intlen) [shape___]int32

func f(a [A]int32) [A][B][1]int32 {
	return h([1][A]int32(a), []intlen{A, B, 1})
}
`,
		},
	)
}

func TestAxisGroupOrigin(t *testing.T) {
	testbuild.Run(t,
		testbuild.Decl{
			Src: `
func f([___M]int32) [M___]int32

func g(a [___dims]int32) [dims___]int32 {
	return a + f(a)
}
`,
			Want: []ir.Node{},
		},
		testbuild.Decl{
			Src: `
func sum([___M]int32) int64

func callCast(x [_]int32) int64 {
	return sum(x)
}
`,
		},
		testbuild.Decl{
			Src: `
func g([___M]int32) [M___]int32 

func f() int32 {
	return g(int32(1))
}
`,
		},
	)
}

func TestSingleAxisName(t *testing.T) {
	testbuild.Run(t,
		testbuild.Decl{
			Src: `
func add(x, y [_ax1][_ax2]float32) [ax1][ax2]float32 {
	return x + y
}

func callAdd(x, y [2][3]float32) [2][3]float32 {
	return add(x, y)
}
`,
		},
		testbuild.Decl{
			Src: `
func g(x [_]float32) int64

func f() int64 {
	return g([...]float32{1, 2, 3})
}
`,
		},
	)
}

func TestAxisErrors(t *testing.T) {
	testbuild.Run(t,
		testbuild.Decl{
			Src: `
func f([___M]float32) [___M]float32
`,
			Err: "shape M using ___M can only be defined in function parameters",
		},
	)
}
