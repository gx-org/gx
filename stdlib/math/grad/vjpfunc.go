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
	switch fT := src.F.(type) {
	case *ir.FuncDecl:
		return vjpFuncDecl(fetcher, m, src, fT)
	default:
		return "", nil, fetcher.Err().Appendf(src.Source(), "cannot compute the gradient of function %s", fT.ShortString())
	}
}

func (m *vjpMacro) buildCallAST(fetcher ir.Fetcher, fn ir.Func) (*ast.CallExpr, bool) {
	macro := m.From().IR()
	return &ast.CallExpr{
		Fun: &ast.SelectorExpr{
			X:   &ast.Ident{Name: macro.File().Package.Name.Name},
			Sel: &ast.Ident{Name: macro.Name()},
		},
		Args: []ast.Expr{
			&ast.Ident{Name: fn.ShortString()},
		},
	}, true
}

func vjpFuncDecl(fetcher ir.Fetcher, parent *vjpMacro, src *ir.FuncValExpr, fn *ir.FuncDecl) (string, *ast.CallExpr, bool) {
	macro := parent.newMacro(fn)
	vjpCall, ok := macro.buildCallAST(fetcher, fn)
	if !ok {
		return "", nil, false
	}
	return fn.Name(), vjpCall, true
}
