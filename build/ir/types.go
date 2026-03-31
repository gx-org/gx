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
	"strings"

	"github.com/pkg/errors"
	"github.com/gx-org/gx/build/ir/irkind"
)

// BaseType is the base of all types.
type BaseType[T ast.Expr] struct {
	Src T
}

func (BaseType[T]) node()         {}
func (BaseType[T]) storage()      {}
func (BaseType[T]) storageValue() {}

// NameDef of the base type always returns a nil name definition.
func (BaseType[T]) NameDef() *ast.Ident { return nil }

// Node returns the source node defining the type.
func (m *BaseType[T]) Node() ast.Node {
	return m.Src
}

// Value returns nil.
func (BaseType[T]) Value(Expr) Expr {
	return nil
}

// Type of a type: always return metatype.
func (BaseType[T]) Type() Type {
	return MetaType()
}

// Same returns true if the other storage is this storage.
func (m *BaseType[T]) Same(o Storage) bool {
	return Storage(m) == o
}

// Specialise a type to a given target.
func (*BaseType[T]) Specialise(spec Specialiser) (Type, CompEvalError, error) {
	return nil, nil, errors.Errorf("type specialisation not supported")
}

// UnifyWith recursively unifies a type parameters with types.
func (*BaseType[T]) UnifyWith(unifier Unifier, typ Type) bool {
	return true
}

type metaType struct {
	BaseType[*ast.Ident]
}

func (*metaType) node() {}

func (*metaType) Equal(TypeCmp, Type) (bool, CompEvalError, error) {
	return false, nil, nil
}

func (*metaType) AssignableTo(TypeCmp, Type) (bool, CompEvalError, error) {
	return false, nil, nil
}

func (*metaType) ConvertibleTo(TypeCmp, Type) (bool, CompEvalError, error) {
	return false, nil, nil
}

func (t *metaType) Kind() irkind.Kind { return irkind.MetaType }

func (t *metaType) DefineString(*File) string {
	return irkind.MetaType.String()
}

func (t *metaType) ReferString(from *File) string {
	return t.DefineString(from)
}

var metaTypeT = &metaType{
	BaseType: BaseType[*ast.Ident]{Src: &ast.Ident{Name: irkind.MetaType.String()}},
}

// MetaType returns the meta type shared across all types.
func MetaType() Type {
	return metaTypeT
}

type invalidType struct {
}

var invalidT = &invalidType{}

// InvalidType returns an invalid type.
// Often used as a placeholder when an error occurred.
func InvalidType() Type {
	return invalidT
}

func (*invalidType) node()         {}
func (*invalidType) storage()      {}
func (*invalidType) storageValue() {}

func (*invalidType) Node() ast.Node { return nil }

func (*invalidType) Same(Storage) bool {
	return true
}

func (*invalidType) Equal(TypeCmp, Type) (bool, CompEvalError, error) {
	// If the type is invalid, an error has already been emitted.
	// We disable any checks to avoid reporting unhelpful errors.
	return false, nil, nil
}

func (*invalidType) AssignableTo(TypeCmp, Type) (bool, CompEvalError, error) {
	return (*invalidType).Equal(nil, nil, nil)
}

func (*invalidType) assignableFrom(TypeCmp, Type) (bool, CompEvalError, error) {
	return (*invalidType).Equal(nil, nil, nil)
}

func (*invalidType) ConvertibleTo(TypeCmp, Type) (bool, CompEvalError, error) {
	return (*invalidType).Equal(nil, nil, nil)
}

func (*invalidType) Kind() irkind.Kind { return irkind.Invalid }

func (*invalidType) NameDef() *ast.Ident { return nil }

func (*invalidType) Type() Type { return InvalidType() }

func (*invalidType) Value(Expr) Expr { return nil }

func (*invalidType) DefineString(*File) string {
	return irkind.Invalid.String()
}

func (t *invalidType) ReferString(from *File) string {
	return t.DefineString(from)
}

// Specialise a type to a given target.
func (t *invalidType) Specialise(spec Specialiser) (Type, CompEvalError, error) {
	return t, nil, nil
}

// UnifyWith recursively unifies a type parameters with types.
func (*invalidType) UnifyWith(unifier Unifier, typ Type) bool {
	return true
}

