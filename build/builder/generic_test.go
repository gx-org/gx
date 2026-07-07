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
	"github.com/gx-org/gx/build/ir/irhelper"
)

var intTypeSet = irhelper.TypeSet(ir.Int32Type(), ir.Int64Type())

func TestGenericSignature(t *testing.T) {
	anyS := irhelper.Field("S", ir.AnyType(), nil)
	intsT := irhelper.Field("T", intTypeSet, nil)
	xAxLen := irhelper.AxisLenName("X")
	yAxLen := irhelper.AxisLenName("Y")
	xAxGroup := irhelper.AxisGroup("X")
	testbuild.Run(t,
		testbuild.Decl{
			Src: `func f[S any](S) S`,
			Want: []ir.IR{
				&ir.FuncBuiltin{
					FType: irhelper.FuncType(
						irhelper.Fields(anyS),
						nil,
						irhelper.Fields(ir.NewGenericTypeParam(anyS)),
						irhelper.Fields(ir.NewGenericTypeParam(anyS)),
					),
				},
			},
		},
		testbuild.Decl{
			Src: `func f[T interface{int32|int64}](T) T`,
			Want: []ir.IR{
				&ir.FuncBuiltin{
					FType: irhelper.FuncType(
						irhelper.Fields(intsT),
						nil,
						irhelper.Fields(ir.NewGenericTypeParam(intsT)),
						irhelper.Fields(ir.NewGenericTypeParam(intsT)),
					),
				},
			},
		},
		testbuild.Decl{
			Src: `func f[S any, T interface{int32|int64}](S) T`,
			Want: []ir.IR{
				&ir.FuncBuiltin{
					FType: irhelper.FuncType(
						irhelper.Fields(anyS, intsT),
						nil,
						irhelper.Fields(ir.NewGenericTypeParam(anyS)),
						irhelper.Fields(ir.NewGenericTypeParam(intsT)),
					),
				},
			},
		},
		testbuild.Decl{
			Src: `func f[X int, Y int](x [X][Y]int32) ([X]int32, [Y]int32)`,
			Want: []ir.IR{
				&ir.FuncBuiltin{
					FType: irhelper.FuncType(
						irhelper.Fields(xAxLen, yAxLen),
						nil,
						irhelper.Fields(
							"x",
							irhelper.ArrayType(
								ir.Int32Type(),
								irhelper.Axis(xAxLen),
								irhelper.Axis(yAxLen),
							)),
						irhelper.Fields(
							"",
							irhelper.ArrayType(
								ir.Int32Type(),
								irhelper.Axis(xAxLen),
							),
							"",
							irhelper.ArrayType(
								ir.Int32Type(),
								irhelper.Axis(yAxLen),
							),
						),
					),
				},
			},
		},
		testbuild.Decl{
			Src: `func f[X, X int](x [X][X]int32) [X]int32 // ERROR type parameter X redeclared`,
		},
		testbuild.Decl{
			Src: `func f[X []int]([unpack(X)]int32) [unpack(X)]int32`,
			Want: []ir.IR{
				&ir.FuncBuiltin{
					FType: irhelper.FuncType(
						irhelper.Fields(xAxGroup),
						nil,
						irhelper.Fields(
							"",
							irhelper.ArrayType(
								ir.Int32Type(),
								irhelper.Axis(irhelper.UnpackAxes(xAxGroup)),
							)),
						irhelper.Fields(
							"",
							irhelper.ArrayType(
								ir.Int32Type(),
								irhelper.Axis(irhelper.UnpackAxes(xAxGroup)),
							),
						),
					),
				},
			},
		},
	)
}

