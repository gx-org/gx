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
	"go/ast"

	"github.com/gx-org/gx/build/ir/irkind"
)

type fieldPathType struct {
}

var (
	fieldPathIdent = &ast.Ident{Name: "fieldpath"}
	fieldPathT     = &fieldPathType{}
)

// FieldPathType returns the builtin fieldpath type.
func FieldPathType() Type {
	return fieldPathT
}

func (*fieldPathType) node()         {}
func (*fieldPathType) storage()      {}
func (*fieldPathType) storageValue() {}

func (*fieldPathType) Node() ast.Node { return fieldPathIdent }

func (*fieldPathType) Refer(file *File) ast.Expr {
	return fieldPathIdent
}

func (*fieldPathType) Same(Storage) bool {
	return true
}

func (t *fieldPathType) Equal(_ TypeCmp, other Type) (bool, error) {
	otherT, ok := other.(*fieldPathType)
	if !ok {
		return false, nil
	}
	return t == otherT, nil
}

func (t *fieldPathType) AssignableTo(tpcmp TypeCmp, other Type) (bool, error) {
	return t.Equal(tpcmp, other)
}

func (t *fieldPathType) assignableFrom(tpcmp TypeCmp, other Type) (bool, error) {
	return t.Equal(tpcmp, other)
}

func (t *fieldPathType) ConvertibleTo(tpcmp TypeCmp, other Type) (bool, error) {
	return t.Equal(tpcmp, other)
}

func (t *fieldPathType) Instantiate(Fetcher, Specialiser) (Type, bool) {
	return t, true
}

func (*fieldPathType) Kind() irkind.Kind { return irkind.FieldPath }

func (*fieldPathType) NameDef() *ast.Ident { return fieldPathIdent }

func (*fieldPathType) Type() Type { return MetaType() }

func (*fieldPathType) Value(Expr) Expr { return nil }

func (*fieldPathType) DefineString(*File) string {
	return irkind.FieldPath.String()
}

func (t *fieldPathType) ReferString(from *File) string {
	return t.DefineString(from)
}

// Specialise a type to a given target.
func (t *fieldPathType) Specialise(spec Specialiser) (Type, bool) {
	return t, true
}

// UnifyWith recursively unifies a type parameters with types.
func (*fieldPathType) UnifyWith(unifier Unifier, typ Type) bool {
	return true
}

func (t *fieldPathType) IndexForVarArgs(ErrSource, int) (Type, bool) {
	return t, true
}
