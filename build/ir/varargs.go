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
	Src *ast.Ellipsis
	Typ *SliceType
}

var _ SlicerType = (*VarArgsType)(nil)

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
	return tp.Typ.Kind()
}

// Type returns the type of the varargs type.
func (tp *VarArgsType) Type() Type {
	return MetaType()
}

// ElementType returns the type of the variable argument.
func (tp *VarArgsType) ElementType() (Type, bool) {
	return tp.Typ.ElementType()
}

// Equal returns true if other is the same type.
func (tp *VarArgsType) Equal(tpcmp TypeCmp, target Type) (bool, CompEvalError, error) {
	return tp.Typ.Equal(tpcmp, target)
}

// AssignableTo reports whether a value of the type can be assigned to another.
func (tp *VarArgsType) AssignableTo(tpcmp TypeCmp, target Type) (bool, CompEvalError, error) {
	return tp.Typ.AssignableTo(tpcmp, target)
}

// ConvertibleTo reports whether a value of the type can be converted to another
// (using static type casting).
func (tp *VarArgsType) ConvertibleTo(tpcmp TypeCmp, target Type) (bool, CompEvalError, error) {
	return tp.Typ.ConvertibleTo(tpcmp, target)
}

// Specialise a type to a given target.
func (tp *VarArgsType) Specialise(spec Specialiser) (Type, CompEvalError, error) {
	return tp.Typ.Specialise(spec)
}

// UnifyWith recursively unifies a type parameters with types.
func (tp *VarArgsType) UnifyWith(uni Unifier, typ Type) bool {
	return tp.Typ.UnifyWith(uni, typ)
}

// DefineString returns a reference to the type given a file context.
func (tp *VarArgsType) DefineString(from *File) string {
	return "..." + tp.Typ.DefineString(from)
}

// ReferString returns the GX source to refer to the type.
func (tp *VarArgsType) ReferString(from *File) string {
	return tp.DefineString(from)
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

// SourceString returns the GX source code of the expression.
func (expr *VarArgsExpr) SourceString(from *File) string {
	return "..." + expr.Elt.Typ.DType.SourceString(from)
}
