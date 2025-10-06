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

	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/interp/compeval/cpevelements"
	"github.com/gx-org/gx/interp"
)

type id struct {
	cpevelements.CoreMacroElement
	fn *ir.FuncDecl
}

var _ cpevelements.FuncASTBuilder = (*id)(nil)

func newID(file *ir.File, call *ir.CallExpr, mac *ir.Macro, args []ir.Element) (ir.MacroElement, error) {
	fn, err := interp.FuncDeclFromElement(args[0])
	if err != nil {
		return nil, err
	}
	return &id{CoreMacroElement: cpevelements.MacroElement(mac, file, call), fn: fn}, nil
}

func (m *id) BuildDecl(ir.PkgFunc) (*ast.FuncDecl, bool) {
	return &ast.FuncDecl{Type: m.fn.FType.Src, Recv: m.fn.Src.Recv}, true
}

func (m *id) BuildBody(fetcher ir.Fetcher, _ ir.Func) (*ast.BlockStmt, []*cpevelements.SyntheticFuncDecl, bool) {
	return m.fn.Body.Src, nil, true
}