func TestGenericCall(t *testing.T) {
	someInt := &ir.NamedType{
		Src:        &ast.TypeSpec{Name: irhelper.IdentAST("someInt")},
		File:       wantFile,
		Underlying: ir.TypeExpr(nil, irhelper.TypeSet(ir.Int32Type(), ir.Int64Type())),
	}
	someIntT := irhelper.Field("T", someInt, nil)
	someIntTP := ir.NewGenericTypeParam(someIntT)
	castNoArgFunc := &ir.FuncDecl{
		Src: &ast.FuncDecl{Name: &ast.Ident{Name: "cast"}},
		FType: irhelper.FuncType(
			irhelper.Fields(someIntT),
			nil,
			irhelper.Fields(),
			irhelper.Fields(someIntTP),
		),
		Body: &ir.BlockStmt{List: []ir.Stmt{
			&ir.ReturnStmt{Results: []ir.Expr{
				&ir.CastExpr{
					X:   irhelper.IntNumberAs(2, someIntTP),
					Typ: someIntTP,
				},
			}},
		}},
	}
	testbuild.Run(t,
		testbuild.DeclarePackage{
			Src: `
package dtype

type (
	Floats interface {
		bfloat16 | float32 | float64
	}

	Ints interface {
		int32 | int64 | uint32 | uint64
	}

	Num interface {
		Floats | Ints
	}
)
`,
		},
		testbuild.Decl{
			Src: `
type someInt interface{ int32 | int64 }

func cast[T someInt]() T {
	return T(2)
}

func callCast() int32 {
	return cast[int32]()
}
`,
			Want: []ir.IR{
				someInt,
				castNoArgFunc,
				&ir.FuncDecl{
					FType: irhelper.FuncType(
						nil, nil,
						irhelper.Fields(),
						irhelper.Fields(ir.Int32Type()),
					),
					Body: &ir.BlockStmt{List: []ir.Stmt{
						&ir.ReturnStmt{Results: []ir.Expr{&ir.FuncCallExpr{
							Callee: irhelper.FuncExpr(castNoArgFunc).NewFType(
								irhelper.FuncType(
									castNoArgFunc.FType.TypeParams,
									nil, nil,
									irhelper.Fields(ir.Int32Type()),
									irhelper.SetTypeParams(ir.Int32Type()),
								),
							),
						}}},
					}},
				},
			},
		},
		testbuild.Decl{
			Src: `
type someInt interface{ int32 | int64}

func g[T someInt]() T

func f[T someInt]() T {
	return g[T]()
}
`,
		},
		testbuild.Decl{
			Src: `
import "dtype"

func f[S []int, T dtype.Num](a [unpack(S)]T) [unpack(S)]T {
	return a+2
}
`,
		},
		testbuild.Decl{
			Src: `
type someInt interface{ int32 | int64 }

func f[T someInt, X []int](x [unpack(X)]T) [unpack(X)]T {
	y := g[T](x)
	return x / y
}

func g[T someInt, X []int](x [unpack(X)]T) T
`,
		},
		testbuild.Decl{
			Src: `
import "dtype"

func f[S []int, T dtype.Num](x [unpack(S)]T) [unpack(S)]T {
	y := g[S,T](x)
	return x / y
}

func g[S []int, T dtype.Num](x [unpack(S)]T) T
`,
		},
		testbuild.Decl{
			Src: `
import "dtype"

func g[T dtype.Num](a T) T

func f() int64 {
	return g(10)
}
`,
		},
		testbuild.Decl{
			Src: `
import "dtype"

func g[T dtype.Num](a T) T

func f() float64 {
	return g(10.0)
}
`,
		},
		testbuild.Decl{
			Src: `
import "dtype"

func g[T dtype.Num](a, b T) T

func f() float32 {
	a := float32(1)
	return g(10, a)
}
`,
		},
		testbuild.Decl{
			Src: `
import "dtype"

func g[T dtype.Floats, X []int](a T, x [unpack(X)]T) [unpack(X)]T

func f() [2][3]float32 {
	return g(10, [2][3]float32{{1, 2, 3}, {4, 5, 6}})
}
`,
		},
		testbuild.Decl{
			Src: `
import "dtype"

func g[T dtype.Num, X []int](a T, x [unpack(X)]T) [unpack(X)]T

func f() [2][3]float32 {
	return g(10, [2][3]float32{{1, 2, 3}, {4, 5, 6}})
}
`,
		},
		testbuild.Decl{
			Src: `
import "dtype"

func g[T dtype.Ints, X []int](a T, x [unpack(X)]T) [unpack(X)]T

func f() [2][3]int32 {
	return g(10.0, [2][3]int32{{1, 2, 3}, {4, 5, 6}}) // ERROR float number does not satisfy dtype.Ints
}
`,
		},
		testbuild.Decl{
			Src: `
import "dtype"

func newFloat32[shape []int]() [unpack(shape)]float32 {
	return [unpack(shape)]float32{}
}

func genericNewFloat32[T dtype.Ints, S []int]() [unpack(S)]T {
	return [unpack(S)]T(newFloat32[S]())
}

func TestGenericCastWithGenericShape() [2][3]int64 {
	return genericNewFloat32[int64][[]int{2, 3}]()
}
`,
		},
		testbuild.Decl{
			Src: `
import "dtype"

func g[X, Y []int, T dtype.Num]([unpack(X)]T, [unpack(Y)]T) [unpack(X)]T

func f() [2]float32 {
	return g([2]float32{}, 2)
}
`,
		},
	)
}

