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

var unknownT = &unknownType{
	BaseType: BaseType[*ast.Ident]{Src: &ast.Ident{Name: irkind.Unknown.String()}},
}

// unknownType is the type returned by function with no results.
type unknownType struct {
	BaseType[*ast.Ident]
}

var _ ArrayType = unknownT

func (*unknownType) node()      {}
func (*unknownType) arrayType() {}

func (t *unknownType) Equal(_ TypeEval, target Type) (bool, error) {
	return true, nil
}

func (t *unknownType) AssignableTo(tpcmp TypeEval, target Type) (bool, error) {
	return true, nil
}

func (t *unknownType) ConvertibleTo(tpcmp TypeEval, target Type) (bool, error) {
	return true, nil
}

func (t *unknownType) Kind() irkind.Kind { return irkind.Unknown }

func (t *unknownType) DefineString(*File) string {
	return irkind.Unknown.String()
}

func (t *unknownType) ReferString(from *File) string {
	return t.DefineString(from)
}

func (t *unknownType) Specialise(Specialiser) (Type, bool) {
	return t, true
}

func (t *unknownType) Instantiate(Fetcher, Instantiator) (Type, bool) {
	return t, true
}

func (t *unknownType) IndexForVarArgs(ErrSource, int) (Type, bool) {
	return t, true
}

func (t *unknownType) ArrayType() ast.Expr {
	return nil
}

func (t *unknownType) Rank() ArrayRank {
	return scalarRank
}

func (t *unknownType) Zero() Expr {
	return &NumberCastExpr{
		X:   zero,
		Typ: t,
	}
}

func (t *unknownType) ElementType() (Type, bool) {
	return t, false
}

func (t *unknownType) DataType() Type {
	return t
}

// UnknownType returns the unknown type.
func UnknownType() ArrayType {
	return unknownT
}

// IsUnknown returns true if one of the type is unknown.
func IsUnknown(types ...Type) bool {
	for _, tp := range types {
		if tp == unknownT {
			return true
		}
	}
	return false
}
