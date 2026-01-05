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

func TestMethods(t *testing.T) {
	fieldI := irh.Field("i", ir.Int32Type(), nil)
	fieldF := irh.Field("f", ir.Float32Type(), nil)
	structA := irh.StructType(fieldI, fieldF)
	typeA := &ir.NamedType{
		Src:        &ast.TypeSpec{Name: &ast.Ident{Name: "A"}},
		File:       wantFile,
		Underlying: ir.TypeExpr(nil, structA),
	}
	aRecv := irh.Fields("a", typeA)
	fI := &ir.FuncDecl{
		Src: &ast.FuncDecl{Name: &ast.Ident{Name: "fI"}},
		FType: irh.FuncType(nil,
			aRecv,
			irh.Fields(),
			irh.Fields(ir.Int32Type()),
		),
		Body: irh.Block(
			&ir.ReturnStmt{
				Results: []ir.Expr{&ir.SelectorExpr{
					X:    irh.ValueRef(aRecv.Fields()[0].Storage()),
					Stor: fieldI.Storage(),
				}},
			},
		)}
	typeA.Methods = []ir.PkgFunc{fI}
	testbuild.Run(t,
		testbuild.Decl{
			Src: `
type A struct {
	i int32
	f float32
}

func (a A) fI() int32 {
	return a.i
}
`,
			Want: []ir.IR{typeA},
		},
		testbuild.Decl{
			Src: `
type A struct {
	i int32
	f float32
}

func (a A) fI() int32 {
	return a.i
}

func call() int32 {
	a := A{i:2,f:3}
	return a.fI()
}
`,
			Want: []ir.IR{
				typeA,
				&ir.FuncDecl{
					FType: irh.FuncType(
						nil, nil,
						irh.Fields(),
						irh.Fields(ir.Int32Type()),
					),
					Body: irh.Block(
						&ir.AssignExprStmt{List: []*ir.AssignExpr{{
							X: &ir.StructLitExpr{
								Elts: []*ir.FieldLit{
									irh.FieldLit(structA.Fields, "i", irh.IntNumberAs(2, ir.Int32Type())),
									irh.FieldLit(structA.Fields, "f", irh.IntNumberAs(3, ir.Float32Type())),
								},
								Typ: typeA,
							},
							Storage: &ir.LocalVarStorage{Src: irh.Ident("a"), Typ: typeA},
						}}},
						&ir.ReturnStmt{
							Results: []ir.Expr{&ir.FuncCallExpr{
								Callee: ir.NewFuncValExpr(
									&ir.SelectorExpr{
										X:    irh.ValueRef(typeA),
										Stor: fI,
									},
									fI),
							}}},
					)},
			},
		},
		testbuild.Decl{
			Src: `
type (
	A struct {}
	
	B struct {
		a A
	}
)

func (A) f() int32

func (b B) f() int32 {
	return b.a.f()
}
`,
		},
	)
}

func TestMethodOnNamedTypes(t *testing.T) {
	typeA := &ir.NamedType{
		File:       wantFile,
		Src:        &ast.TypeSpec{Name: irh.Ident("A")},
		Underlying: ir.TypeExpr(nil, ir.Int32Type()),
	}
	aRecv := irh.Fields("a", typeA)
	val := &ir.FuncDecl{
		Src: &ast.FuncDecl{Name: &ast.Ident{Name: "val"}},
		FType: irh.FuncType(nil,
			aRecv,
			irh.Fields(),
			irh.Fields(ir.Int32Type()),
		),
		Body: irh.Block(
			&ir.ReturnStmt{
				Results: []ir.Expr{&ir.CastExpr{
					Typ: ir.Int32Type(),
					X:   irh.ValueRef(aRecv.Fields()[0].Storage()),
				}},
			},
		)}
	typeA.Methods = []ir.PkgFunc{val}
	testbuild.Run(t,
		testbuild.Decl{
			Src: `
type A int32

func (a A) val() int32 {
	return int32(a)
}

func call() int32 {
	a := A(2)
	return a.val()
}
`,
			Want: []ir.IR{
				typeA,
				&ir.FuncDecl{
					FType: irh.FuncType(
						nil, nil,
						irh.Fields(),
						irh.Fields(ir.Int32Type()),
					),
					Body: irh.Block(
						&ir.AssignExprStmt{List: []*ir.AssignExpr{{
							X: &ir.CastExpr{
								X:   irh.IntNumberAs(2, typeA),
								Typ: typeA,
							},
							Storage: &ir.LocalVarStorage{Src: irh.Ident("a"), Typ: typeA},
						}}},
						&ir.ReturnStmt{
							Results: []ir.Expr{&ir.FuncCallExpr{
								Callee: ir.NewFuncValExpr(
									&ir.SelectorExpr{
										X:    irh.ValueRef(typeA),
										Stor: val,
									},
									val,
								),
							}},
						},
					)},
			},
		},
		testbuild.Decl{
			Src: `
type A float32

func (a A) val() float32 {
	return float32(a)
}

func call() float32 {
	a := A(2.3)
	return a.val()
}
`,
		},
	)
}
