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

// Source returns the source node defining the type.
func (m *BaseType[T]) Source() ast.Node {
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

func (t *metaType) Kind() Kind { return MetaTypeKind }

func (t *metaType) SourceString(*File) string {
	return t.String()
}

func (t *metaType) String() string {
	return MetaTypeKind.String()
}

var metaTypeT = &metaType{
	BaseType: BaseType[*ast.Ident]{Src: &ast.Ident{Name: MetaTypeKind.String()}},
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

func (*invalidType) Source() ast.Node { return nil }

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

func (*invalidType) Kind() Kind { return InvalidKind }

func (*invalidType) NameDef() *ast.Ident { return nil }

func (*invalidType) Type() Type { return InvalidType() }

func (*invalidType) Value(Expr) AssignableExpr { return nil }

func (*invalidType) SourceString(*File) string {
	return InvalidKind.String()
}

func (*invalidType) String() string {
	return InvalidKind.String()
}

type distinctType struct {
	BaseType[ast.Expr]
	kind Kind
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

func (t *distinctType) Kind() Kind { return t.kind }

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

var unknownT = &unknownType{distinctType: distinctType{kind: UnknownKind}}

// UnknownType returns the unknown type.
func UnknownType() Type {
	return unknownT
}

// keywordTyp is the type returned by function with no results.
type keywordTyp struct {
	distinctType
}

var keywordT = &keywordTyp{distinctType: distinctType{kind: FuncKind}}

func keywordType() Type {
	return keywordT
}

// voidType is the type returned by function with no results.
type voidType struct {
	distinctType
}

var voidT = &voidType{distinctType: distinctType{kind: VoidKind}}

// VoidType returns the void type.
func VoidType() Type {
	return voidT
}

type packageType struct {
	distinctType
}

var packageT = &packageType{distinctType: distinctType{kind: PackageKind}}

// PackageType returns the package type.
func PackageType() Type {
	return packageT
}

type rankType struct {
	atomicType
}

var rankT = &rankType{atomicType: atomicType{Knd: RankKind}}

// RankType returns the rank type.
func RankType() Type {
	return rankT
}

type boolType struct {
	atomicType
}

var boolT = &boolType{atomicType: atomicType{Knd: BoolKind}}

// BoolType returns the type for a boolean.
func BoolType() Type {
	return boolT
}

type bfloat16Type struct {
	atomicType
}

var bfloat16T = &bfloat16Type{atomicType: atomicType{Knd: Bfloat16Kind}}

// Bfloat16Type returns the type for a bfloat16.
func Bfloat16Type() Type {
	return bfloat16T
}

type float32Type struct {
	atomicType
}

var float32T = &float32Type{atomicType: atomicType{Knd: Float32Kind}}

// Float32Type returns the type for a float32.
func Float32Type() Type {
	return float32T
}

type float64Type struct {
	atomicType
}

var float64T = &float64Type{atomicType: atomicType{Knd: Float64Kind}}

// Float64Type returns the type for a float64.
func Float64Type() Type {
	return float64T
}

type int32Type struct {
	atomicType
}

var int32T = &int32Type{atomicType: atomicType{Knd: Int32Kind}}

// Int32Type returns the type for a int32.
func Int32Type() Type {
	return int32T
}

type int64Type struct {
	atomicType
}

var int64T = &int64Type{atomicType: atomicType{Knd: Int64Kind}}

// Int64Type returns the type for a int64.
func Int64Type() Type {
	return int64T
}

type numberFloatType struct {
	atomicType
}

var numberFloatT = &numberFloatType{atomicType: atomicType{Knd: NumberFloatKind}}

type intidxType struct {
	atomicType
}

var (
	intidxT         = &intidxType{atomicType: atomicType{Knd: IntIdxKind}}
	axisIndicesType = &SliceType{
		DType: &TypeValExpr{Typ: IntIndexType()},
		Rank:  1,
	}
)

// IntIndexType returns the type for intidx, that is the length of an axis.
func IntIndexType() Type {
	return intidxT
}

// IntIndexSliceType returns a slice of axis lengths type.
func IntIndexSliceType() *SliceType {
	return axisIndicesType
}

type intlenType struct {
	atomicType
}

var (
	intlenT         = &intlenType{atomicType: atomicType{Knd: IntLenKind}}
	axisLengthsType = &SliceType{
		BaseType: BaseType[*ast.ArrayType]{Src: &ast.ArrayType{}},
		DType: &TypeValExpr{
			X:   IntLenType(),
			Typ: IntLenType(),
		},
		Rank: 1,
	}
)

// IntLenType returns the type for intlen, that is the length of an axis.
func IntLenType() Type {
	return intlenT
}

// IntLenSliceType returns a slice of axis lengths type.
func IntLenSliceType() *SliceType {
	return axisLengthsType
}

// NumberFloatType returns the type for a float number.
func NumberFloatType() Type {
	return numberFloatT
}

type numberIntType struct {
	atomicType
}

var numberIntT = &numberIntType{atomicType: atomicType{Knd: NumberIntKind}}

// NumberIntType returns the type for an integer number.
func NumberIntType() Type {
	return numberIntT
}

type stringType struct {
	atomicType
}

var stringT = &stringType{atomicType: atomicType{Knd: StringKind}}

// StringType returns the type for a string.
func StringType() Type {
	return stringT
}

type uint32Type struct {
	atomicType
}

var uint32T = &uint32Type{atomicType: atomicType{Knd: Uint32Kind}}

// Uint32Type returns the type for a uint32.
func Uint32Type() Type {
	return uint32T
}

type uint64Type struct {
	atomicType
}

var uint64T = &uint64Type{atomicType: atomicType{Knd: Uint64Kind}}

// Uint64Type returns the type for a uint64.
func Uint64Type() ArrayType {
	return uint64T
}

// TypeSet represents a set of types.
type TypeSet struct {
	BaseType[*ast.InterfaceType]
	Typs []Type
}

var (
	anyType             = &TypeSet{}
	_       Type        = (*TypeSet)(nil)
	_       ArrayType   = (*TypeSet)(nil)
	_       assignsFrom = (*TypeSet)(nil)
)

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
func (s *TypeSet) Kind() Kind { return InterfaceKind }

// Equal returns true if other is the exact same type set.
func (s *TypeSet) Equal(fetcher Fetcher, target Type) (bool, error) {
	targetSet, ok := target.(*TypeSet)
	if !ok {
		return false, nil
	}
	if s == targetSet {
		return true, nil
	}
	if len(s.Typs) != len(targetSet.Typs) {
		return false, nil
	}
	for i, typ := range s.Typs {
		if ok, err := typ.Equal(fetcher, targetSet.Typs[i]); !ok {
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
	for _, typ := range s.Typs {
		if ok, _ := typ.AssignableTo(fetcher, target); !ok {
			return false, nil
		}
	}
	return len(s.Typs) > 0, nil
}

// AssignableFrom reports whether a given source type is assignable to any members of the set.
func (s *TypeSet) assignableFrom(fetcher Fetcher, source Type) (bool, error) {
	if len(s.Typs) == 0 {
		return true, nil
	}

	if sourceSet, ok := source.(*TypeSet); ok {
		return s.containsTypes(fetcher, sourceSet), nil
	}
	for _, typ := range s.Typs {
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
	for _, typ := range s.Typs {
		if ok, _ := typ.ConvertibleTo(fetcher, target); ok {
			return true, nil
		}
	}
	return false, nil
}

// TypeInclude returns if a typ is included in a type or not.
func TypeInclude(fetcher Fetcher, set Type, typ Type) (bool, error) {
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
	for _, sub := range typeSet.Typs {
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
	for _, typ := range s.Typs {
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
		sub.Typs = append(sub.Typs, sType)
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
	types := make([]string, len(s.Typs))
	for i, typ := range s.Typs {
		types[i] = typ.SourceString(context)
	}
	return s.interfaceString(types)
}

// String representation of the type.
func (s *TypeSet) String() string {
	types := make([]string, len(s.Typs))
	for i, typ := range s.Typs {
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
	for _, typ := range s.Typs {
		if !f(typ) {
			return false
		}
	}
	return len(s.Typs) > 0
}

// containsType returns true if the given type is present in the set.
func (s *TypeSet) containsType(fetcher Fetcher, wantType Type) bool {
	for _, typ := range s.Typs {
		if eq, _ := wantType.Equal(fetcher, typ); eq {
			return true
		}
	}
	return false
}

// containsTypes returns true if all the given types are present in the set.
func (s *TypeSet) containsTypes(fetcher Fetcher, types *TypeSet) bool {
	for _, wantType := range types.Typs {
		if !s.containsType(fetcher, wantType) {
			return false
		}
	}
	return true
}

func fieldListAtomicIR() *FieldList {
	group := &FieldGroup{
		Type: &TypeValExpr{
			Typ: TypeFromKind(IRKind),
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

// AssignableTo reports whether a value of the type can be assigned to another.
func AssignableTo(fetcher Fetcher, x, y Type) (bool, error) {
	if x == y {
		return true, nil
	}
	ey, isAssigner := y.(assigner)
	if isAssigner {
		return ey.assignableFrom(fetcher, x)
	}
	return x.AssignableTo(fetcher, y)
}