func TestGenericArray(t *testing.T) {
	someInt := &ir.NamedType{
		Src:        &ast.TypeSpec{Name: irhelper.IdentAST("someInt")},
		File:       wantFile,
		Underlying: ir.TypeExpr(nil, irhelper.TypeSet(ir.Int32Type(), ir.Int64Type())),
	}
	typeParamFieldT := irhelper.Field("T", someInt, nil)
	typeParamT := ir.NewGenericTypeParam(typeParamFieldT)
	new2x3ArrayFunc := &ir.FuncDecl{
		Src: &ast.FuncDecl{Name: &ast.Ident{Name: "new2x3Array"}},
		FType: irhelper.FuncType(
			irhelper.Fields(typeParamFieldT),
			nil,
			irhelper.Fields(),
			irhelper.Fields(irhelper.ArrayType(typeParamT, 2, 3)),
		),
		Body: &ir.BlockStmt{List: []ir.Stmt{
			&ir.ReturnStmt{Results: []ir.Expr{&ir.ArrayLitExpr{
				Typ: irhelper.ArrayType(typeParamT, 2, 3),
				Elts: []ir.Expr{
					&ir.ArrayLitExpr{
						Typ: irhelper.ArrayType(typeParamT, 3),
						Elts: []ir.Expr{
							irhelper.IntNumberAs(1, typeParamT),
							irhelper.IntNumberAs(2, typeParamT),
							irhelper.IntNumberAs(3, typeParamT),
						},
					},
					&ir.ArrayLitExpr{
						Typ: irhelper.ArrayType(typeParamT, 3),
						Elts: []ir.Expr{
							irhelper.IntNumberAs(4, typeParamT),
							irhelper.IntNumberAs(5, typeParamT),
							irhelper.IntNumberAs(6, typeParamT),
						},
					},
				},
			}}},
		}},
	}
	testbuild.Run(t,
		testbuild.Decl{
			Src: `
type someInt interface{ int32 | int64 }

func new2x3Array[T someInt]() [2][3]T {
	return [2][3]T{{1, 2, 3}, {4, 5, 6}}
}

func callCast() [2][3]int32 {
	return new2x3Array[int32]()
}
`,
			Want: []ir.IR{
				someInt,
				new2x3ArrayFunc,
				&ir.FuncDecl{
					FType: irhelper.FuncType(
						nil, nil,
						irhelper.Fields(),
						irhelper.Fields(irhelper.ArrayType(ir.Int32Type(), 2, 3)),
					),
					Body: &ir.BlockStmt{List: []ir.Stmt{
						&ir.ReturnStmt{Results: []ir.Expr{&ir.FuncCallExpr{
							Callee: irhelper.FuncExpr(new2x3ArrayFunc).NewFType(
								irhelper.FuncType(
									new2x3ArrayFunc.FType.TypeParams,
									nil,
									irhelper.Fields(),
									irhelper.Fields(irhelper.ArrayType(ir.Int32Type(), 2, 3)),
									irhelper.SetTypeParams(ir.Int32Type()),
								)),
						}}},
					}},
				},
			},
		},
		testbuild.Decl{
			Src: `
type someInt interface{ int32 | int64 }

func cast[T someInt]() T {
	return T(2)
}

func callCast() int32 {
	return cast[cast]() // ERROR cast is not a type
}
`,
		},
		testbuild.Decl{
			Src: `
type someInt interface{ int32 | int64 }

func cast[T someInt]() T {
	return T(2)
}

func callCast() int32 {
	return cast[float32]() // ERROR float32 does not satisfy someInt
}
`,
		},
		testbuild.Decl{
			Src: `
type someInt interface{ int32 | int64 }

func f[T someInt, S []int](x [unpack(S)]T) [unpack(S)]T {
	return x
}

func callF() [2]int64 {
	return f[int32]([2]int64{1, 2}) // ERROR cannot use type [2]int64 as [2]int32 in argument to f
}
`,
		},
	)
}

