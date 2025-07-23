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
	"github.com/gx-org/gx/internal/interp/compeval/cpevelements"
	"github.com/gx-org/gx/interp/elements"
	"github.com/gx-org/gx/interp"
)

func newIDMacro(call elements.CallAt, macro *cpevelements.Macro, args []ir.Element) (*cpevelements.SyntheticFunc, error) {
	fn, err := interp.FuncDeclFromElement(args[0])
	if err != nil {
		return nil, err
	}
	return cpevelements.NewSyntheticFunc(&idMacro{fn: fn}), nil
}

type idMacro struct {
	fn *ir.FuncDecl
}

func (m *idMacro) BuildType() (*ast.FuncDecl, error) {
	return &ast.FuncDecl{Type: m.fn.FType.Src}, nil
}

func (m *idMacro) BuildBody(fetcher ir.Fetcher) (*ast.BlockStmt, []*cpevelements.SyntheticFuncDecl, bool) {
	return m.fn.Body.Src, nil, true
}

func TestMacro(t *testing.T) {
	testbuild.Run(t,
		testbuild.DeclarePackage{
			Src: `
package macro 

//gx:irmacro
func ID(any) any
`,
			Post: func(pkg *ir.Package) {
				id := pkg.FindFunc("ID").(*ir.Macro)
				id.BuildSynthetic = cpevelements.MacroImpl(newIDMacro)
			},
		},
		testbuild.Decl{
			Src: `
import "macro"

//gx:=macro.ID(f)
func synthetic()

func f() int32 {
	return 2
}
`,
			Want: []ir.Node{
				&ir.FuncDecl{
					FType: irh.FuncType(
						nil, nil,
						irh.Fields(),
						irh.Fields(ir.Int32Type()),
					),
					Body: irh.Block(
						&ir.ReturnStmt{
							Results: []ir.Expr{
								irh.IntNumberAs(2, ir.Int32Type()),
							},
						},
					)},
				&ir.FuncDecl{
					FType: irh.FuncType(
						nil, nil,
						irh.Fields(),
						irh.Fields(ir.Int32Type()),
					),
					Body: irh.Block(
						&ir.ReturnStmt{
							Results: []ir.Expr{
								irh.IntNumberAs(2, ir.Int32Type()),
							},
						},
					)},
			},
		},
		testbuild.Decl{
			Src: `
import "macro"

//gx:=macro.ID(f)
func synthetic()

func f(x int32) int32 {
	return x
}
`,
		},
	)
}
