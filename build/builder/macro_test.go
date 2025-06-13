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
	"github.com/gx-org/gx/internal/interp/compeval/cpevelements"
	"github.com/gx-org/gx/interp/elements"
)

func newIDMacro(call elements.CallAt, macro *cpevelements.Macro, args []elements.Element) (*cpevelements.SyntheticFunc, error) {
	fn, err := elements.FuncDeclFromElement(args[0])
	if err != nil {
		return nil, err
	}
	return cpevelements.NewSyntheticFunc(macro, &idMacro{fn: fn}), nil
}

type idMacro struct {
	fn *ir.FuncDecl
}

func (m *idMacro) BuildType() (*ir.FuncType, error) {
	return m.fn.FType, nil
}

func (m *idMacro) BuildBody(fetcher ir.Fetcher) (*ir.BlockStmt, bool) {
	return m.fn.Body, true
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
		testbuild.DeclTest{
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
	)
}
