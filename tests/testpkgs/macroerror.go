// Copyright 2026 Google LLC
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

package testpkgs

import (
	"github.com/gx-org/gx/build/builder/testbuild"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/interp/elements"
	"github.com/gx-org/gx/src/gx/ast"
)

// TestMacroError declares a macro package that generates GX code with a compile error.
var TestMacroError = testbuild.DeclarePackage{
	Src: `
package testmacroerror

//gx:irmacro
func Error() any
`,
	Post: func(pkg *ir.Package) {
		errMacro := pkg.FindFunc("Error").(*ir.Macro)
		errMacro.BuildSynthetic = ir.MacroImpl(newMacroError)
	},
}

type macroError struct {
	elements.MacroCall
	file *ir.File
}

var _ ir.FuncASTBuilder = (*macroError)(nil)

func newMacroError(file *ir.File, call *ir.FuncCallExpr, mac *ir.Macro, args []ir.Element) (ir.MacroElement, error) {
	return &macroError{
		MacroCall: elements.NewMacroCall(mac, file, call),
		file:      file,
	}, nil
}

func (m *macroError) BuildDecl(ir.PkgFunc) (*ir.File, *ast.FuncDecl, bool) {
	return m.file, &ast.FuncDecl{
		Type: &ast.FuncType{
			Params: &ast.FieldList{},
			Results: &ast.FieldList{
				List: []*ast.Field{
					{Type: &ast.Ident{Name: "int32"}},
				},
			},
		},
	}, true
}

func (m *macroError) BuildBody(ir.Fetcher, ir.Func) (*ast.BlockStmt, bool) {
	return &ast.BlockStmt{
		List: []ast.Stmt{
			&ast.ReturnStmt{
				Results: []ast.Expr{
					&ast.Ident{Name: "undefinedIdentifier"},
				},
			},
		},
	}, true
}
