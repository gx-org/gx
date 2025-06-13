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
		Src:        &ast.TypeSpec{Name: irh.Ident("A")},
		Underlying: irh.TypeExpr(structA),
	}
	testbuild.Run(t,
		testbuild.DeclTest{
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
			Want: []ir.Node{
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
	)
}
