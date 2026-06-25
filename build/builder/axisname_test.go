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

func TestAxisIntLen(t *testing.T) {
	fieldS := irh.Field("S", ir.IntLenType(), nil)
	newArrayFunc := &ir.FuncBuiltin{
		Src: &ast.FuncDecl{Name: &ast.Ident{Name: "newArray"}},
		FType: irh.FuncType(
			irh.Fields(fieldS),
			nil,
			irh.Fields(),
			irh.Fields(irh.ArrayType(ir.Float32Type(), irh.Axis(fieldS))),
		),
	}
	testbuild.Run(t,
		testbuild.Decl{
			Src: `
func newArray[S intlen]() [S]float32
`,
			Want: []ir.IR{newArrayFunc},
		},
	)
}

func TestAxisName01(t *testing.T) {
	fieldS := irh.Field("S", ir.IntLenSliceType(), nil)
	fieldDims := irh.Field("dims", ir.IntLenSliceType(), nil)
	newArrayFunc := &ir.FuncBuiltin{
		Src: &ast.FuncDecl{Name: &ast.Ident{Name: "newArray"}},
		FType: irh.FuncType(
			irh.Fields(fieldS),
			nil,
			irh.Fields(),
			irh.Fields(irh.ArrayType(
				ir.Float32Type(),
				irh.Axis(irh.UnpackAxes(fieldS)),
			)),
		),
	}
	castFunc := &ir.FuncBuiltin{
		Src: &ast.FuncDecl{Name: &ast.Ident{Name: "cast"}},
		FType: irh.FuncType(
			irh.Fields(fieldS),
			nil,
			irh.Fields(irh.ArrayType(ir.Float32Type(), irh.Axis(irh.UnpackAxes(fieldS)))),
			irh.Fields(irh.ArrayType(ir.Float64Type(), irh.Axis(irh.UnpackAxes(fieldS)))),
		),
	}
	aStorage := &ir.LocalVarStorage{
		Src: &ast.Ident{Name: "a"},
		Typ: irh.ArrayType(ir.Float32Type(), irh.Axis(fieldDims)),
	}
	aAssignment := &ir.AssignExpr{
		Storage: aStorage,
		X: &ir.FuncCallExpr{
			Callee: irh.FuncExpr(newArrayFunc).NewFType(
				irh.FuncType(
					irh.Fields(fieldS),
					nil,
					irh.Fields(),
					irh.Fields(ir.TypeExpr(
						nil,
						irh.ArrayType(ir.Float32Type(), irh.Axis(fieldDims)),
					)),
					irh.SetTypeParams(fieldDims),
				)),
		},
	}
	testbuild.Run(t,
		testbuild.Decl{
			Src: `
func newArray[S []intlen]() [unpack(S)]float32
`,
			Want: []ir.IR{newArrayFunc},
		},
		testbuild.Decl{
			Src: `
func newArray[S []intlen]() [unpack(S)]float32

func f[dims []intlen]() [unpack(dims)]float32 {
	return newArray[dims]()
}
`,
			Want: []ir.IR{
				newArrayFunc,
				&ir.FuncDecl{
					FType: irh.FuncType(
						irh.Fields(fieldDims),
						nil,
						irh.Fields(),
						irh.Fields(ir.TypeExpr(nil, irh.ArrayType(
							ir.Float32Type(),
							irh.Axis(irh.UnpackAxes(fieldDims)),
						))),
					),
					Body: &ir.BlockStmt{List: []ir.Stmt{
						&ir.ReturnStmt{Results: []ir.Expr{
							&ir.FuncCallExpr{
								Callee: irh.FuncExpr(newArrayFunc).NewFType(
									irh.FuncType(
										irh.Fields(fieldDims),
										nil,
										irh.Fields(),
										irh.Fields(irh.ArrayType(
											ir.Float32Type(),
											irh.Axis(fieldDims),
										)),
										irh.SetTypeParams(fieldDims),
									)),
							},
						}},
					}},
				},
			},
		},
		testbuild.Decl{
			Src: `
func newArray[S []intlen]() [unpack(S)]float32

func f[dims []intlen]() [unpack(dims)]float32 {
	a := newArray[dims]()
	return a
}

`,
			Want: []ir.IR{
				newArrayFunc,
				&ir.FuncDecl{
					FType: irh.FuncType(
						irh.Fields(fieldDims),
						nil,
						irh.Fields(),
						irh.Fields(irh.ArrayType(
							ir.Float32Type(),
							irh.Axis(irh.UnpackAxes(fieldDims)),
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
func newArray[S []intlen]() [unpack(S)]float32

func f() [2][3]float32 {
	return newArray[[]intlen{2,3}]()
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
								Callee: irh.FuncExpr(newArrayFunc).NewFType(
									irh.FuncType(
										irh.Fields(fieldDims),
										nil,
										irh.Fields(),
										irh.Fields(irh.ArrayType(ir.Float32Type(), 2, 3)),
										irh.SetTypeParams([]int{2, 3}),
									),
								)},
						}},
					}},
				},
			},
		},
		testbuild.Decl{
			Src: `
func newArray[S []intlen]() [unpack(S)]float32

func f() [4][3]float32 {
	oneAxis := []intlen{4}
	twoAxis := append(oneAxis, 3)
	return newArray[twoAxis]()
}
`,
		},
		testbuild.Decl{
			Src: `
var numClasses intlen

func newArray[Axes []intlen]() [unpack(Axes)]float32

func f[Axes []intlen](x [unpack(Axes)]float32) [unpack(Axes)][numClasses]float32 {
	all := append(axlengths(x), numClasses)
	return newArray[all]()
}
`,
		},
		testbuild.Decl{
			Src: `
func cast[S []intlen]([unpack(S)]float32) [unpack(S)]float64

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
										irh.Fields("S", ir.IntLenSliceType()),
										nil,
										irh.Fields(irh.ArrayType(ir.Float32Type(), 2)),
										irh.Fields(irh.ArrayType(ir.Float64Type(), 2)),
										irh.SetTypeParams([]int{2}),
									)),
							},
						},
						}},
					}},
			},
		},
		testbuild.Decl{
			Src: `
func cast[S []intlen]([unpack(S)]float32) [unpack(S)]float64

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
										irh.Fields(fieldS),
										nil,
										irh.Fields(irh.ArrayType(ir.Float32Type())),
										irh.Fields(irh.ArrayType(ir.Float64Type())),
										irh.SetTypeParams([]int{}),
									)),
							},
						}},
					}},
				},
			},
		},
		testbuild.Decl{
			Src: `
func cast[S []intlen]([unpack(S)]float32) [unpack(S)]float64

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

func g[T floats, S []intlen]([unpack(S)]T) [unpack(S)]T

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
	newVector := &ir.FuncBuiltin{
		Src: &ast.FuncDecl{
			Name: &ast.Ident{Name: "newVector"},
		},
		FType: irh.FuncType(
			irh.Fields(ax),
			nil,
			irh.Fields(),
			irh.Fields(irh.ArrayType(ir.Float32Type(), ax)),
		),
	}
	axes := irh.Field("axes", ir.IntLenSliceType(), nil)
	newArray := &ir.FuncBuiltin{
		Src: &ast.FuncDecl{
			Name: &ast.Ident{Name: "newArray"},
		},
		FType: irh.FuncType(
			irh.Fields(axes),
			nil,
			irh.Fields(),
			irh.Fields(irh.ArrayType(ir.Float32Type(), irh.UnpackAxes(axes))),
		),
	}
	testbuild.Run(t,
		testbuild.Decl{
			Src: `
func newVector[ax intlen]() [ax]float32
`,
			Want: []ir.IR{newVector},
		},
		testbuild.Decl{
			Src: `
func newVector[ax intlen]() [ax]float32

func g() [2]float32 {
	return newVector[2]()
}
`,
			Want: []ir.IR{
				newVector,
				&ir.FuncDecl{
					FType: irh.FuncType(
						nil, nil,
						irh.Fields(),
						irh.Fields(irh.ArrayType(ir.Float32Type(), 2)),
					),
					Body: irh.SingleReturn(&ir.FuncCallExpr{
						Callee: irh.FuncExpr(newVector).NewFType(
							irh.FuncType(
								newVector.FType.TypeParams, nil,
								irh.Fields(),
								irh.Fields(irh.ArrayType(ir.Float32Type(), 2)),
								irh.SetTypeParams(2),
							)),
					}),
				},
			},
		},
		testbuild.Decl{
			Src: `
func f[axes []intlen]() [unpack(axes)]float32

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
								newArray.FType.TypeParams, nil,
								irh.Fields(),
								irh.Fields(irh.ArrayType(ir.Float32Type(), 2, 3)),
								irh.SetTypeParams([]int{2, 3}),
							)),
					}),
				},
			},
		},
	)
}