func TestGenericConvert(t *testing.T) {
	someInt := &ir.NamedType{
		Src:        &ast.TypeSpec{Name: irhelper.IdentAST("someInt")},
		File:       wantFile,
		Underlying: ir.TypeExpr(nil, irhelper.TypeSet(ir.Int32Type(), ir.Int64Type())),
	}
	someIntT := irhelper.Field("T", someInt, nil)
	someIntTP := ir.NewGenericTypeParam(someIntT)
	someIntS := irhelper.Field("S", someInt, someIntT.Group)
	valField := irhelper.Field("val", ir.NewGenericTypeParam(someIntS), nil)
	castAtomFunc := &ir.FuncDecl{
		Src: &ast.FuncDecl{Name: &ast.Ident{Name: "cast"}},
		FType: irhelper.FuncType(
			irhelper.Fields(someIntT.Group),
			nil,
			irhelper.Fields("val", ir.NewGenericTypeParam(someIntS)),
			irhelper.Fields(someIntTP),
		),
		Body: &ir.BlockStmt{List: []ir.Stmt{
			&ir.ReturnStmt{Results: []ir.Expr{
				&ir.CastExpr{Typ: someIntTP, X: irhelper.Ident(valField.Storage())},
			}},
		}},
	}
	testbuild.Run(t,
		testbuild.Decl{
			Src: `
type someInt interface{ int32 | int64 }

func cast[T, S someInt](val S) T {
	return T(val)
}

func callCast() int32 {
	return cast[int32, int64](2)
}
`,
			Want: []ir.IR{
				someInt,
				castAtomFunc,
				&ir.FuncDecl{
					FType: irhelper.FuncType(
						nil, nil,
						irhelper.Fields(),
						irhelper.Fields(ir.Int32Type()),
					),
					Body: &ir.BlockStmt{List: []ir.Stmt{
						&ir.ReturnStmt{Results: []ir.Expr{&ir.FuncCallExpr{
							Callee: irhelper.FuncExpr(castAtomFunc).NewFType(
								irhelper.FuncType(
									irhelper.Fields(
										irhelper.Field("T", someInt, nil),
										irhelper.Field("S", someInt, nil),
									), nil,
									irhelper.Fields("val", ir.Int64Type()),
									irhelper.Fields(ir.Int32Type()),
									irhelper.SetTypeParams(ir.Int32Type(), ir.Int64Type()),
								)),
							Args: []ir.Expr{
								irhelper.IntNumberAs(2, ir.Int64Type()),
							},
						}}},
					}},
				},
			},
		},
		testbuild.Decl{
			Src: `
type someInt interface{ int32 | int64 }

func cast[T, S someInt](val S) T {
	return T(val)
}

func callCast() int32 {
	return cast[int32](int64(2))
}
`,
			Want: []ir.IR{
				someInt,
				castAtomFunc,
				&ir.FuncDecl{
					FType: irhelper.FuncType(
						nil, nil,
						irhelper.Fields(),
						irhelper.Fields(ir.Int32Type()),
					),
					Body: &ir.BlockStmt{List: []ir.Stmt{
						&ir.ReturnStmt{Results: []ir.Expr{&ir.FuncCallExpr{
							Callee: irhelper.FuncExpr(castAtomFunc).NewFType(
								irhelper.FuncType(
									irhelper.Fields(
										irhelper.Field("T", someInt, nil),
										irhelper.Field("S", someInt, nil),
									), nil,
									irhelper.Fields("val", ir.Int64Type()),
									irhelper.Fields(ir.Int32Type()),
									irhelper.SetTypeParams(ir.Int32Type(), ir.Int64Type()),
								)),
							Args: []ir.Expr{
								&ir.CastExpr{
									X:   irhelper.IntNumberAs(2, ir.Int64Type()),
									Typ: ir.Int64Type(),
								},
							},
						}}},
					}},
				},
			},
		},
	)
}

