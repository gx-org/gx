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

func TestNamedTypes(t *testing.T) {
	aField := irh.Field("a", ir.Int32Type(), nil)
	bField := irh.Field("b", ir.Float32Type(), nil)
	cField := irh.Field("c", ir.Float64Type(), nil)
	testbuild.Run(t,
		testbuild.Decl{
			Src: `type A int`,
			Want: []ir.IR{&ir.NamedType{
				Src:        &ast.TypeSpec{Name: irh.IdentAST("A")},
				Underlying: ir.TypeExpr(nil, ir.IntType()),
			}},
		},
		testbuild.Decl{
			Src: `type B interface {
				float32 | float64
			}`,
			Want: []ir.IR{&ir.NamedType{
				Src: &ast.TypeSpec{Name: irh.IdentAST("B")},
				Underlying: ir.TypeExpr(nil, irh.TypeSet(
					ir.Float32Type(),
					ir.Float64Type(),
				))}},
		},
		testbuild.Decl{
			Src: `type A struct {
				a int32
				b float32
				c float64
			}`,
			Want: []ir.IR{&ir.NamedType{
				Src: &ast.TypeSpec{Name: irh.IdentAST("A")},
				Underlying: ir.TypeExpr(nil, irh.StructType(
					aField, bField, cField,
				))}},
		},
		testbuild.Decl{
			Src: `type A [2][3]float32`,
			Want: []ir.IR{&ir.NamedType{
				Src: &ast.TypeSpec{Name: irh.IdentAST("A")},
				Underlying: ir.TypeExpr(nil, irh.ArrayType(
					ir.Float32Type(),
					irh.IntNumberAs(2, ir.IntType()),
					irh.IntNumberAs(3, ir.IntType()),
				)),
			}},
		},
		testbuild.Decl{
			Src: `
type matrix [2][3]float32

func (x matrix) top() [3]float32 {
   return x[0]
}
`,
		},
	)
	typeA := &ir.NamedType{
		File:       wantFile,
		Src:        &ast.TypeSpec{Name: irh.IdentAST("A")},
		Underlying: ir.TypeExpr(nil, ir.Float32Type()),
	}
	testbuild.Run(t,
		testbuild.Decl{
			Src: `
type A float32

func f() [2][3]A
`,
			Want: []ir.IR{
				typeA,
				&ir.FuncBuiltin{
					FType: irh.FuncType(
						nil, nil,
						irh.Fields(),
						irh.Fields(irh.ArrayType(typeA,
							irh.IntNumberAs(2, ir.IntType()),
							irh.IntNumberAs(3, ir.IntType()),
						))),
				},
			},
		},
	)
}

func TestNamedTypeErrors(t *testing.T) {
	testbuild.Run(t,
		testbuild.Decl{
			Src: `
type A float32

func f(x float32, y A) float32 {
	return x+y // ERROR mismatched types A and float32
}
`,
		},
		testbuild.Decl{
			Src: `
type A struct {
	a [2]ageOfTheCaptain // ERROR undefined: ageOfTheCaptain
}

func f() A {
	return A{
		a: [2]float32{2, 3},
	}
}
`,
		},
		testbuild.DeclarePackage{
			Src: `
package pkg
`,
		},
		testbuild.Decl{
			Src: `
import "pkg"

type A struct {
	a [2]pkg // ERROR pkg is not a type
}

func f() A {
	return A{
		a: [2]float32{2, 3},
	}
}
`,
		},
		testbuild.Decl{
			Src: `
type A struct {
	a [2]ageOfTheCaptain // ERROR undefined: ageOfTheCaptain
}

func g([2]float32) [2]float32

func f(a A) A {
	a.a = g(a.a)
	return a
}
`,
		},
	)
}
