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

package ir

import (
	"github.com/gx-org/gx/build/ir/irkind"
	"github.com/gx-org/gx/src/gx/ast"
)

type fieldIndexType struct {
}

var (
	fieldIndexIdent = &ast.Ident{Name: "fieldindex"}
	fieldIndexT     = &fieldIndexType{}
)

// FieldIndexType returns the builtin fieldindex type.
func FieldIndexType() Type {
	return fieldIndexT
}

var fieldIndexSliceType = &SliceType{
	BaseType: BaseType[ast.Expr]{Src: &ast.ArrayType{}},
	DType:    TypeExpr(nil, FieldIndexType()),
	Rank:     1,
}

// FieldIndexSliceType returns the type for a slice of fieldindex.
func FieldIndexSliceType() *SliceType {
	return fieldIndexSliceType
}

func (*fieldIndexType) node()         {}
func (*fieldIndexType) storage()      {}
func (*fieldIndexType) storageValue() {}

func (*fieldIndexType) Node() ast.Node { return fieldIndexIdent }

func (*fieldIndexType) Refer(file *File) ast.Expr {
	return fieldIndexIdent
}

func (*fieldIndexType) Same(Storage) bool {
	return true
}

func (t *fieldIndexType) Equal(_ TypeEval, other Type) (bool, error) {
	otherT, ok := other.(*fieldIndexType)
	if !ok {
		return false, nil
	}
	return t == otherT, nil
}

func (t *fieldIndexType) AssignableTo(tpcmp TypeEval, other Type) (bool, error) {
	return t.Equal(tpcmp, other)
}

func (t *fieldIndexType) assignableFrom(tpcmp TypeEval, other Type) (bool, error) {
	return t.Equal(tpcmp, other)
}

func (t *fieldIndexType) ConvertibleTo(tpcmp TypeEval, other Type) (bool, error) {
	return t.Equal(tpcmp, other)
}

func (t *fieldIndexType) Instantiate(Fetcher, Instantiator) (Type, bool) {
	return t, true
}

func (*fieldIndexType) Kind() irkind.Kind { return irkind.FieldIndex }

func (*fieldIndexType) NameDef() *ast.Ident { return fieldIndexIdent }

func (*fieldIndexType) Type() Type { return MetaType() }

func (*fieldIndexType) Value(Expr) Expr { return nil }

func (*fieldIndexType) DefineString(*File) string {
	return irkind.FieldIndex.String()
}

func (t *fieldIndexType) ReferString(from *File) string {
	return t.DefineString(from)
}

// Specialise a type to a given target.
func (t *fieldIndexType) Specialise(spec Specialiser) (Type, bool) {
	return t, true
}

// UnifyWith recursively unifies a type parameters with types.
func (*fieldIndexType) UnifyWith(unifier Unifier, typ Type) bool {
	return true
}

// Unroll returns an error because fieldindex types do not support unrolling.
func (t *fieldIndexType) Unroll(ev Fetcher, urlr Unroller) (ast.Expr, bool) {
	return nil, ev.Err().AppendInternalf(fieldIndexIdent, "%T does not support unrolling", t)
}

func (t *fieldIndexType) IndexForVarArgs(ErrSource, int) (Type, bool) {
	return t, true
}

func (*fieldIndexType) Visit(Finder) {}
