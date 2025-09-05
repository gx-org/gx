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

type idMacro struct {
	cpevelements.CoreMacroElement
	fn *ir.FuncDecl
}

var _ cpevelements.FuncASTBuilder = (*idMacro)(nil)

func newIDMacro(call elements.CallAt, macro *cpevelements.Macro, args []ir.Element) (cpevelements.MacroElement, error) {
	fn, err := interp.FuncDeclFromElement(args[1])
	if err != nil {
		return nil, err
	}
	return &idMacro{CoreMacroElement: cpevelements.CoreMacroElement{Mac: macro}, fn: fn}, nil
}

func (m *idMacro) BuildDecl() (*ast.FuncDecl, bool) {
	return &ast.FuncDecl{Type: m.fn.FType.Src, Recv: m.fn.Src.Recv}, true
}

func (m *idMacro) BuildBody(fetcher ir.Fetcher, fn ir.Func) (*ast.BlockStmt, []*cpevelements.SyntheticFuncDecl, bool) {
	return m.fn.Body.Src, nil, true
}

var declareMacroPackage = testbuild.DeclarePackage{
	Src: `
package macro

//gx:irmacro
func ID(any, any) any
`,
	Post: func(pkg *ir.Package) {
		id := pkg.FindFunc("ID").(*ir.Macro)
		id.BuildSynthetic = cpevelements.MacroImpl(newIDMacro)
	},
}

func TestMacro(t *testing.T) {
	testbuild.Run(t,
		declareMacroPackage,
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

func TestMacroOnMethod(t *testing.T) {
	typeS := &ir.NamedType{
		File:       wantFile,
		Src:        &ast.TypeSpec{Name: irh.Ident("S")},
		Underlying: irh.TypeExpr(irh.StructType()),
	}
	fType := irh.FuncType(
		nil,
		irh.Fields(typeS),
		irh.Fields(),
		irh.Fields(ir.Int32Type()),
	)
	body := irh.SingleReturn(irh.IntNumberAs(2, ir.Int32Type()))
	typeS.Methods = []ir.PkgFunc{
		&ir.FuncDecl{
			FType: fType,
			Body:  body,
		},
		&ir.FuncDecl{
			FType: fType,
			Body:  body,
		},
	}
	testbuild.Run(t,
		declareMacroPackage,
		testbuild.Decl{
			Src: `
import "macro"

type S struct{}

//gx:=macro.ID(S.f)
func (S) synthetic()

func (S) f() int32 {
	return 2
}
`,
			Want: []ir.Node{
				typeS,
			},
		},
		testbuild.Decl{
			Src: `
import "macro"

type S struct{}

//gx:=macro.ID(S.f)
func synthetic()

func (S) f() int32 {
	return 2
}
`,
			Err: "synthetic requires a test.S type receiver",
		},
		testbuild.Decl{
			Src: `
import "macro"

type S struct{}

//gx:=macro.ID(f)
func (S) synthetic()

func f() int32 {
	return 2
}
`,
			Err: "synthetic requires no receiver",
		},
		testbuild.Decl{
			Src: `
import "macro"

type S struct{}

type T struct{}

//gx:=macro.ID(S.f)
func (T) synthetic()

func (S) f() int32 {
	return 2
}
`,
			Err: "cannot assign S.synthetic to T.synthetic",
		},
	)
}
