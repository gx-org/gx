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
	gxfmt "github.com/gx-org/gx/base/fmt"
	"github.com/gx-org/gx/build/ir/irkind"
	"github.com/gx-org/gx/src/gx/ast"
)

// StructType defines the type of a structure.
type StructType struct {
	BaseType[*ast.StructType]
	origin    *StructType
	GenParams *GenericParams

	Fields *FieldList
}

var (
	_ GenericType = (*StructType)(nil)
	_ Zeroer      = (*StructType)(nil)
)

func (*StructType) node() {}

// Kind returns the structure kind.
func (s *StructType) Kind() irkind.Kind { return irkind.Struct }

// Equal returns true if other is the same type.
func (s *StructType) Equal(tpcmp TypeEval, other Type) (bool, error) {
	otherT, ok := other.(*StructType)
	if !ok {
		return false, nil
	}
	if s == otherT {
		return true, nil
	}
	sFields := s.Fields.Fields()
	oFields := otherT.Fields.Fields()
	if len(sFields) != len(oFields) {
		return false, nil
	}
	for i, sField := range sFields {
		oField := oFields[i]
		sName := sField.Name.Name
		oName := oField.Name.Name
		if oName != sName {
			return false, nil
		}
		eq, err := sField.Type().Equal(tpcmp, oField.Type())
		if !eq || err != nil {
			return eq, err
		}
	}
	return true, nil
}

// AssignableTo reports if the type can be assigned to other.
func (s *StructType) AssignableTo(tpcmp TypeEval, target Type) (bool, error) {
	return s.Equal(tpcmp, target)
}

// ConvertibleTo reports whether a value of the type can be converted to another
// (using static type casting).
func (s *StructType) ConvertibleTo(tpcmp TypeEval, target Type) (bool, error) {
	return false, nil
}

// Node returns the node in the AST tree.
func (s *StructType) Node() ast.Node { return s.Src }

// Value returns a value pointing to the receiver.
func (s *StructType) Value(x Expr) Expr {
	return TypeExpr(x.Expr(), s)
}

// Zero returns a zero literal of this type.
// No expression is assigned to the fields of the literal:
// the zero value of each field is computed when the literal is evaluated.
func (s *StructType) Zero() Expr {
	fields := s.Fields.Fields()
	elts := make([]*FieldLit, 0, len(fields))
	for _, field := range fields {
		if !ValidIdent(field.Name) {
			continue
		}
		elts = append(elts, &FieldLit{FieldStorage: field.Storage()})
	}
	return &StructLitExpr{
		Src:  &ast.CompositeLit{},
		Typ:  s,
		Elts: elts,
	}
}

// SpecialiseStruct specialises a structure.
func (s *StructType) SpecialiseStruct(spec Specialiser) (*StructType, bool) {
	res := *s
	res.origin = s.originSType()
	// res.GenParams is originally a pointer to GenParams from the named type.
	// Make sure to decouple it.
	genericsOk := true
	if s.GenParams != nil {
		var genericParams GenericParams
		genericParams, genericsOk = s.GenParams.Specialise(spec, spec.Target() == s)
		res.GenParams = &genericParams
	}
	var paramsOk bool
	res.Fields = cloneFields(s.Fields, &cloner{
		group: specialiseGroup(spec, &paramsOk),
		field: cloneField,
	})
	return &res, genericsOk && paramsOk
}

// Specialise a type to a given target.
func (s *StructType) Specialise(spec Specialiser) (Type, bool) {
	return s.SpecialiseStruct(spec)
}

// originSType returns the original structure type from which this type was defined, typically through specialisation.
// If origin is nil, then it return itself.
func (s *StructType) originSType() *StructType {
	if s.origin == nil {
		return s
	}
	return s.origin
}

// Origin generic type.
func (s *StructType) Origin() GenericType {
	return s.originSType()
}

// Generics returns the generic values.
func (s *StructType) Generics() *GenericParams {
	return s.GenParams
}

// Instantiate a function type.
func (s *StructType) Instantiate(Fetcher, Instantiator) (Type, bool) {
	return s, true
}

// IndexForVarArgs returns a type specific to a given index in varargs.
func (s *StructType) IndexForVarArgs(ErrSource, int) (Type, bool) {
	return s, true
}

// NumFields returns the number of fields in a structure.
func (s *StructType) NumFields() int {
	n := 0
	for _, group := range s.Fields.List {
		n += group.NumFields()
	}
	return n
}

// StructTypeAST returns the AST tree of the structure type.
func (s *StructType) StructTypeAST(from *File) *ast.StructType {
	src := *s.Src
	src.Fields = s.Fields.FieldListAST(from)
	return &src
}

// DefineString returns a reference to the type given a file context.
func (s *StructType) DefineString(from *File) string {
	ast := s.StructTypeAST(from)
	return gxfmt.Format(ast)
}

// ReferString returns a reference to the type given a file context.
func (s *StructType) ReferString(from *File) string {
	return s.DefineString(from)
}
