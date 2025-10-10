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

package grad

import (
	"go/ast"

	"github.com/gx-org/gx/build/ir"
)

func (m *vjpMacro) vjpFunc(fetcher ir.Fetcher, src *ir.FuncValExpr) (string, *ast.CallExpr, bool) {
	name := "Fun"
	if pkgFunc, ok := src.F.(ir.PkgFunc); ok {
		name = pkgFunc.Name()
	}
	macro := m.From()
	return name, &ast.CallExpr{
		Fun: &ast.SelectorExpr{
			X:   &ast.Ident{Name: macro.File().Package.Name.Name},
			Sel: &ast.Ident{Name: macro.Name()},
		},
		Args: []ast.Expr{src.X.Source().(ast.Expr)},
	}, true
}
