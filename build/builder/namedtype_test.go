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
		testbuild.DeclTest{
			Src: `type A intlen`,
			Want: []ir.Node{&ir.NamedType{
				Src:        &ast.TypeSpec{Name: irh.Ident("A")},
				Underlying: ir.AtomTypeExpr(ir.IntLenType()),
			}},
		},
		testbuild.DeclTest{
			Src: `type B interface {
				float32 | float64
			}`,
			Want: []ir.Node{&ir.NamedType{
				Src: &ast.TypeSpec{Name: irh.Ident("B")},
				Underlying: irh.TypeExpr(irh.TypeSet(
					ir.Float32Type(),
					ir.Float64Type(),
				))}},
		},
		testbuild.DeclTest{
			Src: `type A struct {
				a int32
				b float32
				c float64
			}`,
			Want: []ir.Node{&ir.NamedType{
				Src: &ast.TypeSpec{Name: irh.Ident("A")},
				Underlying: irh.TypeExpr(irh.StructType(
					aField, bField, cField,
				))}},
		},
		testbuild.DeclTest{
			Src: `type A [2][3]float32`,
			Want: []ir.Node{&ir.NamedType{
				Src: &ast.TypeSpec{Name: irh.Ident("A")},
				Underlying: irh.TypeExpr(irh.ArrayType(
					ir.Float32Type(),
					irh.IntNumberAs(2, ir.IntLenType()),
					irh.IntNumberAs(3, ir.IntLenType()),
				)),
			}},
		},
	)
	typeA := &ir.NamedType{
		File:       wantFile,
		Src:        &ast.TypeSpec{Name: irh.Ident("A")},
		Underlying: ir.AtomTypeExpr(ir.Float32Type()),
	}
	testbuild.Run(t,
		testbuild.DeclTest{
			Src: `
type A float32

func f() [2][3]A
`,
			Want: []ir.Node{
				typeA,
				&ir.FuncBuiltin{
					FType: irh.FuncType(
						nil, nil,
						irh.Fields(),
						irh.Fields(irh.ArrayType(typeA,
							irh.IntNumberAs(2, ir.IntLenType()),
							irh.IntNumberAs(3, ir.IntLenType()),
						))),
				},
			},
		},
	)
}
