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

func TestLocalAssignment(t *testing.T) {
	idFunc := &ir.FuncBuiltin{
		Src: &ast.FuncDecl{Name: &ast.Ident{Name: "id"}},
		FType: irh.FuncType(
			nil, nil,
			irh.Fields("x", ir.Float32Type()),
			irh.Fields(ir.Float32Type()),
		),
	}
	xField := irh.Field("x", ir.Float32Type(), nil)
	yField := irh.Field("y", ir.Float32Type(), nil)
	testbuild.Run(t,
		testbuild.Decl{
			Src: `
func id(x float32) float32

func call(x float32, y float32) float32 {
	return id(x/y)
}
`,
			Want: []ir.IR{
				idFunc,
				&ir.FuncDecl{
					FType: irh.FuncType(
						nil, nil,
						irh.Fields(xField, yField),
						irh.Fields(ir.Float32Type()),
					),
					Body: irh.SingleReturn(
						&ir.FuncCallExpr{
							Callee: irh.FuncExpr(idFunc),
							Args: []ir.Expr{&ir.BinaryExpr{
								X:   irh.ValueRef(xField.Storage()),
								Y:   irh.ValueRef(yField.Storage()),
								Typ: ir.Float32Type(),
							}},
						},
					),
				},
			},
		},
	)
}
