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

package ir

import "go/ast"

// UnpackExpr an expression.
type UnpackExpr struct {
	X           Expr
	ElementType Type
}

var _ Expr = (*UnpackExpr)(nil)

func (*UnpackExpr) node() {}

// Node returns the node in the AST tree.
func (u *UnpackExpr) Node() ast.Node {
	return u.X.Node()
}

// Expr returns the source of the expression.
func (u *UnpackExpr) Expr() ast.Expr {
	return u.X.Expr()
}

// Type returns the type of the expression.
func (u *UnpackExpr) Type() Type {
	return u.X.Type()
}

// SourceString returns the GX source code of the expression.
func (u *UnpackExpr) SourceString(from *File) string {
	return u.X.SourceString(from) + "..."
}
