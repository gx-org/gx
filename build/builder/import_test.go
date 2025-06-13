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
	"math/big"
	"testing"

	"github.com/gx-org/gx/build/builder/testbuild"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/build/ir/irhelper"
)

func TestImportType(t *testing.T) {
	typInt := &ir.NamedType{
		File:       &ir.File{Package: &ir.Package{Name: irhelper.Ident("dtype")}},
		Src:        &ast.TypeSpec{Name: irhelper.Ident("Int")},
		Underlying: irhelper.TypeExpr(ir.Int32Type()),
	}
	testbuild.Run(t,
		testbuild.DeclarePackage{
			Src: `
package dtype

type Int int32
`},
		testbuild.DeclTest{
			Src: `
import "dtype"

func bla() dtype.Int
`,
			Want: []ir.Node{
				&ir.FuncBuiltin{
					FType: irhelper.FuncType(
						nil, nil,
						irhelper.Fields(),
						irhelper.Fields(typInt),
					),
				},
			},
		},
	)
}

func TestImportConst(t *testing.T) {
	pkgImportDecl := &ir.ImportDecl{
		Src:  &ast.ImportSpec{Name: irhelper.Ident("pkg")},
		Path: "pkg",
	}
	constExpr := &ir.ConstExpr{
		Decl:  &ir.ConstDecl{},
		VName: irhelper.Ident("MyConst"),
		Val:   &ir.NumberInt{Val: big.NewInt(42)},
	}
	constExpr.Decl.Exprs = []*ir.ConstExpr{constExpr}
	testbuild.Run(t,
		testbuild.DeclarePackage{
			Src: `
package pkg

const MyConst = 42

func F(int32) int32
`},
		testbuild.DeclTest{
			Src: `
import "pkg"

func returnMyConst() int32 {
	return pkg.MyConst
}
`,
			Want: []ir.Node{
				&ir.FuncDecl{
					FType: irhelper.FuncType(
						nil, nil,
						irhelper.Fields(),
						irhelper.Fields(ir.Int32Type()),
					),
					Body: irhelper.Block(
						&ir.ReturnStmt{
							Results: []ir.Expr{
								&ir.NumberCastExpr{
									X: &ir.SelectorExpr{
										X:    irhelper.ValueRef(pkgImportDecl),
										Stor: constExpr,
									},
									Typ: ir.Int32Type(),
								},
							},
						},
					)},
			},
		},
		testbuild.DeclTest{
			Src: `
import "pkg"

func f(a int32) int32 {
	return pkg.F(a)
}
`,
		},
	)
}