type distinctType struct {
	BaseType[ast.Expr]
	kind irkind.Kind
}

func (*distinctType) node() {}

func (t *distinctType) distinct() *distinctType {
	return t
}

func (t *distinctType) Equal(_ TypeCmp, target Type) (bool, CompEvalError, error) {
	other, ok := target.(interface{ distinct() *distinctType })
	if !ok {
		return false, nil, nil
	}
	return t == other.distinct(), nil, nil
}

func (t *distinctType) AssignableTo(tpcmp TypeCmp, target Type) (bool, CompEvalError, error) {
	return t.Equal(tpcmp, target)
}

func (t *distinctType) ConvertibleTo(tpcmp TypeCmp, target Type) (bool, CompEvalError, error) {
	return t.Equal(tpcmp, target)
}

func (t *distinctType) Kind() irkind.Kind { return t.kind }

func (t *distinctType) DefineString(*File) string {
	return t.kind.String()
}

func (t *distinctType) ReferString(from *File) string {
	return t.DefineString(from)
}

// unknownType is the type returned by function with no results.
type unknownType struct {
	distinctType
}

var unknownT = &unknownType{distinctType: distinctType{kind: irkind.Unknown}}

// UnknownType returns the unknown type.
func UnknownType() Type {
	return unknownT
}

// keywordTyp is the type returned by function with no results.
type keywordTyp struct {
	distinctType
}

var keywordT = &keywordTyp{distinctType: distinctType{kind: irkind.Func}}

func keywordType() Type {
	return keywordT
}

// voidType is the type returned by function with no results.
type voidType struct {
	distinctType
}

var voidT = &voidType{distinctType: distinctType{kind: irkind.Void}}

// VoidType returns the void type.
func VoidType() Type {
	return voidT
}

type packageType struct {
	distinctType
}

var packageT = &packageType{distinctType: distinctType{kind: irkind.Package}}

// PackageType returns the package type.
func PackageType() Type {
	return packageT
}

// TypeInclude returns if a typ is included in a type or not.
func TypeInclude(tpcmp TypeCmp, set Type, typ Type) (bool, CompEvalError, error) {
	set = simplifyType(set)
	typ = simplifyType(typ)
	_, isSetNamed := set.(*NamedType)
	_, isTypeNamed := typ.(*NamedType)
	if isSetNamed && isTypeNamed {
		return set.Equal(tpcmp, typ)
	}
	return typeInclude(tpcmp, set, typ)
}

func typeInclude(tpcmp TypeCmp, set Type, typ Type) (bool, CompEvalError, error) {
	set = Underlying(set)
	typeSet, typeSetOk := set.(*Interface)
	if !typeSetOk {
		return set.Equal(tpcmp, typ)
	}
	for _, sub := range typeSet.types {
		in, cpErr, err := typeInclude(tpcmp, sub, typ)
		if cpErr != nil || err != nil {
			return false, nil, err
		}
		if in {
			return true, nil, nil
		}
	}
	return false, nil, nil
}

// Source returns the source code defining the type.
// Always returns nil.
func (s *Interface) Source() ast.Node { return s.ArrayType() }

// Zero returns a zero expression of the same type.
func (s *Interface) Zero() Expr {
	panic("unimplemented")
}

// ElementType returns the type of an element.
func (s *Interface) ElementType() (Type, bool) {
	sub := &Interface{}
	ok := true
	for _, typ := range s.types {
		aType, eltOk := typ.(SlicerType)
		if !eltOk {
			ok = false
			continue
		}
		sType, eltOk := aType.ElementType()
		if !eltOk {
			ok = false
			continue
		}
		sub.types = append(sub.types, sType)
	}
	return sub, ok
}

// Value returns a value pointing to the receiver.
func (s *Interface) Value(x Expr) Expr {
	return TypeExpr(x, s)
}

func (s *Interface) interfaceString(types []string) string {
	if len(types) == 0 {
		return "any"
	}
	var b strings.Builder
	b.WriteString("interface { ")
	b.WriteString(strings.Join(types, "|"))
	b.WriteString(" }")
	return b.String()
}

// DefineString returns the GX source code to define the type.
func (s *Interface) DefineString(from *File) string {
	types := make([]string, len(s.types))
	for i, typ := range s.types {
		types[i] = typ.ReferString(from)
	}
	return s.interfaceString(types)
}

