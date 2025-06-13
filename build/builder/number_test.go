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

func TestNumber(t *testing.T) {
	valField := irh.Field("val", ir.Float32Type(), nil)
	typeA := &ir.NamedType{
		File:       wantFile,
		Src:        &ast.TypeSpec{Name: irh.Ident("A")},
		Underlying: irh.TypeExpr(irh.StructType(valField)),
	}
	typeA.Methods = []ir.PkgFunc{
		&ir.FuncDecl{
			FType: irh.FuncType(
				nil,
				irh.Fields("a", typeA),
				irh.Fields(),
				irh.Fields(ir.Float32Type()),
			),
			Body: irh.Block(
				&ir.AssignExprStmt{List: []*ir.AssignExpr{{
					X: irh.FloatNumberAs(0, ir.Float32Type()),
					Storage: &ir.StructFieldStorage{Sel: &ir.SelectorExpr{
						X:    irh.ValueRef(typeA),
						Stor: valField.Storage(),
					}},
				}}},
				&ir.ReturnStmt{Results: []ir.Expr{
					&ir.SelectorExpr{
						X:    irh.ValueRef(typeA),
						Stor: valField.Storage(),
					},
				}},
			),
		},
	}
	testbuild.Run(t,
		testbuild.DeclTest{
			Src: `
func id(int64) int64

func f() int64 {
	a := 2
	return id(a)
}
`,
		},
		testbuild.DeclTest{
			Src: `
type A struct {
	val float32
}

func (a A) f() float32 {
	a.val = 0.0	
	return a.val
}
`,
		},
	)
}
