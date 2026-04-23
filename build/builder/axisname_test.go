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
	sliceLiteral := &ir.SliceLitExpr{
		Typ: ir.IntLenSliceType(),
		Elts: []ir.Expr{
			irh.IntNumberAs(2, ir.IntLenType()),
			irh.IntNumberAs(3, ir.IntLenType()),
		},
	}
	newArrayFunc := &ir.FuncBuiltin{
		Src: &ast.FuncDecl{Name: &ast.Ident{Name: "newArray"}},
		FType: irh.FuncType(
			nil, nil,
			irh.Fields("dims", ir.IntLenSliceType()),
			irh.Fields(irh.ArrayType(ir.Float32Type(), irh.Axis("dims___"))),
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
		X: &ir.FuncCallExpr{
			Args: []ir.Expr{irh.Ident(dimsField.Storage())},
			Callee: irh.FuncExpr(newArrayFunc).NewFType(
				irh.FuncType(
					nil, nil,
					irh.Fields("dims", ir.TypeExpr(nil, ir.IntLenSliceType())),
					irh.Fields(ir.TypeExpr(
						nil,
						irh.ArrayType(ir.Float32Type(), irh.Axis("dims___")),
					)),
				)),
		},
	}
	testbuild.Run(t,
		testbuild.Decl{
			Src: `
func newArray(dims []intlen) [dims___]float32
`,
			Want: []ir.IR{newArrayFunc},
		},
		testbuild.Decl{
			Src: `
func newArray(dims []intlen) [dims___]float32

func f(dims []intlen) [dims___]float32 {
	return newArray(dims)
}
`,
			Want: []ir.IR{
				newArrayFunc,
				&ir.FuncDecl{
					FType: irh.FuncType(
						nil, nil,
						irh.Fields(dimsField),
						irh.Fields(ir.TypeExpr(nil, irh.ArrayType(
							ir.Float32Type(),
							irh.Axis("dims___"),
						))),
					),
					Body: &ir.BlockStmt{List: []ir.Stmt{
						&ir.ReturnStmt{Results: []ir.Expr{
							&ir.FuncCallExpr{
								Args: []ir.Expr{irh.Ident(dimsField.Storage())},
								Callee: irh.FuncExpr(newArrayFunc).NewFType(
									irh.FuncType(
										nil, nil,
										irh.Fields("dims", ir.IntLenSliceType()),
										irh.Fields(irh.ArrayType(
											ir.Float32Type(),
											irh.Axis("dims___")),
										),
									),
								),
							},
						}},
					}},
				},
			},
		},
		testbuild.Decl{
			Src: `
func newArray(dims []intlen) [dims___]float32

func f(dims []intlen) [dims___]float32 {
	a := newArray(dims)
	return a
}

`,
			Want: []ir.IR{
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
							irh.Ident(aAssignment),
						}},
					}},
				},
			},
		},
		testbuild.Decl{
			Src: `
func newArray(dims []intlen) [dims___]float32

func f() [2][3]float32 {
	return newArray([]intlen{2,3})
}
`,
			Want: []ir.IR{
				newArrayFunc,
				&ir.FuncDecl{
					FType: irh.FuncType(
						nil, nil,
						irh.Fields(),
						irh.Fields(irh.ArrayType(ir.Float32Type(), 2, 3)),
					),
					Body: &ir.BlockStmt{List: []ir.Stmt{
						&ir.ReturnStmt{Results: []ir.Expr{
							&ir.FuncCallExpr{
								Args: []ir.Expr{sliceLiteral},
								Callee: irh.FuncExpr(newArrayFunc).NewFType(
									irh.FuncType(
										nil, nil,
										irh.Fields("dims", ir.IntLenSliceType()),
										irh.Fields(irh.ArrayType(ir.Float32Type(), 2, 3)),
									),
								)},
						}},
					}},
				},
			},
		},
		testbuild.Decl{
			Src: `
func newArray(dims []intlen) [dims___]float32

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

func newArray[axes []intlen]() [axes]float32

func f[axes []intlen](x [axes]float32) [axes][numClasses]float32 {
	ax := append(axlengths(x), numClasses)
	return newArray[ax]()
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
			Want: []ir.IR{
				castFunc,
				&ir.FuncDecl{
					FType: irh.FuncType(
						nil, nil,
						irh.Fields(),
						irh.Fields(irh.ArrayType(ir.Float64Type(), 2)),
					),
					Body: &ir.BlockStmt{List: []ir.Stmt{
						&ir.ReturnStmt{Results: []ir.Expr{
							&ir.FuncCallExpr{
								Args: []ir.Expr{&ir.ArrayLitExpr{
									Typ: irh.ArrayType(ir.Float32Type(), 2),
									Elts: []ir.Expr{
										irh.IntNumberAs(3, ir.Float32Type()),
										irh.IntNumberAs(4, ir.Float32Type()),
									},
								}},
								Callee: irh.FuncExpr(castFunc).NewFType(
									irh.FuncType(
										nil, nil,
										irh.Fields(irh.ArrayType(ir.Float32Type(), 2)),
										irh.Fields(irh.ArrayType(ir.Float64Type(), 2)),
									)),
							},
						},
						}},
					}},
			},
		},
		testbuild.Decl{
			Src: `
func cast([___S]float32) [S___]float64

func f() float64 {
	return cast(float32(1))
}
`,
			Want: []ir.IR{
				castFunc,
				&ir.FuncDecl{
					FType: irh.FuncType(
						nil, nil,
						irh.Fields(),
						irh.Fields(ir.Float64Type()),
					),
					Body: &ir.BlockStmt{List: []ir.Stmt{
						&ir.ReturnStmt{Results: []ir.Expr{
							&ir.FuncCallExpr{
								Args: []ir.Expr{&ir.CastExpr{
									X:   irh.IntNumberAs(1, ir.Float32Type()),
									Typ: ir.Float32Type(),
								}},
								Callee: irh.FuncExpr(castFunc).NewFType(
									irh.FuncType(
										nil, nil,
										irh.Fields(irh.ArrayType(ir.Float32Type())),
										irh.Fields(irh.ArrayType(ir.Float64Type())),
									)),
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

func g[T floats, S []intlen]([S]T) [S]T

func f() [2]float32 {
	return g[float32]([...]float32{
		0, 1,
	})
}
`,
		},
		testbuild.Decl{
			Src: `
func f(x [_ax]float32) int64 {
	return len(x)
}
`,
		},
	)
}