// ReferString returns the GX source to refer to the type.
func (s *Interface) ReferString(from *File) string {
	return s.DefineString(from)
}

func (s *Interface) equalArray(tpcmp TypeCmp, target ArrayType) (bool, CompEvalError, error) {
	return s.Equal(tpcmp, target)
}

// hasCapability returns true if and only if the capability applies to all types in the set.
// Returns false if the type set is empty.
func (s *Interface) hasCapability(f func(Type) bool) bool {
	for _, typ := range s.types {
		if !f(typ) {
			return false
		}
	}
	return len(s.types) > 0
}

// containsType returns true if the given type is present in the set.
func (s *Interface) containsType(tpcmp TypeCmp, wantType Type) (bool, CompEvalError, error) {
	for _, typ := range s.types {
		eq, cpErr, err := wantType.Equal(tpcmp, typ)
		if cpErr != nil || err != nil {
			return false, cpErr, err
		}
		if eq {
			return true, nil, nil
		}
	}
	return false, nil, nil
}

// containsTypes returns true if all the given types are present in the set.
func (s *Interface) containsTypes(tpcmp TypeCmp, types *Interface) (bool, CompEvalError, error) {
	for _, wantType := range types.types {
		eq, cpErr, err := s.containsType(tpcmp, wantType)
		if cpErr != nil || err != nil {
			return false, cpErr, err
		}
		if !eq {
			return false, nil, nil
		}
	}
	return true, nil, nil
}

func fieldListAtomicIR() *FieldList {
	group := &FieldGroup{
		Type: TypeExpr(nil, TypeFromKind(irkind.IR)),
	}
	group.Fields = []*Field{&Field{
		Group: group,
	}}
	return &FieldList{
		List: []*FieldGroup{group},
	}
}

var metaFuncType = &FuncType{
	Params:  fieldListAtomicIR(),
	Results: fieldListAtomicIR(),
}

// MetaFuncType returns the type of a meta function.
func MetaFuncType() *FuncType {
	return metaFuncType
}

// ToArrayType converts a type into an array type.
// Returns nil if the conversion is not possible
func ToArrayType(typ Type) ArrayType {
	switch typT := typ.(type) {
	case *TypeParam:
		return ToArrayType(typT.Field.Group.Type.Val())
	case *NamedType:
		return ToArrayType(typT.Underlying.Val())
	case *Interface:
		if !typT.hasCapability(IsDataType) {
			return nil
		}
		return typT
	case ArrayType:
		return typT
	}
	return nil
}

type equaler interface {
	equal(tpcmp TypeCmp, x Type) (bool, CompEvalError, error)
}

// Equal returns true if x and y are the same type.
func Equal(tpcmp TypeCmp, x, y Type) (bool, CompEvalError, error) {
	if x == y {
		return true, nil, nil
	}
	ey, isEqualer := y.(equaler)
	if isEqualer {
		return ey.equal(tpcmp, x)
	}
	return x.Equal(tpcmp, y)
}

type assigner interface {
	assignableFrom(tpcmp TypeCmp, x Type) (bool, CompEvalError, error)
}

func simplifyType(t Type) Type {
	typeParam, ok := t.(*TypeParam)
	if !ok {
		return t
	}
	return typeParam.Field.Group.Type.Val()
}

// IsInvalidType returns true if a type is invalid.
func IsInvalidType(typ Type) bool {
	if typ == nil {
		return false
	}
	return typ.Kind() == irkind.Invalid
}

// AssignableTo reports whether a value of the type can be assigned to another.
func AssignableTo(tpcmp TypeCmp, x, y Type) (bool, CompEvalError, error) {
	if IsInvalidType(x) || IsInvalidType(y) {
		// An error should have already been reported. We skip the check
		// to prevent additional confusing errors.
		return true, nil, nil
	}
	if x.Kind() == irkind.Tuple || y.Kind() == irkind.Tuple {
		return true, nil, nil
	}
	x = simplifyType(x)
	y = simplifyType(y)
	if x == y {
		return true, nil, nil
	}
	ey, isAssigner := y.(assigner)
	if isAssigner {
		return ey.assignableFrom(tpcmp, x)
	}
	return x.AssignableTo(tpcmp, y)
}
