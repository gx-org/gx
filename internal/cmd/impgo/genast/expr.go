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

package genast

import "go/ast"

// Expr is an AST expression with helper functions.
type Expr[T ast.Expr] struct {
	X T
}

// IdentExpr is an identifier expression.
type IdentExpr = *Expr[*ast.Ident]

// Select builds a select expression.
func (x Expr[T]) Select(sel string) Expr[*ast.SelectorExpr] {
	return Expr[*ast.SelectorExpr]{X: &ast.SelectorExpr{
		X:   x.X,
		Sel: Ident(sel).X,
	}}
}

// Index builds an index expression.
func (x Expr[T]) Index(i ast.Expr) Expr[*ast.IndexExpr] {
	return Expr[*ast.IndexExpr]{X: &ast.IndexExpr{
		X:     x.X,
		Index: i,
	}}
}

// Star returns a pointer of the expression.
func (x Expr[T]) Star() Expr[*ast.StarExpr] {
	return Expr[*ast.StarExpr]{X: &ast.StarExpr{
		X: x.X,
	}}
}

// IndexAt builds an index expression given an integer.
func (x Expr[T]) IndexAt(i int) Expr[*ast.IndexExpr] {
	return Expr[*ast.IndexExpr]{X: &ast.IndexExpr{
		X:     x.X,
		Index: IntLit(i),
	}}
}

// AstExprs converts a slice of Expr into a slice of ast.Expr.
func AstExprs[T ast.Expr](xs []*Expr[T]) []ast.Expr {
	res := make([]ast.Expr, len(xs))
	for i, x := range xs {
		res[i] = x.X
	}
	return res
}
