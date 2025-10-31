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

// Package exprdeps extracts identifier dependencies from AST expressions.
package exprdeps

import (
	"go/ast"
	"slices"

	"github.com/gx-org/gx/base/ordered"
)

func idents(done *ordered.Map[string, *ast.Ident], expr ast.Expr) {
	switch exprT := expr.(type) {
	case *ast.Ident:
		if exprT == nil {
			return
		}
		done.Store(exprT.Name, exprT)
	case *ast.ParenExpr:
		idents(done, exprT.X)
	case *ast.UnaryExpr:
		idents(done, exprT.X)
	case *ast.BinaryExpr:
		idents(done, exprT.X)
		idents(done, exprT.Y)
	}
}

// Idents returns a slice of all identifiers used in an expression.
func Idents(expr ast.Expr) []*ast.Ident {
	done := ordered.NewMap[string, *ast.Ident]()
	idents(done, expr)
	return slices.Collect(done.Values())
}
