// Copyright 2024 Google LLC
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

	"github.com/gx-org/gx/build/ir/irkind"
)

// VarArgsType represents a type defined by a varargs expression.
type VarArgsType struct {
	Src   *ast.Ellipsis
	Slice *SliceType
}

var _ Type = (*VarArgsType)(nil)

func (*VarArgsType) node()         {}
func (*VarArgsType) storage()      {}
func (*VarArgsType) storageValue() {}

// Node returns the node in the AST tree.
func (tp *VarArgsType) Node() ast.Node {
	return tp.Src
}

// Same returns true if the other storage is this storage.
func (tp *VarArgsType) Same(Storage) bool {
	return false
}

// NameDef returns the name defining the storage.
func (*VarArgsType) NameDef() *ast.Ident {
	return nil
}

// Value returns the value of the type.
func (tp *VarArgsType) Value(x Expr) Expr {
	return TypeExpr(x, tp)
}

// Kind of the type.
func (tp *VarArgsType) Kind() irkind.Kind {
	return tp.Slice.Kind()
}

// Type returns the type of the varargs type.
func (tp *VarArgsType) Type() Type {
	return MetaType()
}

// Equal returns true if other is the same type.
func (tp *VarArgsType) Equal(fetcher Fetcher, target Type) (bool, error) {
	return tp.Slice.Equal(fetcher, target)
}

// AssignableTo reports whether a value of the type can be assigned to another.
func (tp *VarArgsType) AssignableTo(fetcher Fetcher, target Type) (bool, error) {
	return tp.Slice.AssignableTo(fetcher, target)
}

// ConvertibleTo reports whether a value of the type can be converted to another
// (using static type casting).
func (tp *VarArgsType) ConvertibleTo(fetcher Fetcher, target Type) (bool, error) {
	return tp.Slice.ConvertibleTo(fetcher, target)
}

// Specialise a type to a given target.
func (tp *VarArgsType) Specialise(spec Specialiser) (Type, error) {
	return tp.Slice.Specialise(spec)
}

// UnifyWith recursively unifies a type parameters with types.
func (tp *VarArgsType) UnifyWith(uni Unifier, typ Type) bool {
	return tp.Slice.UnifyWith(uni, typ)
}

// SourceString returns a reference to the type given a file context.
func (tp *VarArgsType) SourceString(file *File) string {
	return "..." + tp.Slice.SourceString(file)
}

// String representation of the type.
func (tp *VarArgsType) String() string {
	return "..." + tp.Slice.String()
}

// VarArgsExpr represents a type expression to represent a varargs type.
type VarArgsExpr struct {
	Elt *VarArgsType
}

var _ Expr = (*VarArgsExpr)(nil)

func (*VarArgsExpr) node() {}

// Node returns the node in the AST tree.
func (expr *VarArgsExpr) Node() ast.Node {
	return expr.Elt.Node()
}

// Expr returns the source of the expression.
func (expr *VarArgsExpr) Expr() ast.Expr {
	return expr.Elt.Src
}

// Type returns the type of the expression.
func (expr *VarArgsExpr) Type() Type {
	return expr.Elt.Type()
}

// String returns a string representation of the expression.
func (expr *VarArgsExpr) String() string {
	return "..." + expr.Elt.Slice.DType.String()
}
