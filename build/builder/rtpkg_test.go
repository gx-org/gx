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
	"testing"

	"github.com/gx-org/gx/build/builder/testbuild"
	"github.com/gx-org/gx/build/ir"
	irh "github.com/gx-org/gx/build/ir/irhelper"
	"github.com/gx-org/gx/src/gx/ast"
)

func TestRuntimePackageCall(t *testing.T) {
	testbuild.Run(t,
		testbuild.Decl{
			Src: `
import "runtime:hello"

func F() int {
	return hello.Hello()
}
`,
			Want: []ir.IR{
				&ir.FuncDecl{
					FType: irh.FuncType(
						nil, nil,
						irh.Fields(),
						irh.Fields(ir.IntType()),
					),
					Body: irh.SingleReturn(&ir.FuncCallExpr{
						Callee: ir.NewFuncValExpr(
							&ir.SelectorExpr{
								X: &ir.PackageRef{
									X: &ir.Ident{
										Src: &ast.Ident{
											Name: "hello",
										},
									},
								},
								Stor: &ir.LocalVarStorage{
									Src: &ast.Ident{
										Name: "Hello",
									},
									Typ: ir.UnknownType(),
								},
							},
							irh.UnknownFunc(),
						),
					}),
				},
			},
		},
	)
}
