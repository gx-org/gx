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

	"github.com/gx-org/gx/build/ir/irkind"
)

type errorType struct {
	iface Interface
}

var errorSrc = &ast.Field{
	Names: []*ast.Ident{
		&ast.Ident{Name: "Error"},
	},
	Type: &ast.FuncType{},
}

var errorTyp = &errorType{
	iface: Interface{
		methods: []*IMethod{
			// Define method: error() string
			&IMethod{
				Src: errorSrc,
				FType: &FuncType{
					BaseType: BaseType[*ast.FuncType]{
						Src: errorSrc.Type.(*ast.FuncType),
					},
					Params: &FieldList{},
					Results: &FieldList{
						List: []*FieldGroup{
							&FieldGroup{Type: TypeExpr(nil, StringType())},
						},
					},
				},
			},
		},
	},
}
var errorIdent = &ast.Ident{Name: "error"}

var (
	_ Type     = errorTyp
	_ assigner = errorTyp
)

// ErrorType returns the type for the keyword error.
func ErrorType() TypeMethods {
	return errorTyp
}

func (*errorType) node()         {}
func (*errorType) storage()      {}
func (*errorType) storageValue() {}

// Kind returns the scalar kind.
func (s *errorType) Kind() irkind.Kind { return irkind.Interface }

// Equal returns true if other is the exact same type set.
func (s *errorType) Equal(fetcher Fetcher, target Type) (bool, CompEvalError, error) {
	return s.iface.Equal(fetcher, target)
}

// AssignableTo reports whether a value of the type can be assigned to another.
func (s *errorType) AssignableTo(fetcher Fetcher, target Type) (bool, CompEvalError, error) {
	return s.iface.AssignableTo(fetcher, target)
}

func (s *errorType) assignableFrom(fetcher Fetcher, x Type) (bool, CompEvalError, error) {
	return s.iface.assignableFromWithName(fetcher, x, s.DefineString)
}

// ConvertibleTo reports whether a value of the type can be converted to another
// (using static type casting).
func (s *errorType) ConvertibleTo(fetcher Fetcher, target Type) (bool, CompEvalError, error) {
	return s.iface.ConvertibleTo(fetcher, target)
}

// DefineString returns the GX source code to define the type.
func (s *errorType) DefineString(from *File) string {
	return "error"
}

func (*errorType) Methods() []PkgFunc {
	return nil
}

// NameDef of the base type always returns a nil name definition.
func (s *errorType) NameDef() *ast.Ident { return errorIdent }

// Node returns the source node defining the type.
func (s *errorType) Node() ast.Node {
	return s.iface.Node()
}

// Same returns true if the other storage is this storage.
func (s *errorType) Same(o Storage) bool {
	return Storage(s) == o
}

// Specialise a type to a given target.
func (s *errorType) Specialise(spec Specialiser) (Type, error) {
	return s, nil
}

// Value returns a value pointing to the receiver.
func (s *errorType) Value(x Expr) Expr {
	return TypeExpr(x, s)
}

// UnifyWith recursively unifies a type parameters with types.
func (s *errorType) UnifyWith(unifier Unifier, typ Type) bool {
	return true
}

// Type of a type: always return metatype.
func (s *errorType) Type() Type {
	return MetaType()
}

// ReferString returns the GX source to refer to the type.
func (s *errorType) ReferString(from *File) string {
	return s.DefineString(from)
}