func TestAxisName02(t *testing.T) {
	fieldS := irh.Field("S", ir.IntLenSliceType(), nil)
	newArrayFunc := &ir.FuncBuiltin{
		Src: &ast.FuncDecl{Name: &ast.Ident{Name: "newArray"}},
		FType: irh.FuncType(
			irh.Fields(fieldS),
			nil,
			irh.Fields(),
			irh.Fields(irh.ArrayType(ir.Float32Type(), irh.Axis(irh.UnpackAxes(fieldS)))),
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
func newArray[S []intlen]() [unpack(S)]float32

func f() [2][3]float32  {
	fDims := []intlen{2, 3}
	return newArray[fDims]()
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
								Callee: irh.FuncExpr(newArrayFunc).NewFType(
									irh.FuncType(
										irh.Fields(fieldS),
										nil,
										irh.Fields(),
										irh.Fields(irh.ArrayType(ir.Float32Type(), 2, 3)),
										irh.SetTypeParams([]int{2, 3}),
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

func g[M []intlen]([unpack(M)]float32) [unpack(M)]float32

func f(x [A][B]float32) [A][B]float32 {
	return g(x) 
}
`,
		},
		testbuild.Decl{
			Src: `
var A, B intlen

func h[dims []intlen, in []intlen](x [unpack(in)]int32) [unpack(dims)]int32

func f(a [A]int32) [A][B][1]int32 {
	return h[[]intlen{A, B, 1}](a)
}
`,
		},
		testbuild.Decl{
			Src: `
var A, B intlen

func h[dims []intlen](x [1][A]int32) [unpack(dims)]int32

func f(a [A]int32) [A][B][1]int32 {
	return h[[]intlen{A, B, 1}]([1][A]int32(a))
}
`,
		},
		testbuild.Decl{
			Src: `
func newArray[dims []intlen]() [unpack(dims)]float32

func f() float32  {
	return newArray[[]intlen{}]()
}
`,
		},
	)
}

func TestAxisGroupOrigin(t *testing.T) {
	testbuild.Run(t,
		testbuild.Decl{
			Src: `
func f[S []intlen]([unpack(S)]int32) [unpack(S)]int32

func g[S []intlen](a [unpack(S)]int32) [unpack(S)]int32 {
	return a + f(a)
}
`,
		},
		testbuild.Decl{
			Src: `
func sum[S []intlen]([unpack(S)]int32) int64

func callCast[ax intlen](x [ax]int32) int64 {
	return sum(x)
}
`,
		},
		testbuild.Decl{
			Src: `
func g[M []intlen]([unpack(M)]int32) [unpack(M)]int32 

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
func g[ax intlen](x [ax]float32) int64

func f() int64 {
	return g([...]float32{1, 2, 3})
}
`,
		},
		testbuild.Decl{
			Src: `
			type Num interface {
				float32|int32
			}

			func Log[S []intlen, T Num]([unpack(S)]T) [unpack(S)]T

			func f[Size intlen, NumClasses intlen, DT Num](probs [Size][NumClasses]DT) [Size][NumClasses]DT {
				return Log(probs)
			}
			`,
		},
	)
}

func TestAxisErrors(t *testing.T) {
	testbuild.Run(t,
		testbuild.Decl{
			Src: `
func f[a intlen]([a]float32) [a]float32

func g(x float32) [2]float32 {
	return f(x) // ERROR cannot use type float32 as [0]float32 in argument to test.f
}
`,
			Err: "no axis left to define generic parameter a",
		},
		testbuild.Decl{
			Src: `
func newArray[lens []intlen]() [unpack(lens)]float32

func f() float32 {
	var a [1]float32
	a = newArray[[]intlen{}]() // ERROR cannot use float32 as [1]float32 value in assignment
	return a[0]
}
`,
		},
		testbuild.Decl{
			Src: `
type floats interface {
	float32|float64
}

func g[Shape []intlen, T floats]([]intlen) [unpack(Shape)]T

func f[Shape []intlen, T floats]() [unpack(Shape)]T {
	return g[T](Shape) // ERROR cannot use metatype as []intlen generic value Shape
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

func reverse[S []intlen, T floats](par [unpack(S)]T) ([unpack(S)]T, func(res [unpack(S)]T) [unpack(S)]T)

func f() (float64, func(float64) float64) {
	return reverse(0.8)
}
`,
		},
		testbuild.Decl{
			Src: `
type floats interface {
	float32 | float64
}

func g[M []intlen, T floats](x [unpack(M)]T) [unpack(M)]T

func f[M []intlen, T floats]([unpack(M)]T) [unpack(M)]T

func reverse[M []intlen, T floats](par [unpack(M)]T) ([unpack(M)]T, func(res [unpack(M)]T) [unpack(M)]T) {
      fwd0 := f(par)
      selfVJPFunc := func(res [unpack(M)]T) [unpack(M)]T {
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
		testbuild.Decl{
			Src: `
func Broadcast[Dst, Src []intlen](x [unpack(Src)]int32) [unpack(Dst)]int32

func F[ax1, ax2 intlen](x [ax1]int32) [ax1][ax2]int32 {
	ax := append([]intlen{ax1}, ax2)
	return Broadcast[ax](([ax1][1]int32)(x))
}`,
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

func f[M []intlen, T floats]([unpack(M)]T) [unpack(M)]T

func Test() float64 {
	trace(f)
	return 0
}
`,
		},
	)
}
