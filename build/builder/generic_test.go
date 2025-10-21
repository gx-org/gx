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
	testbuild.Run(t,
		testbuild.Decl{
			Src: `func f[S any](S) S`,
			Want: []ir.Node{
				&ir.FuncBuiltin{
					FType: irhelper.FuncType(
						irhelper.Fields(anyS),
						nil,
						irhelper.Fields(&ir.TypeParam{Field: anyS}),
						irhelper.Fields(&ir.TypeParam{Field: anyS}),
					),
				},
			},
		},
		testbuild.Decl{
			Src: `func f[T interface{int32|int64}](T) T`,
			Want: []ir.Node{
				&ir.FuncBuiltin{
					FType: irhelper.FuncType(
						irhelper.Fields(intsT),
						nil,
						irhelper.Fields(&ir.TypeParam{Field: intsT}),
						irhelper.Fields(&ir.TypeParam{Field: intsT}),
					),
				},
			},
		},
		testbuild.Decl{
			Src: `func f[S any, T interface{int32|int64}](S) T`,
			Want: []ir.Node{
				&ir.FuncBuiltin{
					FType: irhelper.FuncType(
						irhelper.Fields(anyS, intsT),
						nil,
						irhelper.Fields(&ir.TypeParam{Field: anyS}),
						irhelper.Fields(&ir.TypeParam{Field: intsT}),
					),
				},
			},
		},
		testbuild.Decl{
			Src: `func f(x [_X][_Y]int32) ([X]int32, [Y]int32)`,
			Want: []ir.Node{
				&ir.FuncBuiltin{
					FType: irhelper.FuncType(
						nil, nil,
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
			Src: `func f(x [_X][_X]int32) [X]int32`,
			Err: "axis length X assignment repeated",
		},
		testbuild.Decl{
			Src: `func f([___X]int32) [X___]int32`,
			Want: []ir.Node{
				&ir.FuncBuiltin{
					FType: irhelper.FuncType(
						nil, nil,
						irhelper.Fields(
							"",
							irhelper.ArrayType(
								ir.Int32Type(),
								irhelper.Axis("___X"),
							)),
						irhelper.Fields(
							"",
							irhelper.ArrayType(
								ir.Int32Type(),
								irhelper.Axis("X___"),
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
		Src:        &ast.TypeSpec{Name: irhelper.Ident("someInt")},
		File:       wantFile,
		Underlying: irhelper.TypeExpr(irhelper.TypeSet(ir.Int32Type(), ir.Int64Type())),
	}
	someIntT := irhelper.Field("T", someInt, nil)
	someIntTP := &ir.TypeParam{Field: someIntT}
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
			Want: []ir.Node{
				someInt,
				castNoArgFunc,
				&ir.FuncDecl{
					FType: irhelper.FuncType(
						nil, nil,
						irhelper.Fields(),
						irhelper.Fields(ir.Int32Type()),
					),
					Body: &ir.BlockStmt{List: []ir.Stmt{
						&ir.ReturnStmt{Results: []ir.Expr{&ir.CallExpr{
							Callee: &ir.FuncValExpr{
								X: &ir.FuncValExpr{
									X: irhelper.ValueRef(castNoArgFunc),
									F: &ir.SpecialisedFunc{
										X: irhelper.ValueRef(castNoArgFunc),
										F: &ir.FuncValExpr{
											X: irhelper.ValueRef(castNoArgFunc),
											F: castNoArgFunc,
											T: castNoArgFunc.FType,
										},
										T: castNoArgFunc.FType,
									},
									T: castNoArgFunc.FType,
								},
								F: &ir.SpecialisedFunc{
									X: irhelper.ValueRef(castNoArgFunc),
									F: &ir.FuncValExpr{
										X: irhelper.ValueRef(castNoArgFunc),
										F: castNoArgFunc,
										T: castNoArgFunc.FType,
									},
									T: castNoArgFunc.FType,
								},
								T: castNoArgFunc.FType,
							},
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
type someInt interface{ int32 | int64 }

func f[T someInt](x [___X]T) [X___]T {
	y := g[T](x)
	return x / y
}

func g[T someInt](x [___X]T) T
`,
		},
		testbuild.Decl{
			Src: `
import "dtype"

func f[T dtype.Num](x [___X]T) [X___]T {
	y := g[T](x)
	return x / y
}

func g[T dtype.Num](x [___X]T) T
`,
		},
	)
}

func TestGenericArray(t *testing.T) {
	someInt := &ir.NamedType{
		Src:        &ast.TypeSpec{Name: irhelper.Ident("someInt")},
		File:       wantFile,
		Underlying: irhelper.TypeExpr(irhelper.TypeSet(ir.Int32Type(), ir.Int64Type())),
	}
	typeParamFieldT := irhelper.Field("T", someInt, nil)
	typeParamT := &ir.TypeParam{Field: typeParamFieldT}
	new2x3ArrayFunc := &ir.FuncDecl{
		Src: &ast.FuncDecl{Name: &ast.Ident{Name: "new2x3Array"}},
		FType: irhelper.FuncType(
			irhelper.Fields(typeParamFieldT),
			nil,
			irhelper.Fields(),
			irhelper.Fields(irhelper.ArrayType(someInt, 2, 3)),
		),
		Body: &ir.BlockStmt{List: []ir.Stmt{
			&ir.ReturnStmt{Results: []ir.Expr{&ir.ArrayLitExpr{
				Typ: irhelper.ArrayType(someInt, 2, 3),
				Elts: []ir.AssignableExpr{
					&ir.ArrayLitExpr{
						Typ: irhelper.ArrayType(someInt, 3),
						Elts: []ir.AssignableExpr{
							irhelper.IntNumberAs(1, typeParamT),
							irhelper.IntNumberAs(2, typeParamT),
							irhelper.IntNumberAs(3, typeParamT),
						},
					},
					&ir.ArrayLitExpr{
						Typ: irhelper.ArrayType(someInt, 3),
						Elts: []ir.AssignableExpr{
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
			Want: []ir.Node{
				someInt,
				new2x3ArrayFunc,
				&ir.FuncDecl{
					FType: irhelper.FuncType(
						nil, nil,
						irhelper.Fields(),
						irhelper.Fields(irhelper.ArrayType(ir.Int32Type(), 2, 3)),
					),
					Body: &ir.BlockStmt{List: []ir.Stmt{
						&ir.ReturnStmt{Results: []ir.Expr{&ir.CallExpr{
							Callee: &ir.FuncValExpr{
								X: &ir.FuncValExpr{
									X: irhelper.ValueRef(new2x3ArrayFunc),
									F: &ir.SpecialisedFunc{
										X: irhelper.ValueRef(new2x3ArrayFunc),
										F: &ir.FuncValExpr{
											X: irhelper.ValueRef(new2x3ArrayFunc),
											F: new2x3ArrayFunc,
											T: new2x3ArrayFunc.FType,
										},
										T: new2x3ArrayFunc.FType,
									},
									T: new2x3ArrayFunc.FType,
								},
								F: &ir.SpecialisedFunc{
									X: irhelper.ValueRef(new2x3ArrayFunc),
									F: &ir.FuncValExpr{
										X: irhelper.ValueRef(new2x3ArrayFunc),
										F: new2x3ArrayFunc,
										T: new2x3ArrayFunc.FType,
									},
									T: new2x3ArrayFunc.FType,
								},
								T: irhelper.FuncType(
									irhelper.Fields(), nil,
									irhelper.Fields(),
									irhelper.Fields(irhelper.ArrayType(ir.Int32Type(), 2, 3)),
								),
							},
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
	return cast[cast]()
}
`,
			Err: "cast not a type",
		},
		testbuild.Decl{
			Src: `
type someInt interface{ int32 | int64 }

func cast[T someInt]() T {
	return T(2)
}

func callCast() int32 {
	return cast[float32]()
}
`,
			Err: "float32 does not satisfy test.someInt",
		},
	)
}

func TestGenericConvert(t *testing.T) {
	someInt := &ir.NamedType{
		Src:        &ast.TypeSpec{Name: irhelper.Ident("someInt")},
		File:       wantFile,
		Underlying: irhelper.TypeExpr(irhelper.TypeSet(ir.Int32Type(), ir.Int64Type())),
	}
	someIntT := irhelper.Field("T", someInt, nil)
	someIntTP := &ir.TypeParam{Field: someIntT}
	someIntS := irhelper.Field("S", someInt, someIntT.Group)
	valField := irhelper.Field("val", &ir.TypeParam{Field: someIntS}, nil)
	castAtomFunc := &ir.FuncDecl{
		Src: &ast.FuncDecl{Name: &ast.Ident{Name: "cast"}},
		FType: irhelper.FuncType(
			irhelper.Fields(someIntT.Group),
			nil,
			irhelper.Fields("val", &ir.TypeParam{Field: someIntS}),
			irhelper.Fields(someIntTP),
		),
		Body: &ir.BlockStmt{List: []ir.Stmt{
			&ir.ReturnStmt{Results: []ir.Expr{
				&ir.CastExpr{Typ: someIntTP, X: irhelper.ValueRef(valField.Storage())},
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
			Want: []ir.Node{
				someInt,
				castAtomFunc,
				&ir.FuncDecl{
					FType: irhelper.FuncType(
						nil, nil,
						irhelper.Fields(),
						irhelper.Fields(ir.Int32Type()),
					),
					Body: &ir.BlockStmt{List: []ir.Stmt{
						&ir.ReturnStmt{Results: []ir.Expr{&ir.CallExpr{
							Callee: &ir.FuncValExpr{
								X: irhelper.ValueRef(castAtomFunc),
								F: castAtomFunc,
								T: irhelper.FuncType(
									irhelper.Fields(), nil,
									irhelper.Fields(ir.Int64Type()),
									irhelper.Fields(ir.Int32Type()),
								),
							},
							Args: []ir.AssignableExpr{
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
			Want: []ir.Node{
				someInt,
				castAtomFunc,
				&ir.FuncDecl{
					FType: irhelper.FuncType(
						nil, nil,
						irhelper.Fields(),
						irhelper.Fields(ir.Int32Type()),
					),
					Body: &ir.BlockStmt{List: []ir.Stmt{
						&ir.ReturnStmt{Results: []ir.Expr{&ir.CallExpr{
							Callee: &ir.FuncValExpr{
								X: irhelper.ValueRef(castAtomFunc),
								F: castAtomFunc,
								T: irhelper.FuncType(
									irhelper.Fields(), nil,
									irhelper.Fields(ir.Int64Type()),
									irhelper.Fields(ir.Int32Type()),
								),
							},
							Args: []ir.AssignableExpr{
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
		Src:        &ast.TypeSpec{Name: irhelper.Ident("someInt")},
		File:       wantFile,
		Underlying: irhelper.TypeExpr(irhelper.TypeSet(ir.Int32Type(), ir.Int64Type())),
	}
	castArrayFunc := &ir.FuncBuiltin{
		Src: &ast.FuncDecl{Name: &ast.Ident{Name: "cast"}},
		FType: irhelper.FuncType(
			irhelper.Fields("T", "S", someInt),
			nil,
			irhelper.Fields(irhelper.ArrayType(someInt, irhelper.Axis("___M"))),
			irhelper.Fields(irhelper.ArrayType(someInt, irhelper.Axis("M___"))),
		),
	}
	xField := irhelper.Field("x", irhelper.ArrayType(ir.Int64Type(), 2), nil)
	testbuild.Run(t,
		testbuild.Decl{
			Src: `
type someInt interface{ int32 | int64 }

func cast[T, S someInt]([___M]S) [M___]T

func callCast(x [2]int64) [2]int32 {
	return cast[int32, int64](x)
}
`,
			Want: []ir.Node{
				someInt,
				castArrayFunc,
				&ir.FuncDecl{
					FType: irhelper.FuncType(
						nil, nil,
						irhelper.Fields("x", irhelper.ArrayType(ir.Int64Type(), irhelper.Axis(2))),
						irhelper.Fields(irhelper.ArrayType(ir.Int32Type(), irhelper.Axis(2))),
					),
					Body: &ir.BlockStmt{List: []ir.Stmt{
						&ir.ReturnStmt{Results: []ir.Expr{&ir.CallExpr{
							Callee: &ir.FuncValExpr{
								X: irhelper.ValueRef(castArrayFunc),
								F: castArrayFunc,
								T: irhelper.FuncType(
									irhelper.Fields(), nil,
									irhelper.Fields(irhelper.ArrayType(ir.Int64Type(), irhelper.Axis("___M"))),
									irhelper.Fields(irhelper.ArrayType(ir.Int32Type(), 2)),
								),
							},
							Args: []ir.AssignableExpr{
								irhelper.ValueRef(xField.Storage()),
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
	return cast(float32(2))
}
`,
			Err: "float32 does not satisfy test.someInt",
		},
	)

}

func TestGenericErrors(t *testing.T) {
	testbuild.Run(t,
		testbuild.Decl{
			Src: `
type Floats interface {
	float32 | float
}
`,
			Err: "undefined: float",
		},
		testbuild.Decl{
			Src: `
func F[t interface{}](a [BatchSize]T) [BatchSize]T
`,
			Err: "undefined: T",
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