func TestGenericAxis(t *testing.T) {
	ax := irh.Field("ax", ir.IntLenType(), nil)
	axes := irh.Field("axes", ir.IntLenSliceType(), nil)
	newArray := &ir.FuncBuiltin{
		Src: &ast.FuncDecl{
			Name: &ast.Ident{Name: "newArray"},
		},
		FType: irh.FuncType(
			irh.Fields(axes),
			nil,
			irh.Fields(),
			irh.Fields(irh.ArrayType(ir.Float32Type(), axes)),
		),
	}
	testbuild.Run(t,
		testbuild.Decl{
			Src: `
func f[ax intlen]() [ax]float32
`,
			Want: []ir.IR{
				&ir.FuncBuiltin{
					FType: irh.FuncType(
						irh.Fields(ax),
						nil,
						irh.Fields(),
						irh.Fields(irh.ArrayType(ir.Float32Type(), ax)),
					),
				},
			},
		},
		testbuild.Decl{
			Src: `
func f[ax intlen]() [ax]float32

func g() [2]float32 {
	return f[2]()
}
`,
		},
		testbuild.Decl{
			Src: `
func f[ax intlen]() [ax]float32 {
	return [ax]float32{}
}

func g() [2]float32 {
	return f[2]()
}
`,
		},
		testbuild.Decl{
			Src: `
func f[axes []intlen]() [axes]float32

func g() [2][3]float32 {
	return f[[]intlen{2,3}]()
}
`,
			Want: []ir.IR{
				newArray,
				&ir.FuncDecl{
					FType: irh.FuncType(nil, nil,
						irh.Fields(),
						irh.Fields(irh.ArrayType(ir.Float32Type(), 2, 3)),
					),
					Body: irh.SingleReturn(&ir.FuncCallExpr{
						Callee: irh.FuncExpr(newArray).NewFType(
							irh.FuncType(
								irh.Fields(), nil,
								irh.Fields(),
								irh.Fields(irh.ArrayType(ir.Float32Type(), 2, 3)),
								irh.SetAxisLength("axes", []int{2, 3}),
							)),
					}),
				},
			},
		},
	)
}