func TestGenericCastArray(t *testing.T) {
	someInt := &ir.NamedType{
		Src:        &ast.TypeSpec{Name: irhelper.IdentAST("someInt")},
		File:       wantFile,
		Underlying: ir.TypeExpr(nil, irhelper.TypeSet(ir.Int32Type(), ir.Int64Type())),
	}
	typeParamM := irhelper.Field("M", ir.IntSliceType(), nil)
	typeParams := irhelper.Fields("T", "S", someInt, typeParamM)
	typeParamT := ir.NewGenericTypeParam(typeParams.List[0].Fields[0])
	typeParamS := ir.NewGenericTypeParam(typeParams.List[0].Fields[1])
	castArrayFunc := &ir.FuncBuiltin{
		Src: &ast.FuncDecl{Name: &ast.Ident{Name: "cast"}},
		FType: irhelper.FuncType(
			typeParams,
			nil,
			irhelper.Fields(irhelper.ArrayType(typeParamS, irhelper.Axis(irhelper.UnpackAxes(typeParamM)))),
			irhelper.Fields(irhelper.ArrayType(typeParamT, irhelper.Axis(irhelper.UnpackAxes(typeParamM)))),
		),
	}
	xField := irhelper.Field("x", irhelper.ArrayType(ir.Int64Type(), 2), nil)
	testbuild.Run(t,
		testbuild.Decl{
			Src: `
type someInt interface{ int32 | int64 }

func cast[T, S someInt, M []int]([unpack(M)]S) [unpack(M)]T

func callCast(x [2]int64) [2]int32 {
	return cast[int32, int64](x)
}
`,
			Want: []ir.IR{
				someInt,
				castArrayFunc,
				&ir.FuncDecl{
					FType: irhelper.FuncType(
						nil, nil,
						irhelper.Fields("x", irhelper.ArrayType(ir.Int64Type(), irhelper.Axis(2))),
						irhelper.Fields(irhelper.ArrayType(ir.Int32Type(), irhelper.Axis(2))),
					),
					Body: &ir.BlockStmt{List: []ir.Stmt{
						&ir.ReturnStmt{Results: []ir.Expr{&ir.FuncCallExpr{
							Callee: irhelper.FuncExpr(castArrayFunc).NewFType(
								irhelper.FuncType(
									irhelper.Fields(
										irhelper.Field("T", someInt, nil),
										irhelper.Field("S", someInt, nil),
										irhelper.Field("M", ir.IntSliceType(), nil),
									), nil,
									irhelper.Fields(irhelper.ArrayType(ir.Int64Type(), 2)),
									irhelper.Fields(irhelper.ArrayType(ir.Int32Type(), 2)),
									irhelper.SetTypeParams(
										ir.Int32Type(),
										ir.Int64Type(),
										[]int{2}),
								),
							),
							Args: []ir.Expr{
								irhelper.Ident(xField.Storage()),
							},
						}}},
					}},
				},
			},
		},
		testbuild.Decl{
			Src: `
type someInt interface{ int32 | int64 }

func cast[T someInt](val T) T {
	return val
}

func callCast() int32 {
	return cast(float32(2)) // ERROR float32 does not satisfy someInt
}
`,
		},
	)

}

