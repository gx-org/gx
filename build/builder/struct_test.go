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

func TestStruct(t *testing.T) {
	aField := irh.Field("a", ir.Int32Type(), nil)
	bField := irh.Field("b", ir.Float32Type(), nil)
	structA := irh.StructType(aField, bField)
	typeA := &ir.NamedType{
		File:       wantFile,
		Src:        &ast.TypeSpec{Name: irh.IdentAST("A")},
		Underlying: ir.TypeExpr(nil, structA),
	}
	testbuild.Run(t,
		testbuild.Decl{
			Src: `
type A struct {
	a int32
	b float32
}

func a() A {
	return A{
		a: 1,
		b: 2,
	}
}
`,
			Want: []ir.IR{
				typeA,
				&ir.FuncDecl{
					FType: irh.FuncType(
						nil, nil,
						irh.Fields(),
						irh.Fields(typeA),
					),
					Body: irh.Block(
						&ir.ReturnStmt{
							Results: []ir.Expr{&ir.StructLitExpr{
								Elts: []*ir.FieldLit{
									irh.FieldLit(structA.Fields, "a", irh.IntNumberAs(1, ir.Int32Type())),
									irh.FieldLit(structA.Fields, "b", irh.IntNumberAs(2, ir.Float32Type())),
								},
								Typ: typeA,
							}},
						},
					)},
			},
		},
		testbuild.Decl{
			Src: `
type S struct {
	Field int32
}

func g(a int32) int32

func f(s []S) int32 {
	return g(s[0].Field)
}
`,
		},
		testbuild.Decl{
			Src: `
type (
	Container struct {
		els []Element
	}
	
	Element struct {
		field int32
	}
)



func g(a int32) int32

func f(s []Container) int32 {
	return g(s[0].els[0].field)	
}
`,
		},
		testbuild.Decl{
			Src: `
type S struct {
	F int32
}

func New() S {
	return S{F:2}
}

func CallNew() S {
	return New()
}
`,
		},
		testbuild.Decl{
			Src: `
type S struct {
	F int32
	int32
}

func F(s S) S
`,
		},
		testbuild.Decl{
			Src: `
type S struct {
	F int32
	int32
}

func F() S {
	return S{F:0}
}
`,
		},
		testbuild.Decl{
			Src: `
type S struct {
	a float32 
}

func g(s S) S

func f(s S) S {
	return g(s)
}
`,
		},
		testbuild.Decl{
			Src: `
type S struct {
	a float32
}

func (s S) New[Shape []intlen]() [Shape]float32 {
	return [Shape]float32(s.a)
}
`,
		},
		testbuild.Decl{
			Src: `
type S struct {
	a float32
}

func (s S) New[Shape []intlen]() [Shape]float32 {
	return [Shape]float32(s.a)
}

func F() [3][2]float32 {
	s := S{a:4}
	return s.New[[]intlen{3,2}]()
}
`,
		},
	)
}

func TestStructErrors(t *testing.T) {
	testbuild.Run(t,
		testbuild.Decl{
			Src: `
type A struct {
	a int32
	b float32
}

func a() A {
	return A // ERROR A (type) is not an expression
}
`,
		},
	)
}