func TestAxisName02(t *testing.T) {
	newArrayFunc := &ir.FuncBuiltin{
		Src: &ast.FuncDecl{Name: &ast.Ident{Name: "newArray"}},
		FType: irh.FuncType(
			nil, nil,
			irh.Fields("dims", ir.IntLenSliceType()),
			irh.Fields(irh.ArrayType(ir.Float32Type(), irh.Axis("dims___"))),
		),
	}
	fDims := &ir.LocalVarStorage{
		Src: irh.IdentAST("fDims"),
		Typ: ir.IntLenSliceType(),
	}
	fDimsAssignment := &ir.AssignExpr{
		Storage: fDims,
		X: &ir.SliceLitExpr{
			Typ: ir.IntLenSliceType(),
			Elts: []ir.Expr{
				irh.IntNumberAs(2, ir.IntLenType()),
				irh.IntNumberAs(3, ir.IntLenType()),
			},
		},
	}
	testbuild.Run(t,
		testbuild.Decl{
			Src: `
func newArray(dims []intlen) [dims___]float32

func f() [2][3]float32  {
	fDims := []intlen{2, 3}
	return newArray(fDims)
}
`,
			Want: []ir.IR{
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
							&ir.FuncCallExpr{
								Args: []ir.Expr{irh.Ident(fDimsAssignment)},
								Callee: irh.FuncExpr(newArrayFunc).NewFType(
									irh.FuncType(
										nil, nil,
										irh.Fields("dims", ir.IntLenSliceType()),
										irh.Fields(irh.ArrayType(ir.Float32Type(), 2, 3)),
									)),
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

func h(x [___]int32, dims []intlen) [dims___]int32

func f(a [A]int32) [A][B][1]int32 {
	return h(a, []intlen{A, B, 1})
}
`,
		},
		testbuild.Decl{
			Src: `
var A, B intlen

func h(x [1][A]int32, dims []intlen) [dims___]int32

func f(a [A]int32) [A][B][1]int32 {
	return h([1][A]int32(a), []intlen{A, B, 1})
}
`,
		},
		testbuild.Decl{
			Src: `
func newArray(dims []intlen) [dims___]float32

func f() float32  {
	return newArray([]intlen{})
}
`,
		},
	)
}

func TestAxisGroupOrigin(t *testing.T) {
	testbuild.Run(t,
		testbuild.Decl{
			Src: `
func f[S []intlen]([S]int32) [S]int32

func g[S []intlen](a [S]int32) [S]int32 {
	return a + f(a)
}
`,
			Want: []ir.IR{},
		},
		testbuild.Decl{
			Src: `
func sum[S []intlen]([S]int32) int64

func callCast[ax intlen](x [ax]int32) int64 {
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
func g(x [_ax]float32) int64

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
func f([___M]float32) [___M]float32 // ERROR shape M using ___M can only be defined in function parameters
`,
		},
		testbuild.Decl{
			Src: `
func f([_a]float32) [a]float32

func g(x float32) [2]float32 {
	return f(x) // ERROR cannot use type float32 as [0]float32 in argument to test.f
}
`,
			Err: "no axis left to define a",
		},
		testbuild.Decl{
			Src: `
func newArray(lens []intlen) [lens___]float32

func f() float32 {
	var a [1]float32
	a = newArray([]intlen{}) // ERROR cannot use float32 as [1]float32 value in assignment
	return a[0]
}
`,
		},
	)
}

func TestAxisStatements(t *testing.T) {
	testbuild.Run(t,
		testbuild.Decl{
			Src: `
type floats interface {
	float32 | float64
}

func g[T floats](x [___M]T) [M___]T

func f[T floats]([___M]T) [M___]T

func reverse[T floats](par [___M]T) ([M___]T, func(res [M___]T) [M___]T) {
      fwd0 := f(par)
      selfVJPFunc := func(res [M___]T) [M___]T {
              return g(par)
      }
      return fwd0, selfVJPFunc
}

func Test() (float64, float64) {
	y, vjp := reverse(0.8)
	yprime := vjp(1.0)
	return y, yprime
}
`,
		},
	)
}

func TestAxisNameWithBuiltin(t *testing.T) {
	testbuild.Run(t,
		testbuild.Decl{
			Src: `
type floats interface {
	float32 | float64
}

func f[T floats]([___M]T) [M___]T

func Test() float64 {
	trace(f)
	return 0
}
`,
		},
	)
}
