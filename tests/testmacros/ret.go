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

package testmacros

import (
	"go/ast"
	"go/token"
	"strconv"

	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/interp/compeval/cpevelements"
	"github.com/gx-org/gx/interp/elements"
	"github.com/gx-org/gx/interp"
)

type updateReturn struct {
	cpevelements.CoreMacroElement
	fn  *ir.FuncDecl
	str string
}

var _ cpevelements.FuncASTBuilder = (*updateReturn)(nil)

func newUpdateReturn(call elements.CallAt, macro *cpevelements.Macro, args []ir.Element) (cpevelements.MacroElement, error) {
	fn, err := interp.FuncDeclFromElement(args[0])
	if err != nil {
		return nil, err
	}
	str, err := elements.StringFromElement(args[1])
	if err != nil {
		return nil, err
	}
	return &updateReturn{
		CoreMacroElement: macro.Element(call),
		fn:               fn,
		str:              str,
	}, nil
}

func (m *updateReturn) BuildDecl(ir.PkgFunc) (*ast.FuncDecl, bool) {
	return &ast.FuncDecl{Type: m.fn.FType.Src, Recv: m.fn.Src.Recv}, true
}

func (m *updateReturn) body() *ast.BlockStmt {
	return &ast.BlockStmt{
		List: []ast.Stmt{
			&ast.ReturnStmt{Results: []ast.Expr{
				&ast.BasicLit{
					Kind:  token.STRING,
					Value: strconv.Quote(m.str),
				},
			}},
		},
	}
}

func (m *updateReturn) BuildBody(fetcher ir.Fetcher, _ ir.Func) (*ast.BlockStmt, []*cpevelements.SyntheticFuncDecl, bool) {
	return m.body(), nil, true
}

func (m *updateReturn) BuildFuncLit(fetcher ir.Fetcher) (*ast.FuncLit, bool) {
	return &ast.FuncLit{
		Type: m.fn.FType.Src,
		Body: m.body(),
	}, true
}