func TestGenericExpression(t *testing.T) {
	testbuild.Run(t,
		testbuild.Decl{
			Src: `
type Floats interface {
	float32 | float64
}

func F[T Floats](x T) T {
	return 2*x
}
`,
		},
	)
}

func TestGenericWithInference(t *testing.T) {
	testbuild.Run(t,
		testbuild.Decl{
			Src: `
type Floats interface {
	float32 | float64
}

func F[T Floats, M []int](x [unpack(M)]T) [unpack(M)]T {
	return 2*x
}

func f() float32 {
	return F(float32(2))
}
`,
		},
		testbuild.Decl{
			Src: `
type Floats interface {
	float32 | float64
}

func F[T Floats, M []int](x [unpack(M)]T) [unpack(M)]T {
	return 2*x
}

func f() [2]float32 {
	return F([...]float32{1, 2})
}
`,
		},
	)
}

func TestGenericCallGeneric(t *testing.T) {
	testbuild.Run(t,
		testbuild.Decl{
			Src: `
type floats interface {
	float32 | float64
}

func f[T floats](x T) T {
	return 2*x
}

func g[T floats](x T) T {
	return f[T](x)
}
`,
		},
	)
}

func TestGenericFuncLit(t *testing.T) {
	testbuild.Run(t,
		testbuild.Decl{
			Src: `
type floats interface {
	float32 | float64
}

func f[T floats](x T) func () T {
	return func() T {
		return x
	}
}

func g(x float32) float32 {
	return f[float32](x)()
}
`,
		},
		testbuild.Decl{
			Src: `
type floats interface {
	float32 | float64
}

func f[T floats, M []int](x [unpack(M)]T) func () [unpack(M)]T {
	return func() [unpack(M)]T {
		return x
	}
}

func g(x float32) float32 {
	return f[float32](x)()
}
`,
		},
		testbuild.Decl{
			Src: `
type floats interface {
	float32 | float64
}

func f[T floats, M []int]([unpack(M)]T) [unpack(M)]T

func reverse[T floats, M []int](par [unpack(M)]T) func(res [unpack(M)]T) [unpack(M)]T {
	selfVJPFunc := func(res [unpack(M)]T) [unpack(M)]T {
		return f(par)
	}
	return selfVJPFunc
}
`,
		},
	)
}

func TestGenericErrors(t *testing.T) {
	testbuild.Run(t,
		testbuild.Decl{
			Src: `
type Floats interface {
	float32 | float // ERROR undefined: float
}
`,
		},
		testbuild.Decl{
			Src: `
func F[t interface{}](a [BatchSize]T) [BatchSize]T // ERROR undefined: T
`,
			Err: "undefined: T",
		},
		testbuild.Decl{
			Src: `
func F[T whatisai.X]() T // ERROR undefined: whatisai
`,
		},
		testbuild.Decl{
			Src: `
func f[T interface{ float32 | float64 }]() T {
	T := 3 // ERROR cannot assign to T
	return T // ERROR T (type) is not an expression
}
`,
		},
		testbuild.Decl{
			Src: `
func f[T interface{ float32 | float64 }]() T {
	T = 3 // ERROR cannot assign to T
	return T // ERROR T (type) is not an expression
}
`,
		},
		testbuild.Decl{
			Src: `
func f[T AgeOfTheCaptain](shape []int) [unpack(shape)]T // ERROR undefined: AgeOfTheCaptain
`,
			Err: "array of T not supported",
		},
		testbuild.Decl{
			Src: `
type Floats interface {
	float32 | float64
}

func f[T Floats]() T

func g() float32 {
	return f[2]() // ERROR 2 is not a type
}
`,
		},
	)

}
