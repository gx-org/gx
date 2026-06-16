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
	X      Expr
	EltTyp Type
}

var (
	_ ExprWithSpecialise = (*UnpackExpr)(nil)
	_ ExprWithUnify      = (*UnpackExpr)(nil)
	_ VarArgsIndexer     = (*VarArgsIndex)(nil)
)

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

// UnifyWith recursively unifies a type parameters with types.
func (u *UnpackExpr) UnifyWith(uni Unifier, targets []AxisLengths) ([]AxisLengths, bool) {
	return unifyExpr(uni, targets, u.X)
}

// Specialise the expression.
func (u *UnpackExpr) Specialise(spec Specialiser) (Expr, bool) {
	r := *u
	var ok bool
	r.X, ok = specialiseExpr(spec, u.X)
	return &r, ok
}

// IndexForVarArgs returns a type specific to a given index in varargs.
func (u *UnpackExpr) IndexForVarArgs(i int) Expr {
	r := *u
	r.X = varArgsIndexExpr(i, r.X)
	return &r
}

// SourceString returns the GX source code of the expression.
func (u *UnpackExpr) SourceString(from *File) string {
	return u.X.SourceString(from) + "..."
}
