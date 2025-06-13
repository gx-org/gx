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

package grad_test

import (
	"testing"

	_ "embed"

	"github.com/gx-org/gx/build/builder/testbuild"
	"github.com/gx-org/gx/build/ir"
	irh "github.com/gx-org/gx/build/ir/irhelper"
	"github.com/gx-org/gx/internal/interp/compeval/cpevelements"
	"github.com/gx-org/gx/stdlib/math/grad"
)

//go:embed grad.gx
var gradSrc []byte

func TestGradFunc(t *testing.T) {
	testbuild.Run(t,
		testbuild.DeclarePackage{
			Src: string(gradSrc),
			Post: func(pkg *ir.Package) {
				id := pkg.FindFunc("Func").(*ir.Macro)
				id.BuildSynthetic = cpevelements.MacroImpl(grad.FuncGrad)
			},
		},
		testbuild.Decl{
			Src: `
import "grad"

//gx:=grad.Func(f, "x")
func fGrad()

func f(x float32) float32 {
	return 2.0
}
`,
			Want: []ir.Node{
				&ir.FuncDecl{
					FType: irh.FuncType(
						nil, nil,
						irh.Fields("x", ir.Float32Type()),
						irh.Fields(ir.Float32Type()),
					),
					Body: irh.Block(
						&ir.ReturnStmt{
							Results: []ir.Expr{
								irh.IntNumberAs(0, ir.Float32Type()),
							},
						},
					)},
				&ir.FuncDecl{
					FType: irh.FuncType(
						nil, nil,
						irh.Fields("x", ir.Float32Type()),
						irh.Fields(ir.Float32Type()),
					),
					Body: irh.Block(
						&ir.ReturnStmt{
							Results: []ir.Expr{
								irh.FloatNumberAs(2, ir.Float32Type()),
							},
						},
					)},
			},
		},
	)
}
