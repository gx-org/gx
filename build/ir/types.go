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
	"fmt"
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
func (BaseType[T]) Value(Expr) AssignableExpr {
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
func (*BaseType[T]) Specialise(spec Specialiser) (Type, error) {
	return nil, errors.Errorf("type specialisation not supported")
}

// UnifyWith recursively unifies a type parameters with types.
func (*BaseType[T]) UnifyWith(unifier Unifier, typ Type) bool {
	return true
}

type metaType struct {
	BaseType[*ast.Ident]
}

func (*metaType) node() {}

func (*metaType) Equal(Fetcher, Type) (bool, error) {
	return false, nil
}

func (*metaType) AssignableTo(Fetcher, Type) (bool, error) {
	return false, nil
}

func (*metaType) ConvertibleTo(fetcher Fetcher, target Type) (bool, error) {
	return false, nil
}

func (t *metaType) Kind() irkind.Kind { return irkind.MetaType }

func (t *metaType) SourceString(*File) string {
	return t.String()
}

func (t *metaType) String() string {
	return irkind.MetaType.String()
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

func (*invalidType) Equal(Fetcher, Type) (bool, error) {
	// If the type is invalid, an error has already been emitted.
	// We disable any checks to avoid reporting unhelpful errors.
	return false, nil
}

func (*invalidType) AssignableTo(Fetcher, Type) (bool, error) {
	return (*invalidType).Equal(nil, nil, nil)
}

func (*invalidType) assignableFrom(Fetcher, Type) (bool, error) {
	return (*invalidType).Equal(nil, nil, nil)
}

func (*invalidType) ConvertibleTo(Fetcher, Type) (bool, error) {
	return (*invalidType).Equal(nil, nil, nil)
}

func (*invalidType) Kind() irkind.Kind { return irkind.Invalid }

func (*invalidType) NameDef() *ast.Ident { return nil }

func (*invalidType) Type() Type { return InvalidType() }

func (*invalidType) Value(Expr) AssignableExpr { return nil }

func (*invalidType) SourceString(*File) string {
	return irkind.Invalid.String()
}

func (*invalidType) String() string {
	return irkind.Invalid.String()
}

// Specialise a type to a given target.
func (m *invalidType) Specialise(spec Specialiser) (Type, error) {
	return m, nil
}

// UnifyWith recursively unifies a type parameters with types.
func (m *invalidType) UnifyWith(unifier Unifier, typ Type) bool {
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

func (t *distinctType) Equal(_ Fetcher, target Type) (bool, error) {
	other, ok := target.(interface{ distinct() *distinctType })
	if !ok {
		return false, nil
	}
	return t == other.distinct(), nil
}

func (t *distinctType) AssignableTo(fetcher Fetcher, target Type) (bool, error) {
	return t.Equal(fetcher, target)
}

func (t *distinctType) ConvertibleTo(fetcher Fetcher, target Type) (bool, error) {
	return t.Equal(fetcher, target)
}

func (t *distinctType) Kind() irkind.Kind { return t.kind }

func (t *distinctType) SourceString(*File) string {
	return t.String()
}

func (t *distinctType) String() string {
	return t.kind.String()
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

// TypeSet represents a set of types.
type TypeSet struct {
	BaseType[*ast.InterfaceType]
	types []Type
}

var (
	anyType             = &TypeSet{}
	_       Type        = (*TypeSet)(nil)
	_       ArrayType   = (*TypeSet)(nil)
	_       assignsFrom = (*TypeSet)(nil)
)

// NewTypeSet returns a new type set given a set of types.
func NewTypeSet(src *ast.InterfaceType, types []Type) *TypeSet {
	if src == nil {
		src = &ast.InterfaceType{}
	}
	return &TypeSet{
		BaseType: BaseType[*ast.InterfaceType]{Src: src},
		types:    types,
	}
}

// AnyType returns the type for the keyword any.
func AnyType() Type {
	return anyType
}

func (*TypeSet) node() {}

// Rank of the array.
func (*TypeSet) Rank() ArrayRank { return scalarRank }

// DataType returns the type of the element.
func (s *TypeSet) DataType() Type {
	return s
}

// ArrayType returns the source code defining the type.
// Always returns nil.
func (s *TypeSet) ArrayType() ast.Expr {
	return s.BaseType.Src
}

// Kind returns the scalar kind.
func (s *TypeSet) Kind() irkind.Kind { return irkind.Interface }

// Equal returns true if other is the exact same type set.
func (s *TypeSet) Equal(fetcher Fetcher, target Type) (bool, error) {
	targetSet, ok := target.(*TypeSet)
	if !ok {
		return false, nil
	}
	if s == targetSet {
		return true, nil
	}
	if len(s.types) != len(targetSet.types) {
		return false, nil
	}
	for i, typ := range s.types {
		if ok, err := typ.Equal(fetcher, targetSet.types[i]); !ok {
			if err != nil {
				err = fmt.Errorf("type set %q not equal to %q: %w", s, targetSet, err)
			}
			return false, err
		}
	}
	return true, nil
}

// AssignableTo reports whether a value of the type can be assigned to another.
func (s *TypeSet) AssignableTo(fetcher Fetcher, target Type) (bool, error) {
	if targetSet, ok := target.(*TypeSet); ok {
		return targetSet.assignableFrom(fetcher, s)
	}
	for _, typ := range s.types {
		if ok, _ := typ.AssignableTo(fetcher, target); !ok {
			return false, nil
		}
	}
	return len(s.types) > 0, nil
}

// AssignableFrom reports whether a given source type is assignable to any members of the set.
func (s *TypeSet) assignableFrom(fetcher Fetcher, source Type) (bool, error) {
	if len(s.types) == 0 {
		return true, nil
	}

	if sourceSet, ok := source.(*TypeSet); ok {
		return s.containsTypes(fetcher, sourceSet), nil
	}
	for _, typ := range s.types {
		if ok, _ := source.AssignableTo(fetcher, typ); ok {
			return true, nil
		}
	}
	return false, nil
}

// ConvertibleTo reports whether a value of the type can be converted to another
// (using static type casting).
func (s *TypeSet) ConvertibleTo(fetcher Fetcher, target Type) (bool, error) {
	if _, ok := target.(*TypeSet); ok {
		return s.Equal(fetcher, target)
	}
	for _, typ := range s.types {
		if ok, _ := typ.ConvertibleTo(fetcher, target); ok {
			return true, nil
		}
	}
	return false, nil
}

// Specialise a type to a given target.
func (s *TypeSet) Specialise(Specialiser) (Type, error) {
	return s, nil
}

// UnifyWith recursively unifies a type parameters with types.
func (*TypeSet) UnifyWith(unifier Unifier, typ Type) bool {
	return true
}

// TypeInclude returns if a typ is included in a type or not.
func TypeInclude(fetcher Fetcher, set Type, typ Type) (bool, error) {
	set = simplifyType(set)
	typ = simplifyType(typ)
	_, isSetNamed := set.(*NamedType)
	_, isTypeNamed := typ.(*NamedType)
	if isSetNamed && isTypeNamed {
		return set.Equal(fetcher, typ)
	}
	return typeInclude(fetcher, set, typ)
}

func typeInclude(fetcher Fetcher, set Type, typ Type) (bool, error) {
	set = Underlying(set)
	typeSet, typeSetOk := set.(*TypeSet)
	if !typeSetOk {
		return set.Equal(fetcher, typ)
	}
	for _, sub := range typeSet.types {
		in, err := typeInclude(fetcher, sub, typ)
		if err != nil {
			return false, err
		}
		if in {
			return true, nil
		}
	}
	return false, nil
}

// Source returns the source code defining the type.
// Always returns nil.
func (s *TypeSet) Source() ast.Node { return s.ArrayType() }

// Zero returns a zero expression of the same type.
func (s *TypeSet) Zero() AssignableExpr {
	panic("unimplemented")
}

// ElementType returns the type of an element.
func (s *TypeSet) ElementType() (Type, bool) {
	sub := &TypeSet{}
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
func (s *TypeSet) Value(x Expr) AssignableExpr {
	return &TypeValExpr{X: x, Typ: s}
}

func (s *TypeSet) interfaceString(types []string) string {
	if len(types) == 0 {
		return "any"
	}
	var b strings.Builder
	b.WriteString("interface { ")
	b.WriteString(strings.Join(types, "|"))
	b.WriteString(" }")
	return b.String()
}

// SourceString returns a string representation of the signature of a function.
func (s *TypeSet) SourceString(context *File) string {
	types := make([]string, len(s.types))
	for i, typ := range s.types {
		types[i] = typ.SourceString(context)
	}
	return s.interfaceString(types)
}

// String representation of the type.
func (s *TypeSet) String() string {
	types := make([]string, len(s.types))
	for i, typ := range s.types {
		types[i] = typ.String()
	}
	return s.interfaceString(types)
}

func (s *TypeSet) equalArray(fetcher Fetcher, target ArrayType) (bool, error) {
	return s.Equal(fetcher, target)
}

// hasCapability returns true if and only if the capability applies to all types in the set.
// Returns false if the type set is empty.
func (s *TypeSet) hasCapability(f func(Type) bool) bool {
	for _, typ := range s.types {
		if !f(typ) {
			return false
		}
	}
	return len(s.types) > 0
}

// containsType returns true if the given type is present in the set.
func (s *TypeSet) containsType(fetcher Fetcher, wantType Type) bool {
	for _, typ := range s.types {
		if eq, _ := wantType.Equal(fetcher, typ); eq {
			return true
		}
	}
	return false
}

// containsTypes returns true if all the given types are present in the set.
func (s *TypeSet) containsTypes(fetcher Fetcher, types *TypeSet) bool {
	for _, wantType := range types.types {
		if !s.containsType(fetcher, wantType) {
			return false
		}
	}
	return true
}

func fieldListAtomicIR() *FieldList {
	group := &FieldGroup{
		Type: &TypeValExpr{
			Typ: TypeFromKind(irkind.IR),
		},
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

// AtomTypeExpr returns a type expression for an atomic type.
func AtomTypeExpr(typ Type) *TypeValExpr {
	return &TypeValExpr{
		X:   typ,
		Typ: typ,
	}
}

// ToArrayType converts a type into an array type.
// Returns nil if the conversion is not possible
func ToArrayType(typ Type) ArrayType {
	switch typT := typ.(type) {
	case *TypeParam:
		return ToArrayType(typT.Field.Group.Type.Typ)
	case *NamedType:
		return ToArrayType(typT.Underlying.Typ)
	case *TypeSet:
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
	equal(fetcher Fetcher, x Type) (bool, error)
}

// Equal returns true if x and y are the same type.
func Equal(fetcher Fetcher, x, y Type) (bool, error) {
	if x == y {
		return true, nil
	}
	ey, isEqualer := y.(equaler)
	if isEqualer {
		return ey.equal(fetcher, x)
	}
	return x.Equal(fetcher, y)
}

type assigner interface {
	assignableFrom(fetcher Fetcher, x Type) (bool, error)
}

func simplifyType(t Type) Type {
	typeParam, ok := t.(*TypeParam)
	if !ok {
		return t
	}
	return typeParam.Field.Group.Type.Typ
}

// IsInvalidType returns true if a type is invalid.
func IsInvalidType(typ Type) bool {
	return typ == nil || typ.Kind() == irkind.Invalid
}

// AssignableTo reports whether a value of the type can be assigned to another.
func AssignableTo(fetcher Fetcher, x, y Type) (bool, error) {
	if IsInvalidType(x) || IsInvalidType(y) {
		// An error should have already been reported. We skip the check
		// to prevent additional confusing errors.
		return true, nil
	}
	x = simplifyType(x)
	y = simplifyType(y)
	if x == y {
		return true, nil
	}
	ey, isAssigner := y.(assigner)
	if isAssigner {
		return ey.assignableFrom(fetcher, x)
	}
	return x.AssignableTo(fetcher, y)
}

// AssignableToAt reports whether a value of the type can be assigned to another.
// An error is added at the specified code location if the value cannot be assigned.
func AssignableToAt(fetcher Fetcher, pos ast.Node, src, dst Type) bool {
	assignable, err := AssignableTo(fetcher, src, dst)
	if err != nil {
		return fetcher.Err().AppendAt(pos, err)
	}
	if !assignable {
		return fetcher.Err().Appendf(pos, "cannot use %s as %s value in assignment", src.String(), dst.String())
	}
	return true
}
