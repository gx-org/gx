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

import (
	"go/ast"
)

// nilType is the type returned by function with no results.
type nilType struct {
	distinctType
}

// NilType returns the type for the nil identifier.
func NilType() Type {
	return nilT
}

// Nil implements the storage for the nil keyword.
type Nil struct {
	Src *ast.Ident
}

var _ Storage = (*Nil)(nil)

func (*Nil) node()    {}
func (*Nil) storage() {}

// Node returns the AST node of the storage.
func (n *Nil) Node() ast.Node {
	return n.Src
}

// NameDef returns the identifier of the storage.
func (n *Nil) NameDef() *ast.Ident {
	return n.Src
}

// Same returns true if the other storage is a nil storage as well.
func (*Nil) Same(other Storage) bool {
	_, ok := other.(*Nil)
	return ok
}

// Type of the Nil storage.
func (*Nil) Type() Type {
	return NilType()
}

// NilCastExpr casts nil to a given type.
type NilCastExpr struct {
	X   Expr
	Typ Type
}

var _ Expr = (*NilCastExpr)(nil)

func (*NilCastExpr) node() {}

// Node returns the AST node of the storage.
func (n *NilCastExpr) Node() ast.Node {
	return n.X.Node()
}

// SourceString returns a string representing the expression.
func (n *NilCastExpr) SourceString(from *File) string {
	return n.X.SourceString(from)
}

// Expr returns the syntax tree of the expression.
func (n *NilCastExpr) Expr() ast.Expr {
	return n.X.Expr()
}

// Type of the expression.
func (n *NilCastExpr) Type() Type {
	return n.Typ
}
