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

// Package ir is the GX Intermediate Representation (IR) tree.
// The tree is built by the GX builder
// [google3/third_party/gxlang/gx/build/builder/builder]
// from GX source code.
//
// The structure and semantic is modeled after the go/ast package.
package ir

import (
	"fmt"
	"go/ast"
	"go/token"
	"iter"
	"maps"
	"math/big"
	"path/filepath"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/pkg/errors"
	"github.com/gx-org/backend/dtype"
	"github.com/gx-org/backend/shape"
	"github.com/gx-org/gx/build/ir/annotations"
	"github.com/gx-org/gx/build/ir/irkind"
)

// ----------------------------------------------------------------------------
// Types of node in the tree.
type (
	// IR in the tree.
	IR interface {
		// node marks a structure as a node structure.
		// It prevents external implementations of the interface.
		// It prevents using arbitrary structure in this package to be used as nodes.
		node()
	}

	// Node is a node with a position in GX source code.
	Node interface {
		IR
		Node() ast.Node
	}

	// Storage is a node that can store a value.
	// (e.g. a field in a function)
	Storage interface {
		Node
		storage()
		NameDef() *ast.Ident
		Same(Storage) bool
		Type() Type
	}

	// StorageWithValue is a storage to which a value has been assigned to.
	// (e.g. a variable that has been defined)
	StorageWithValue interface {
		storageValue()
		Storage
		// Value returns the value stored given the expression pointing to the storage.
		Value(Expr) Expr
	}

	// Value is a value used by the compiler. It can be any thing with a type (package, constant, ...).
	// This is a superset of Expr.
	Value interface {
		Node
		Type() Type
	}

	// WithStore is an expression which points to a storage.
	WithStore interface {
		Store() Storage
	}
)

// ----------------------------------------------------------------------------
// Types definition.
type (
	// Specialiser provides methods to specialise a type.
	Specialiser interface {
		Fetcher
		TypeOf(tParamName string) Type
		ValueOf(axLengthname string) Element
	}

	// Unifier provides methods to unify types.
	Unifier interface {
		Fetcher
		Node() ast.Node
		DefineTParam(tp *TypeParam, typ Type) bool
		DefineAxis(*AxisStmt, []AxisLengths) ([]AxisLengths, bool)
	}

	// Type of a value.
	Type interface {
		StorageWithValue

		// Kind of the type.
		Kind() irkind.Kind

		// Equal returns true if other is the same type.
		Equal(Fetcher, Type) (bool, error)

		// AssignableTo reports whether a value of the type can be assigned to another.
		AssignableTo(Fetcher, Type) (bool, error)

		// ConvertibleTo reports whether a value of the type can be converted to another
		// (using static type casting).
		ConvertibleTo(Fetcher, Type) (bool, error)

		// Specialise a type to a given target.
		Specialise(Specialiser) (Type, error)

		// UnifyWith recursively unifies a type parameters with types.
		UnifyWith(Unifier, Type) bool

		// SourceString returns a reference to the type given a file context.
		SourceString(file *File) string

		// String representation of the type.
		String() string
	}

	// Zeroer is a type able to create a zero value of the type as an expression.
	Zeroer interface {
		Type
		Zero() Expr
	}

	// SlicerType is a type supporting slice operations:
	// (e.g. slicerType[i])
	SlicerType interface {
		Type
		// ElementType returns the type of an element, that is the type
		// returned by a slice at the top level. For example:
		// float32[2][3][4] returns float32[3][4]
		// If the number of axis is 0, nil is returned.
		ElementType() (Type, bool)
	}

	// ArrayType is a type with a rank.
	ArrayType interface {
		Zeroer
		Node
		SlicerType

		// ArrayType returns the source code defining the array type.
		// May be nil.
		ArrayType() ast.Expr

		// Rank returns the rank of the array,
		// that is, a list of the array's axes.
		Rank() ArrayRank

		// DataType returns the element type of the array.
		DataType() Type

		equalArray(Fetcher, ArrayType) (bool, error)
	}

	// assignsFrom allows types to control how they are assigned to; other types may opt to use this
	// logic when present (but aren't currently required to).
	assignsFrom interface {
		// assignableFrom reports whether a value of some type can be assigned to the receiver.
		assignableFrom(Fetcher, Type) (bool, error)
	}

	// convertsFrom allows types to control how they are converted to; other types may opt to use this
	// logic when present (but aren't currently required to).
	convertsFrom interface {
		// convertibleFrom reports whether a value of some type can be converted to the receiver.
		convertibleFrom(Fetcher, Type) (bool, error)
	}

	// NamedType defines a new type from an existing type.
	NamedType struct {
		BaseType[ast.Expr]

		Src        *ast.TypeSpec
		File       *File
		Underlying *TypeValExpr

		Methods []PkgFunc
	}

	// StructType defines the type of a structure.
	StructType struct {
		BaseType[*ast.StructType]

		Fields *FieldList
	}

	// InterfaceType defines the type of an interface.
	InterfaceType struct {
		BaseType[ast.Expr]
	}

	// SliceType defines the type for a slice.
	SliceType struct {
		BaseType[*ast.ArrayType]

		DType *TypeValExpr
		Rank  int
	}

	// arrayType defines the type of an array from code.
	arrayType struct {
		BaseType[ast.Expr]

		DTypeF Type
		RankF  ArrayRank
	}

	// BuiltinType is an opaque type maintained by the backend.
	// These are the only atomic types that can have a state.
	BuiltinType struct {
		BaseType[ast.Expr]

		Impl any
	}

	// TupleType is the type of the result of a function that returns more than one value.
	TupleType struct {
		BaseType[ast.Expr]

		Types []Type
	}

	// TypeParam is a type mapping to a field name in a function type parameters.
	TypeParam struct {
		BaseType[ast.Expr]

		Field *Field
	}
)

var (
	_ StorageWithValue = (*NamedType)(nil)
	_ Type             = (*NamedType)(nil)
	_ Type             = (*StructType)(nil)
	_ Type             = (*InterfaceType)(nil)
	_ SlicerType       = (*SliceType)(nil)
	_ Type             = (*TupleType)(nil)
	_ ArrayType        = (*atomicType)(nil)
	_ ArrayType        = (*arrayType)(nil)
	_ Type             = (*TypeParam)(nil)
)

// DefaultFloatType is the default type used for a scalar.
var DefaultFloatType = Float32Type()

var (

	// DefaultIntType is the default type used for an integer.
	DefaultIntType = TypeFromKind(irkind.Int64)
)

// Int is the default integer for indices, for loops, etc.
// It needs to match DefaultIntKind above
type Int = int64

func (*TupleType) node() {}

// Kind returns the scalar kind.
func (s *TupleType) Kind() irkind.Kind { return irkind.Tuple }

func (s *TupleType) apply(fetcher Fetcher, target Type, f func(Type, Fetcher, Type) (bool, error)) (bool, error) {
	targetTuple, ok := target.(*TupleType)
	if !ok {
		return false, nil
	}
	if len(s.Types) != len(targetTuple.Types) {
		return false, nil
	}
	for n, typ := range s.Types {
		if ok, err := f(typ, fetcher, targetTuple.Types[n]); !ok {
			return ok, err
		}
	}
	return true, nil
}

// Equal returns true if other is the same type.
func (s *TupleType) Equal(fetcher Fetcher, target Type) (bool, error) {
	return s.apply(fetcher, target, (Type).Equal)
}

// AssignableTo reports whether a value of the type can be assigned to another.
func (s *TupleType) AssignableTo(fetcher Fetcher, target Type) (bool, error) {
	return s.apply(fetcher, target, (Type).AssignableTo)
}

// ConvertibleTo reports whether a value of the type can be converted to another
// (using static type casting). Always returns false.
func (s *TupleType) ConvertibleTo(fetcher Fetcher, target Type) (bool, error) {
	return s.apply(fetcher, target, (Type).ConvertibleTo)
}

// Value returns a value pointing to the receiver.
func (s *TupleType) Value(x Expr) Expr {
	return TypeExpr(x, s)
}

// SourceString returns a string representation of the signature of a function.
func (s *TupleType) SourceString(context *File) string {
	return s.String()
}

// String representation of the type.
func (s *TupleType) String() string {
	ss := make([]string, len(s.Types))
	for i, typ := range s.Types {
		ss[i] = typ.String()
	}
	return fmt.Sprintf("(%s)", strings.Join(ss, ","))
}

func (*InterfaceType) node() {}

// Kind returns the interface kind.
func (s *InterfaceType) Kind() irkind.Kind { return irkind.Interface }

// Equal returns true if other is the same type.
func (s *InterfaceType) Equal(Fetcher, Type) (bool, error) { return false, nil }

// AssignableTo reports whether a value of the type can be assigned to another.
// Always returns false.
func (s *InterfaceType) AssignableTo(Fetcher, Type) (bool, error) { return false, nil }

// ConvertibleTo reports whether a value of the type can be converted to another
// (using static type casting). Always returns false.
func (s *InterfaceType) ConvertibleTo(Fetcher, Type) (bool, error) { return false, nil }

// Value returns a value pointing to the receiver.
func (s *InterfaceType) Value(x Expr) Expr {
	return TypeExpr(x, s)
}

// SourceString returns a reference to the type given a file context.
func (s *InterfaceType) SourceString(context *File) string {
	return s.String()
}

// String representation of the type.
func (s *InterfaceType) String() string { return s.Kind().String() }

// IsValid returns true if the type is valid.
func IsValid(tp Type) bool {
	return tp.Kind() != irkind.Invalid
}

func (*BuiltinType) node() {}

// Kind returns the scalar kind.
func (s *BuiltinType) Kind() irkind.Kind { return irkind.Builtin }

// Equal returns true if other is the same type.
func (s *BuiltinType) Equal(_ Fetcher, other Type) (bool, error) {
	otherT, ok := other.(*BuiltinType)
	if !ok {
		return false, nil
	}
	return s.Impl == otherT.Impl, nil
}

// AssignableTo reports if the type can be assigned to other.
func (s *BuiltinType) AssignableTo(fetcher Fetcher, target Type) (bool, error) {
	return s.Equal(fetcher, target)
}

// ConvertibleTo reports whether a value of the type can be converted to another
// (using static type casting).
func (s *BuiltinType) ConvertibleTo(fetcher Fetcher, target Type) (bool, error) {
	return s.Equal(fetcher, target)
}

// Value returns a value pointing to the receiver.
func (s *BuiltinType) Value(x Expr) Expr {
	return TypeExpr(x, s)
}

// SourceString returns a string representation of the signature of a function.
func (s *BuiltinType) SourceString(context *File) string {
	return s.String()
}

// String representation of the type.
func (s *BuiltinType) String() string {
	return fmt.Sprint(s.Impl)
}

func (*atomicType) node()   {}
func (*atomicType) atomic() {}

// Kind returns the scalar kind.
func (s *atomicType) Kind() irkind.Kind { return s.Knd }

func (s *atomicType) equalAtomic(other ArrayType) (bool, error) {
	return s.Knd == other.Kind(), nil
}

func (s *atomicType) equalArray(fetcher Fetcher, other ArrayType) (bool, error) {
	dtypeEq, err := s.Equal(fetcher, other.DataType())
	if !dtypeEq || err != nil {
		return false, err
	}
	return s.Rank().Equal(fetcher, other.Rank())
}

// Equal returns true if other is the same type.
func (s *atomicType) Equal(fetcher Fetcher, other Type) (bool, error) {
	otherT, ok := other.(ArrayType)
	if !ok {
		return false, nil
	}
	if otherT.Rank().IsAtomic() {
		return s.equalAtomic(otherT)
	}
	return s.equalArray(fetcher, otherT)
}

var scalarRank = &Rank{}

// Rank of the array.
func (s *atomicType) Rank() ArrayRank { return scalarRank }

// AssignableTo reports if the type can be assigned to other.
func (s *atomicType) AssignableTo(fetcher Fetcher, target Type) (bool, error) {
	if assignFrom, ok := target.(assignsFrom); ok {
		return assignFrom.assignableFrom(fetcher, s)
	}

	targetT, ok := target.(ArrayType)
	if !ok {
		return false, nil
	}
	if targetT.Rank().IsAtomic() {
		if s.Knd == irkind.NumberInt && (IsInteger(target) || IsFloat(target)) {
			return true, nil
		}
		if s.Knd == irkind.NumberFloat && IsFloat(target) {
			return true, nil
		}
		return s.equalAtomic(targetT)
	}
	dtypeEq, err := s.AssignableTo(fetcher, targetT.DataType())
	if !dtypeEq || err != nil {
		return false, err
	}
	rankOk, err := s.Rank().AssignableTo(fetcher, targetT.Rank())
	if !rankOk || err != nil {
		return rankOk, err
	}
	return true, nil
}

// ConvertibleTo reports whether a value of the type can be converted to another
// (using static type casting).
func (s *atomicType) ConvertibleTo(fetcher Fetcher, target Type) (bool, error) {
	if convertFrom, ok := target.(convertsFrom); ok {
		return convertFrom.convertibleFrom(fetcher, s)
	}

	targetT, ok := target.(ArrayType)
	if !ok {
		return false, nil
	}
	return SupportOperators(s) == SupportOperators(targetT.DataType()), nil
}

// Specialise a type to a given target.
func (s *atomicType) Specialise(Specialiser) (Type, error) {
	return s, nil
}

// Node returns the source code defining the type.
// Always returns nil.
func (s *atomicType) Node() ast.Node {
	return s.ArrayType()
}

// Value returns a value pointing to the receiver.
func (s *atomicType) Value(x Expr) Expr {
	return TypeExpr(x, s)
}

// ElementType returns an invalid type because an atomic type cannot be sliced.
func (s *atomicType) ElementType() (Type, bool) {
	return InvalidType(), false
}

// DataType returns the type of the element.
func (s *atomicType) DataType() Type {
	return s
}

// ArrayType returns the source code defining the type.
// Always returns nil.
func (s *atomicType) ArrayType() ast.Expr {
	return &ast.Ident{Name: s.Knd.String()}
}

var (
	zero      = &NumberInt{Src: &ast.BasicLit{Value: "0"}, Val: big.NewInt(0)}
	zeroFloat = &NumberFloat{Src: &ast.BasicLit{Value: "0.0"}, Val: big.NewFloat(0)}
)

// Zero returns a zero expression of the same type.
func (s *atomicType) Zero() Expr {
	return &NumberCastExpr{
		X:   zero,
		Typ: s,
	}
}

// SourceString returns a reference to the type given a file context.
func (s *atomicType) SourceString(*File) string {
	return s.String()
}

// String representation of the type.
func (s *atomicType) String() string {
	return s.Kind().String()
}

// MethodByName returns a method given its name, or nil if not method has that name.
func (s *NamedType) MethodByName(name string) PkgFunc {
	for _, method := range s.Methods {
		if method.Name() == name {
			return method
		}
	}
	return nil
}

// Kind of the underlying type.
func (s *NamedType) Kind() irkind.Kind { return s.Underlying.Val().Kind() }

// Equal returns true if other is the same type.
func (s *NamedType) Equal(fetcher Fetcher, other Type) (bool, error) {
	otherT, ok := other.(*NamedType)
	if !ok {
		return false, nil
	}
	if s == otherT {
		return true, nil
	}
	if s.FullName() != otherT.FullName() {
		return false, nil
	}
	return s.Underlying.Val().Equal(fetcher, otherT.Underlying.Val())
}

// AssignableTo reports if the type can be assigned to other.
func (s *NamedType) AssignableTo(fetcher Fetcher, target Type) (bool, error) {
	return s.Equal(fetcher, target)
}

// AssignableFrom reports whether a given source type is assignable to a named type.
func (s *NamedType) assignableFrom(fetcher Fetcher, source Type) (bool, error) {
	if target, ok := s.Underlying.Val().(assignsFrom); ok {
		return target.assignableFrom(fetcher, source)
	}
	return s.Equal(fetcher, source)
}

// ConvertibleTo reports whether a value of the type can be converted to another
// (using static type casting).
func (s *NamedType) ConvertibleTo(fetcher Fetcher, target Type) (bool, error) {
	typeSet, ok := Underlying(s.Underlying.Val()).(*TypeSet)
	if ok {
		return typeSet.ConvertibleTo(fetcher, target)
	}
	switch targetT := target.(type) {
	case *NamedType:
		return s.Equal(fetcher, targetT)
	case *TypeParam:
		return s.ConvertibleTo(fetcher, targetT.Field.Group.Type.Val())
	}
	return s.Underlying.Val().ConvertibleTo(fetcher, target)
}

func (s *NamedType) convertibleFrom(fetcher Fetcher, source Type) (bool, error) {
	return source.ConvertibleTo(fetcher, s.Underlying.Val())
}

// FullName returns the full name of the type, that is the full package path and the type name.
func (s *NamedType) FullName() string {
	return s.File.Package.Path + "." + s.Name()
}

// Node returns the node in the AST tree.
func (s *NamedType) Node() ast.Node { return s.Src }

// Name of the type.
func (s *NamedType) Name() string {
	return s.NameDef().Name
}

// NameDef returns the name defining the storage.
func (s *NamedType) NameDef() *ast.Ident {
	return s.Src.Name
}

// Type returns the type of an expression.
func (s *NamedType) Type() Type {
	return MetaType()
}

// Value assigned to the constant.
func (s *NamedType) Value(x Expr) Expr {
	return TypeExpr(x, s)
}

// Same returns true if the other storage is this storage.
func (s *NamedType) Same(o Storage) bool {
	return Storage(s) == o
}

// SourceString returns a reference to the type given a file context.
func (s *NamedType) SourceString(context *File) string {
	if context == nil {
		return s.String()
	}
	if s.File == nil || s.File.Package == context.Package {
		return s.Name()
	}
	return s.File.Package.Name.Name + "." + s.Name()
}

// String representation of the type.
func (s *NamedType) String() string {
	if s.File == nil {
		return s.Name()
	}
	return s.File.Package.Name.Name + "." + s.Name()
}

// Specialise a type to a given target.
func (s *NamedType) Specialise(spec Specialiser) (Type, error) {
	_, err := s.Underlying.Val().Specialise(spec)
	return s, err
}

// Package returns the package to which the type belongs to.
func (s *NamedType) Package() *Package {
	return s.File.Package
}

// Underlying returns the underlying type of a type.
func Underlying(typ Type) Type {
	switch typT := typ.(type) {
	case *NamedType:
		return Underlying(typT.Underlying.Val())
	case *TypeParam:
		return Underlying(typT.Field.Type())
	default:
		return typ
	}
}

func (*StructType) node() {}

// Kind returns the structure kind.
func (s *StructType) Kind() irkind.Kind { return irkind.Struct }

// Equal returns true if other is the same type.
func (s *StructType) Equal(_ Fetcher, other Type) (bool, error) {
	otherT, ok := other.(*StructType)
	if !ok {
		return false, nil
	}
	return s == otherT, nil
}

// AssignableTo reports if the type can be assigned to other.
func (s *StructType) AssignableTo(fetcher Fetcher, target Type) (bool, error) {
	return s.Equal(fetcher, target)
}

// ConvertibleTo reports whether a value of the type can be converted to another
// (using static type casting).
func (s *StructType) ConvertibleTo(fetcher Fetcher, target Type) (bool, error) {
	return false, nil
}

// Node returns the node in the AST tree.
func (s *StructType) Node() ast.Node { return s.Src }

// Value returns a value pointing to the receiver.
func (s *StructType) Value(x Expr) Expr {
	return TypeExpr(x, s)
}

// SourceString returns a reference to the type given a file context.
func (s *StructType) SourceString(context *File) string {
	return s.String()
}

// String representation of the type.
func (s *StructType) String() string {
	return s.Kind().String()
}

// Specialise a type to a given target.
func (s *StructType) Specialise(spec Specialiser) (Type, error) {
	return s, nil
}

// NumFields returns the number of fields in a structure.
func (s *StructType) NumFields() int {
	n := 0
	for _, group := range s.Fields.List {
		n += group.NumFields()
	}
	return n
}

func (*SliceType) node() {}

// Kind returns the slide kind.
func (s *SliceType) Kind() irkind.Kind { return irkind.Slice }

// Equal returns true if other is the same type.
func (s *SliceType) Equal(fetcher Fetcher, other Type) (bool, error) {
	otherT, ok := other.(*SliceType)
	if !ok {
		return false, nil
	}
	if s.Rank != otherT.Rank {
		return false, nil
	}
	return otherT.DType.Val().Equal(fetcher, s.DType.Val())
}

// AssignableTo reports if the type can be assigned to other.
func (s *SliceType) AssignableTo(fetcher Fetcher, target Type) (bool, error) {
	return s.Equal(fetcher, target)
}

// ConvertibleTo reports whether a value of the type can be converted to another
// (using static type casting).
func (s *SliceType) ConvertibleTo(fetcher Fetcher, target Type) (bool, error) {
	return false, nil
}

// Value returns a value pointing to the receiver.
func (s *SliceType) Value(x Expr) Expr {
	return TypeExpr(x, s)
}

// Node returns the node in the AST tree.
func (s *SliceType) Node() ast.Node { return s.Src }

// ElementType returns the type of an element in that slice.
func (s *SliceType) ElementType() (Type, bool) {
	if s.Rank <= 0 {
		return InvalidType(), false
	}
	if s.Rank == 1 {
		return s.DType.Val(), true
	}
	return &SliceType{
		BaseType: s.BaseType,
		DType:    s.DType,
		Rank:     s.Rank - 1,
	}, true
}

func (s *SliceType) rankString(dtype string) string {
	rank := "[?]"
	if s.Rank > 0 {
		rank = strings.Repeat("[]", s.Rank)
	}
	return rank + dtype
}

// SourceString returns a reference to the type given a file context.
func (s *SliceType) SourceString(context *File) string {
	return s.rankString(s.DType.Val().SourceString(context))
}

// Specialise a type to a given target.
func (s *SliceType) Specialise(spec Specialiser) (Type, error) {
	return s, nil
}

// UnifyWith recursively unifies a type parameters with types.
func (s *SliceType) UnifyWith(Unifier, Type) bool {
	return true
}

// String representation of the type.
func (s *SliceType) String() string {
	dtype := "unknown"
	if s.DType != nil {
		dtype = s.DType.String()
	}
	return s.rankString(dtype)
}

func (*arrayType) node() {}

// NewArrayType returns a new array from a data type and a rank.
func NewArrayType(src ast.Expr, dtype Type, rank ArrayRank) ArrayType {
	if dtype == nil {
		dtype = UnknownType()
	}
	if rank == nil {
		rank = &RankInfer{}
	}
	return &arrayType{
		BaseType: BaseType[ast.Expr]{Src: src},
		DTypeF:   dtype,
		RankF:    rank,
	}
}

// Kind returns the tensor kind.
func (s *arrayType) Kind() irkind.Kind {
	if s.RankF == nil {
		return irkind.Invalid
	}
	if s.RankF.IsAtomic() {
		return s.DTypeF.Kind()
	}
	return irkind.Array
}

// ElementType returns the type of the element with rank-1.
func (s *arrayType) ElementType() (Type, bool) {
	subRank, ok := s.RankF.SubRank()
	if !ok {
		return InvalidType(), false
	}
	if s.RankF.IsAtomic() {
		return s.DTypeF, true
	}
	return &arrayType{
		BaseType: BaseType[ast.Expr]{
			Src: &ast.ArrayType{},
		},
		DTypeF: s.DTypeF,
		RankF:  subRank,
	}, true
}

// DataType returns the type of the data stored in the array.
func (s *arrayType) DataType() Type { return s.DTypeF }

// Rank of the array.
func (s *arrayType) Rank() ArrayRank { return s.RankF }

func (s *arrayType) assignableToArray(fetcher Fetcher, other ArrayType) (bool, error) {
	dtypeEq, err := s.DataType().AssignableTo(fetcher, other.DataType())
	if !dtypeEq || err != nil {
		return dtypeEq, err
	}
	return s.Rank().AssignableTo(fetcher, other.Rank())
}

func (s *arrayType) equalArray(fetcher Fetcher, other ArrayType) (bool, error) {
	dtypeEq, err := s.DataType().Equal(fetcher, other.DataType())
	if !dtypeEq || err != nil {
		return dtypeEq, err
	}
	return s.Rank().Equal(fetcher, other.Rank())
}

// Equal returns true if other is the same type.
func (s *arrayType) Equal(fetcher Fetcher, other Type) (bool, error) {
	otherT, ok := other.(ArrayType)
	if !ok {
		return false, nil
	}
	if otherT.Rank().IsAtomic() {
		return otherT.equalArray(fetcher, s)
	}
	return s.equalArray(fetcher, otherT)
}

func (s *arrayType) assignableFrom(fetcher Fetcher, x Type) (bool, error) {
	arrayType, isArrayType := x.(ArrayType)
	if isArrayType {
		return s.assignableToArray(fetcher, arrayType)
	}
	if !s.RankF.IsAtomic() {
		return false, nil
	}
	return AssignableTo(fetcher, x, s.DataType())
}

// AssignableTo reports if the type can be assigned to other.
func (s *arrayType) AssignableTo(fetcher Fetcher, target Type) (bool, error) {
	targetT, ok := target.(ArrayType)
	if !ok {
		return false, nil
	}
	if targetT.Rank().IsAtomic() {
		return targetT.equalArray(fetcher, s)
	}
	return s.assignableToArray(fetcher, targetT)
}

// ConvertibleTo reports whether a value of the type can be converted to another
// (using static type casting).
func (s *arrayType) ConvertibleTo(fetcher Fetcher, target Type) (bool, error) {
	target = Underlying(target)
	targetT, ok := target.(ArrayType)
	if !ok {
		return false, nil
	}
	if targetT.Rank().IsAtomic() {
		return s.RankF.ConvertibleTo(fetcher, scalarRank)
	}
	dtypeOk, err := s.DTypeF.ConvertibleTo(fetcher, targetT.DataType())
	if !dtypeOk || err != nil {
		return dtypeOk, err
	}
	return s.RankF.ConvertibleTo(fetcher, targetT.Rank())
}

// Specialise a type to a given target.
func (s *arrayType) Specialise(spec Specialiser) (Type, error) {
	dtype, err := s.DTypeF.Specialise(spec)
	if err != nil {
		return InvalidType(), err
	}
	rank, err := s.RankF.Specialise(spec)
	if err != nil {
		return InvalidType(), err
	}
	return NewArrayType(s.Src, dtype, rank), nil
}

// UnifyWith recursively unifies a type parameters with types.
func (s *arrayType) UnifyWith(uni Unifier, typ Type) bool {
	other, ok := typ.(ArrayType)
	if !ok {
		return true
	}
	if !s.DTypeF.UnifyWith(uni, other.DataType()) {
		return false
	}
	return s.RankF.UnifyWith(uni, other.Rank())
}

// Value returns a value pointing to the receiver.
func (s *arrayType) Value(x Expr) Expr {
	return TypeExpr(x, s)
}

// Node returns the node in the AST tree.
func (s *arrayType) Node() ast.Node { return s.ArrayType() }

// ArrayType returns the source code defining the type.
func (s *arrayType) ArrayType() ast.Expr {
	return s.Src
}

// Zero returns a zero literal of this type
func (s *arrayType) Zero() Expr {
	cst := &NumberCastExpr{
		X:   zero,
		Typ: s.DataType(),
	}
	if s.RankF.IsAtomic() {
		return cst
	}
	return &CastExpr{
		Src: &ast.CallExpr{},
		Typ: s,
		X:   cst,
	}
}

func (s *arrayType) rankString(dtype string) string {
	rank := "[invalid]"
	if s.RankF != nil {
		rank = s.RankF.String()
	}
	return rank + dtype
}

// SourceString returns a reference to the type given a file context.
func (s *arrayType) SourceString(context *File) string {
	return s.rankString(s.DTypeF.SourceString(context))
}

// String representation of the tensor type.
func (s *arrayType) String() string {
	dtype := "dtype"
	if s.DTypeF != nil {
		dtype = s.DTypeF.String()
	}
	return s.rankString(dtype)
}

func (*TypeParam) node() {}

// Kind of the type.
func (s *TypeParam) Kind() irkind.Kind {
	return s.Field.Type().Kind()
}

func (s *TypeParam) equal(fetcher Fetcher, typ Type) (bool, error) {
	switch typT := typ.(type) {
	case *atomicType:
		return false, nil
	case *TypeParam:
		if s.Field == typT.Field {
			return true, nil
		}
	case *NamedType:
		return s.Field.Type().Equal(fetcher, typ)
	case ArrayType:
		if !typT.Rank().IsAtomic() {
			return false, nil
		}
		return s.Equal(fetcher, typT.DataType())
	}
	return false, nil
}

// Equal returns true if other is the same type.
func (s *TypeParam) Equal(fetcher Fetcher, typ Type) (bool, error) {
	return s.equal(fetcher, typ)
}

func (s *TypeParam) assignableFrom(fetcher Fetcher, other Type) (bool, error) {
	return other.AssignableTo(fetcher, s.Field.Type())
}

// AssignableTo reports whether a value of the type can be assigned to another.
func (s *TypeParam) AssignableTo(fetcher Fetcher, typ Type) (bool, error) {
	return s.Equal(fetcher, typ)
}

// ConvertibleTo reports whether a value of the type can be converted to another
// (using static type casting).
func (s *TypeParam) ConvertibleTo(fetcher Fetcher, typ Type) (bool, error) {
	return s.Field.Type().ConvertibleTo(fetcher, typ)
}

// convertibleFrom reports whether a value of some type can be converted to the receiver.
func (s *TypeParam) convertibleFrom(fetcher Fetcher, from Type) (bool, error) {
	return from.ConvertibleTo(fetcher, s.Field.Type())
}

// Node defining the type.
func (s *TypeParam) Node() ast.Node {
	return s.Field.Type().Node()
}

// Type of the type.
func (s *TypeParam) Type() Type {
	return s
}

// Same returns true if the other storage is this storage.
func (s *TypeParam) Same(o Storage) bool {
	return Storage(s) == o
}

// Value returns a value pointing to the receiver.
func (s *TypeParam) Value(x Expr) Expr {
	return TypeExpr(x, s)
}

// SourceString returns a reference to the type given a file context.
func (s *TypeParam) SourceString(context *File) string {
	return s.Field.Name.Name
}

// Specialise a type to a given target.
func (s *TypeParam) Specialise(spec Specialiser) (Type, error) {
	tp := spec.TypeOf(s.Field.Name.Name)
	if tp == nil {
		return s, nil
	}
	return tp, nil
}

// UnifyWith recursively unifies a type parameters with types.
func (s *TypeParam) UnifyWith(uni Unifier, typ Type) bool {
	return uni.DefineTParam(s, typ)
}

// String representation of the type.
func (s *TypeParam) String() string {
	return s.Field.Name.Name
}

// IsStatic return true if the type is static, that is if the instance of the type
// can be evaluated when the graph is being constructed.
func IsStatic(tp Type) bool {
	switch tp.Kind() {
	case irkind.Slice:
		return IsStatic(tp.(*SliceType).DType.Val())
	case irkind.IntIdx, irkind.IntLen:
		return true
	}
	return false
}

// ----------------------------------------------------------------------------
// Package level declarations.
type (
	// Package is a GX package.
	Package struct {
		FSet  *token.FileSet
		Files map[string]*File

		Name *ast.Ident
		Path string

		Decls *Declarations
	}

	// Declarations in a package.
	Declarations struct {
		Package *Package

		Consts []*ConstSpec
		Funcs  []PkgFunc
		Types  []*NamedType
		Vars   []*VarSpec
	}

	// File represents a GX source file.
	File struct {
		Package *Package
		Src     *ast.File

		Imports []*ImportDecl
	}

	// FuncImpl is a builtin opaque function implementation provided by
	// a backend or a host language.
	//
	// For example, the function num.Concat is not implemented in GX
	// and directly provided by the backend.
	FuncImpl interface {
		// Name of the builtin function.
		Name() string

		// BuildFuncType builds the type of a function given how it is called.
		BuildFuncType(fetcher Fetcher, call *FuncCallExpr) (*FuncType, error)

		// Implementation of the function, provided by the backend.
		Implementation() any
	}

	// Func is a callable GX function.
	Func interface {
		Node
		annotations.Annotated

		// File owning the function.
		File() *File

		// Type of the function.
		// If the function is generic, then the type will have been inferred by the compiler from the
		// types passed as args. Note that both FuncType() and Type() must return the same underlying
		// value, though FuncType returns the concrete type.
		FuncType() *FuncType

		// ShortString returns a human readable representation of the function.
		ShortString() string

		// Type of the function.
		Type() Type

		// String representation of the function.
		String() string
	}

	// PkgFunc is a function declared at the package level.
	PkgFunc interface {
		pkgFunc()
		StorageWithValue
		Func

		New() PkgFunc

		// Name of the function.
		Name() string

		// Doc returns associated documentation or nil.
		Doc() *ast.CommentGroup

		// Type of the function.
		// If the function is generic, then the type will have been inferred by the compiler from the
		// types passed as args. Note that both FuncType() and Type() must return the same underlying
		// value, though FuncType returns the concrete type.
		// FuncType returns nil if the function needs the call to build its type.
		FuncType() *FuncType
	}

	// Statically assert that the Func and Expr interfaces are compatible.
	_ interface {
		Func
		Expr
	}

	// FuncDecl is a GX function declared at the package level.
	FuncDecl struct {
		FFile *File
		Src   *ast.FuncDecl
		FType *FuncType
		Body  *BlockStmt
		Anns  annotations.Annotations
	}

	// FuncBuiltin is a function provided by a backend.
	// The implementation is opaque to GX.
	FuncBuiltin struct {
		FFile *File
		Src   *ast.FuncDecl
		FType *FuncType
		Impl  FuncImpl
		Anns  annotations.Annotations
	}

	// FuncKeyword is a function implementing a GX keyword.
	FuncKeyword struct {
		ID   *ast.Ident
		Impl FuncImpl
	}

	// FuncLit is a function literal.
	FuncLit struct {
		Src   *ast.FuncLit
		FType *FuncType
		FFile *File
		Body  *BlockStmt
		Anns  annotations.Annotations
	}

	// ImportDecl imports a package.
	ImportDecl struct {
		Src     *ast.ImportSpec
		Path    string
		Package *Package
	}

	// VarExpr is a name,expr static variable pair.
	VarExpr struct {
		Decl  *VarSpec
		VName *ast.Ident
	}

	// VarSpec declares a static variable and, optionally, a default value.
	VarSpec struct {
		FFile *File
		Src   *ast.ValueSpec
		TypeV Type
		Exprs []*VarExpr
	}

	// ConstExpr is a name,expr constant pair.
	ConstExpr struct {
		Decl *ConstSpec

		VName *ast.Ident
		Val   Expr
	}

	// ConstSpec declares a package constant.
	ConstSpec struct {
		FFile *File
		Src   *ast.ValueSpec
		Type  *TypeValExpr
		Exprs []*ConstExpr
	}
)

var (
	_ IR               = (*Package)(nil)
	_ StorageWithValue = (*ImportDecl)(nil)
	_ IR               = (*ConstSpec)(nil)
	_ StorageWithValue = (*ConstExpr)(nil)
	_ IR               = (*VarSpec)(nil)
	_ Storage          = (*VarExpr)(nil)
	_ PkgFunc          = (*FuncBuiltin)(nil)
	_ Func             = (*FuncKeyword)(nil)
	_ Storage          = (*FuncKeyword)(nil)
	_ PkgFunc          = (*FuncDecl)(nil)
	_ Func             = (*FuncLit)(nil)
	_ PkgFunc          = (*Macro)(nil)
	_ WithStore        = (*FuncLit)(nil)
	_ StorageWithValue = (*FuncLit)(nil)
)

func (*Package) node() {}

// File returns the AST tree given a file name.
// Return nil if the name does not match any file.
// If the name is empty, returns the first file (in alphabetical order).
func (pkg *Package) File(name string) *File {
	if len(pkg.Files) == 0 {
		return nil
	}
	if name == "" {
		names := slices.Sorted(maps.Keys(pkg.Files))
		name = names[0]
	}
	return pkg.Files[name]
}

// FullName returns the full name of the package, including its path.
func (pkg *Package) FullName() string {
	if pkg.Path == "" {
		return pkg.Name.Name
	}
	return pkg.Path + "/" + pkg.Name.Name
}

// TypeByName returns a type defined in the package given its name.
// Returns nil if the type could not be found.
func (decls *Declarations) TypeByName(name string) *NamedType {
	for _, tp := range decls.Types {
		if tp.Name() == name {
			return tp
		}
	}
	return nil
}

// IsExported returns true if a name is exported
// (the first letter is capitalized).
func IsExported(name string) bool {
	if len(name) == 0 {
		return false
	}
	first, _ := utf8.DecodeRuneInString(name)
	return unicode.IsUpper(first)
}

// ExportedFuncs returns the list of exported functions of a package.
func (pkg *Package) ExportedFuncs() iter.Seq[PkgFunc] {
	return func(yield func(PkgFunc) bool) {
		for _, fn := range pkg.Decls.Funcs {
			if !IsExported(fn.Name()) {
				continue
			}
			if !yield(fn) {
				break
			}
		}
	}
}

// FindFunc returns a function given its name or nil if not found.
func (pkg *Package) FindFunc(name string) PkgFunc {
	for _, fn := range pkg.Decls.Funcs {
		if fn.Name() == name {
			return fn
		}
	}
	return nil
}

// FindStatic returns a static variable given its name or nil if not found.
func (pkg *Package) FindStatic(name string) *VarExpr {
	for _, vars := range pkg.Decls.Vars {
		for _, vr := range vars.Exprs {
			if vr.VName.Name == name {
				return vr
			}
		}
	}
	return nil
}

// ExportedTypes returns the list of exported types of a package.
func (pkg *Package) ExportedTypes() []*NamedType {
	var types []*NamedType
	for _, tp := range pkg.Decls.Types {
		if !IsExported(tp.Name()) {
			continue
		}
		types = append(types, tp)
	}
	return types
}

// ExportedConsts returns the list of exported constants.
func (pkg *Package) ExportedConsts() []*ConstExpr {
	var exprs []*ConstExpr
	for _, csts := range pkg.Decls.Consts {
		for _, cst := range csts.Exprs {
			if !IsExported(cst.VName.Name) {
				continue
			}
			exprs = append(exprs, cst)
		}
	}
	return exprs
}

// ExportedStatics returns the list of exported static variables.
func (pkg *Package) ExportedStatics() []*VarExpr {
	var exprs []*VarExpr
	for _, vars := range pkg.Decls.Vars {
		for _, vr := range vars.Exprs {
			if !IsExported(vr.VName.Name) {
				continue
			}
			exprs = append(exprs, vr)
		}
	}
	return exprs
}

// String representation of the package.
func (pkg *Package) String() string {
	return pkg.FullName()
}

func (*File) node() {}

// Name of the file.
func (f *File) Name() string {
	return f.Package.FSet.Position(f.Src.Pos()).Filename
}

// FindImport returns an import declaration given a package full path.
// Returns nil if the path cannot be found.
func (f *File) FindImport(path string) *ImportDecl {
	for _, imp := range f.Imports {
		if imp.Path == path {
			return imp
		}
	}
	return nil
}

// FileSet returns the package fileset.
func (f *File) FileSet() *token.FileSet {
	if f == nil {
		return nil
	}
	return f.Package.FSet
}

func (*FuncDecl) node()         {}
func (*FuncDecl) staticValue()  {}
func (*FuncDecl) storage()      {}
func (*FuncDecl) storageValue() {}
func (*FuncDecl) pkgFunc()      {}

// Node returns the node in the AST tree.
func (s *FuncDecl) Node() ast.Node { return s.Src }

// Name of the function. Returns an empty string if the function is anonymous.
func (s *FuncDecl) Name() string {
	id := s.Src.Name
	if id == nil {
		return ""
	}
	return id.Name
}

// NameDef is the name definition of the function.
func (s *FuncDecl) NameDef() *ast.Ident {
	return s.Src.Name
}

// Value returns a reference to the function.
func (s *FuncDecl) Value(x Expr) Expr {
	return NewFuncValExpr(x, s)
}

// FullyQualifiedName returns a fully qualified function name, that is
// the full package path and the name of the function.
func (s *FuncDecl) FullyQualifiedName() string {
	return s.FFile.Package.FullName() + "." + s.Name()
}

// Doc returns associated documentation or nil.
func (s *FuncDecl) Doc() *ast.CommentGroup {
	return s.Src.Doc
}

// Type returns the type of the function.
func (s *FuncDecl) Type() Type {
	return s.FType
}

// FuncType returns the concrete type of the function.
func (s *FuncDecl) FuncType() *FuncType {
	return s.FType
}

// Same returns true if the other storage is this storage.
func (s *FuncDecl) Same(o Storage) bool {
	return Storage(s) == o
}

// File declaring the function.
func (s *FuncDecl) File() *File {
	return s.FFile
}

// Annotations returns the annotations attached to the function.
func (s *FuncDecl) Annotations() *annotations.Annotations {
	return &s.Anns
}

// New returns a new function given a source, a file, and a type.
func (s *FuncDecl) New() PkgFunc {
	n := *s
	return &n
}

// ShortString returns the name of the function.
func (s *FuncDecl) ShortString() string {
	name := s.Name()
	if name != "" {
		return name
	}
	return s.FuncType().String()
}

func (*FuncBuiltin) node()         {}
func (*FuncBuiltin) staticValue()  {}
func (*FuncBuiltin) storage()      {}
func (*FuncBuiltin) storageValue() {}
func (*FuncBuiltin) pkgFunc()      {}

// Node returns the node in the AST tree.
func (s *FuncBuiltin) Node() ast.Node { return s.Src }

// Name of the function. Returns an empty string if the function is anonymous.
func (s *FuncBuiltin) Name() string {
	return s.NameDef().Name
}

// NameDef is the name definition of the function.
func (s *FuncBuiltin) NameDef() *ast.Ident {
	return s.Src.Name
}

// ShortString returns the name of the function.
func (s *FuncBuiltin) ShortString() string {
	return s.FFile.Package.Name.Name + "." + s.Name()
}

// Value returns a reference to the function.
func (s *FuncBuiltin) Value(x Expr) Expr {
	return NewFuncValExpr(x, s)
}

// Doc returns associated documentation or nil.
func (s *FuncBuiltin) Doc() *ast.CommentGroup {
	return nil
}

// Type returns the type of the function.
func (s *FuncBuiltin) Type() Type {
	return s.FType
}

// FuncType returns the concrete type of the function.
func (s *FuncBuiltin) FuncType() *FuncType {
	return s.FType
}

// Same returns true if the other storage is this storage.
func (s *FuncBuiltin) Same(o Storage) bool {
	return Storage(s) == o
}

// File declaring the function.
func (s *FuncBuiltin) File() *File {
	return s.FFile
}

// Annotations returns the annotations attached to the function.
func (s *FuncBuiltin) Annotations() *annotations.Annotations {
	return &s.Anns
}

// New returns a new function given a source, a file, and a type.
func (s *FuncBuiltin) New() PkgFunc {
	n := *s
	return &n
}

func (*FuncKeyword) node()    {}
func (*FuncKeyword) storage() {}

// Node returns the node in the AST tree.
func (s *FuncKeyword) Node() ast.Node { return s.ID }

// NameDef is the name definition of the function.
func (s *FuncKeyword) NameDef() *ast.Ident {
	return s.ID
}

// Name of the function. Returns an empty string if the function is anonymous.
func (s *FuncKeyword) Name() string {
	return s.ID.Name
}

// Doc returns associated documentation or nil.
func (s *FuncKeyword) Doc() *ast.CommentGroup {
	return nil
}

// Type returns the type of the function.
func (s *FuncKeyword) Type() Type {
	return keywordType()
}

// FuncType returns the concrete type of the function.
func (s *FuncKeyword) FuncType() *FuncType {
	return nil
}

// File declaring the function.
func (s *FuncKeyword) File() *File {
	return nil
}

// Annotations returns the annotations attached to the function.
func (s *FuncKeyword) Annotations() *annotations.Annotations {
	return nil
}

// Same returns true if the other storage is this storage.
func (s *FuncKeyword) Same(o Storage) bool {
	return Storage(s) == o
}

// ShortString returns a short string representation of the keyword.
func (s *FuncKeyword) ShortString() string {
	return s.String()
}

// String return the name of the keyword.
func (s *FuncKeyword) String() string {
	return s.ID.Name
}

func (*FuncLit) node()         {}
func (*FuncLit) staticValue()  {}
func (*FuncLit) storage()      {}
func (*FuncLit) storageValue() {}

// Name of the function. Returns an empty string since literals are always anonymous.
func (s *FuncLit) Name() string {
	return ""
}

// Store returns the function literal as a store.
func (s *FuncLit) Store() Storage {
	return s
}

// NameDef returns nil because function literals are anonymous.
func (s *FuncLit) NameDef() *ast.Ident {
	return nil
}

// Value returns the function literal as an assignable expression.
func (s *FuncLit) Value(x Expr) Expr {
	return NewFuncValExpr(x, s)
}

// ShortString returns the name of the function.
func (s *FuncLit) ShortString() string {
	return s.FType.String()
}

// Doc returns associated documentation or nil.
func (s *FuncLit) Doc() *ast.CommentGroup {
	return nil
}

// File declaring the function literal.
func (s *FuncLit) File() *File {
	return s.FFile
}

// Type returns the type of the function.
func (s *FuncLit) Type() Type {
	return s.FType
}

// FuncType returns the concrete type of the function.
func (s *FuncLit) FuncType() *FuncType {
	return s.FType
}

// Same returns true if the other storage is this storage.
func (s *FuncLit) Same(o Storage) bool {
	return Storage(s) == o
}

// Node returns the node in the AST tree.
func (s *FuncLit) Node() ast.Node { return s.Src }

// Expr returns the expression AST.
func (s *FuncLit) Expr() ast.Expr { return s.Src }

// Annotations returns the annotations attached to the function.
func (s *FuncLit) Annotations() *annotations.Annotations {
	return &s.Anns
}

func (*ImportDecl) node()         {}
func (*ImportDecl) staticValue()  {}
func (*ImportDecl) storage()      {}
func (*ImportDecl) storageValue() {}

// Node returns the node in the AST tree.
func (s *ImportDecl) Node() ast.Node {
	return s.Src
}

// Name used to reference the package.
func (s *ImportDecl) Name() string {
	if s.Src.Name == nil {
		return filepath.Base(s.Path)
	}
	return s.Src.Name.Name
}

// NameDef is the name definition of the function.
func (s *ImportDecl) NameDef() *ast.Ident {
	if s.Src.Name != nil {
		return s.Src.Name
	}
	return &ast.Ident{
		NamePos: s.Src.Path.ValuePos,
		Name:    s.Name(),
	}
}

// Type returns the IR package type.
func (*ImportDecl) Type() Type {
	return PackageType()
}

// Value returns a reference to the function.
func (s *ImportDecl) Value(x Expr) Expr {
	return &PackageRef{
		X:    x.(*ValueRef),
		Decl: s,
	}
}

// Same returns true if the other storage is this storage.
func (s *ImportDecl) Same(o Storage) bool {
	return Storage(s) == o
}

func (s *ImportDecl) String() string {
	return s.Path
}

func (*ConstSpec) node() {}

func (*ConstExpr) node()         {}
func (*ConstExpr) storage()      {}
func (*ConstExpr) storageValue() {}

// Node returns the node in the AST tree.
func (cst *ConstExpr) Node() ast.Node {
	return cst.VName
}

// Same returns true if the other storage is this storage.
func (cst *ConstExpr) Same(o Storage) bool {
	return Storage(cst) == o
}

// NameDef returns the name defining the storage.
func (cst *ConstExpr) NameDef() *ast.Ident {
	return cst.VName
}

// Type returns the type of an expression.
func (cst *ConstExpr) Type() Type {
	if cst.Val == nil {
		return UnknownType()
	}
	return cst.Val.Type()
}

// Value assigned to the constant.
func (cst *ConstExpr) Value(Expr) Expr {
	return cst.Val
}

func (*VarSpec) node() {}

func (*VarExpr) node()    {}
func (*VarExpr) storage() {}

// Node returns the node in the AST tree.
func (vr *VarExpr) Node() ast.Node {
	return vr.VName
}

// Same returns true if the other storage is this storage.
func (vr *VarExpr) Same(o Storage) bool {
	return Storage(vr) == o
}

// Type returns the type of an expression.
func (vr *VarExpr) Type() Type {
	return vr.Decl.TypeV
}

// NameDef returns the identifier for the static variable.
func (vr *VarExpr) NameDef() *ast.Ident {
	return vr.VName
}

// ----------------------------------------------------------------------------
// Fields in function arguments, results, and structures.
type (
	// FieldList is a list of fields, enclosed by parentheses,
	// curly braces, or square brackets.
	FieldList struct {
		Src  *ast.FieldList
		List []*FieldGroup
	}

	// FieldGroup is a list of field matched to a type.
	FieldGroup struct {
		Src    *ast.Field
		Fields []*Field
		Type   *TypeValExpr
	}

	// Field is a field belonging to a field group.
	Field struct {
		Group *FieldGroup

		Name *ast.Ident
	}
)

var (
	_ Node = (*FieldList)(nil)
	_ Node = (*FieldGroup)(nil)
	_ Expr = (*Field)(nil)
)

func (*FieldList) node() {}

// Node returns the source defining the field.
func (s *FieldList) Node() ast.Node {
	return s.Src
}

// Fields returns a list of all the fields in the structure
// in a consistent order which can be used as a reference.
func (s *FieldList) Fields() []*Field {
	if s == nil {
		return nil
	}
	var fields []*Field
	for _, grp := range s.List {
		if len(grp.Fields) == 0 {
			fields = append(fields, &Field{
				Group: grp,
			})
			continue
		}
		for _, field := range grp.Fields {
			fields = append(fields, field)
		}
	}
	return fields
}

// FindField returns the field matching the given name or nil if not found.
func (s *FieldList) FindField(name string) *Field {
	for _, grp := range s.List {
		for _, field := range grp.Fields {
			if field.Name.Name == name {
				return field
			}
		}
	}
	return nil
}

// Len returns the total number of fields in the list.
func (s *FieldList) Len() int {
	if s == nil {
		return 0
	}
	r := 0
	for _, grp := range s.List {
		r += grp.NumFields()
	}
	return r
}

// Type returns the fields as a type.
func (s *FieldList) Type() Type {
	switch s.Len() {
	case 0:
		return VoidType()
	case 1:
		return s.Fields()[0].Type()
	}
	return s.TupleType()
}

// TupleType returns the fields as a tuple, regardless of their number.
func (s *FieldList) TupleType() *TupleType {
	fields := s.Fields()
	types := make([]Type, len(fields))
	for i, field := range fields {
		types[i] = field.Type()
	}
	return &TupleType{
		Types: types,
	}
}

func (*FieldGroup) node() {}

// Node returns the node in the AST tree.
func (s *FieldGroup) Node() ast.Node {
	if len(s.Src.Names) > 0 {
		return s.Src.Names[0]
	}
	return s.Src.Type
}

// NumFields returns the number of field in the group.
func (s *FieldGroup) NumFields() int {
	if len(s.Fields) == 0 {
		return 1
	}
	return len(s.Fields)
}

func (s *Field) node() {}

// Expr returns
func (s *Field) Expr() ast.Expr {
	return s.Name
}

// Node returns the source defining the field.
func (s *Field) Node() ast.Node {
	if s.Name == nil {
		return s.Group.Node()
	}
	return s.Name
}

// Storage returns a storage pointing to the field.
func (s *Field) Storage() *FieldStorage {
	return &FieldStorage{Field: s}
}

// Type returns the type of the field.
func (s *Field) Type() Type {
	if s.Group.Type == nil {
		return nil
	}
	return s.Group.Type.Val()
}

// String returns a string representation of the field.
func (s *Field) String() string {
	if s.Name == nil {
		return s.Group.Type.String()
	}
	return fmt.Sprintf("%s %s", s.Name.Name, s.Group.Type.String())
}

// ----------------------------------------------------------------------------
// Expressions.
type (
	// Expr is an expression that returns a (typed) result.
	Expr interface {
		Node
		Expr() ast.Expr
		Type() Type
		String() string
	}

	// StringLiteral is a string defined by a literal.
	StringLiteral struct {
		Src *ast.BasicLit
	}

	// Number is a constant defined in the source code to which no concrete type has been assigned.
	Number interface {
		Expr
		numberExpr()
	}

	// NumberFloat is a float number for which the type has not been inferred yet.
	NumberFloat struct {
		Src *ast.BasicLit
		Val *big.Float
	}

	// NumberInt is an integer number for which the type has not been inferred yet.
	NumberInt struct {
		Src *ast.BasicLit
		Val *big.Int
	}

	// NumberCastExpr casts a number to a given type.
	NumberCastExpr struct {
		X   Expr
		Typ Type
	}

	// AtomicValue is implemented by all atomic values.
	AtomicValue interface {
		Expr
		atomicValue()
	}

	// AtomicValueT is a builtin constant.
	AtomicValueT[T dtype.GoDataType] struct {
		Src ast.Expr
		Val T
		Typ Type
	}

	// ArrayLitExpr is an array literal.
	ArrayLitExpr struct {
		Src  *ast.CompositeLit
		Typ  ArrayType
		Elts []Expr
	}

	// SliceLitExpr is a slice literal.
	SliceLitExpr struct {
		Src  ast.Expr
		Typ  Type
		Elts []Expr
	}

	// FieldLit assigns a value to a field in a structure literal.
	FieldLit struct {
		*FieldStorage
		X Expr
	}

	// StructLitExpr is a structure literal.
	StructLitExpr struct {
		Src  *ast.CompositeLit
		Elts []*FieldLit
		Typ  Type
	}

	// UnaryExpr is an operator with a single argument.
	UnaryExpr struct {
		Src *ast.UnaryExpr
		X   Expr
	}

	// ParenExpr is a parenthesized expression.
	ParenExpr struct {
		Src *ast.ParenExpr
		X   Expr
	}

	// BinaryExpr is an operator with two arguments.
	BinaryExpr struct {
		Src  *ast.BinaryExpr
		X, Y Expr
		Typ  Type
	}

	// TypeValExpr is an expression pointing to a type.
	TypeValExpr struct {
		x   Expr
		val Type
	}

	// FuncValExpr is an expression pointing to a function.
	FuncValExpr struct {
		x Expr
		f Func
		// T is the function type if the function has been specialised by the call
		// (in the context of generic functions).
		t *FuncType
	}

	// Callee function being called from a call expression.
	Callee interface {
		Node
		Func() Func
		FuncType() *FuncType
		ShortString() string
		SourceString() string
	}

	// FuncCallExpr is an expression calling a function.
	FuncCallExpr struct {
		Src    *ast.CallExpr
		Args   []Expr
		Callee Callee
	}

	// CallResultExpr represents the ith result of a function call as an expression.
	CallResultExpr struct {
		Index int
		Call  *FuncCallExpr
	}

	// TypeCastExpr is an abstract type conversion.
	TypeCastExpr interface {
		Expr
		Orig() Expr
	}

	// CastExpr casts a type to another.
	CastExpr struct {
		Src ast.Expr
		Typ Type

		X Expr
	}

	// TypeAssertExpr casts a type to another while disabling compiler checks on array dimensions.
	// Dimension checks are postponed from compilation time to run time.
	TypeAssertExpr struct {
		Src *ast.TypeAssertExpr
		Typ Type

		X Expr
	}

	// ValueRef is a reference to a value.
	ValueRef struct {
		Src  *ast.Ident
		Stor Storage
	}

	// PackageRef is a reference to a package.
	PackageRef struct {
		X    *ValueRef
		Decl *ImportDecl
	}

	// SelectorExpr selects a field on a structure.
	SelectorExpr struct {
		Src  *ast.SelectorExpr
		X    Expr
		Stor Storage
	}

	// IndexExpr selects an index on a indexable type.
	IndexExpr struct {
		Src   *ast.IndexExpr
		X     Expr
		Index Expr
		Typ   Type
	}

	// IndexListExpr selects an index on a indexable type.
	IndexListExpr struct {
		Src     *ast.IndexListExpr
		X       Expr
		Indices Expr
		Typ     Type
	}

	// EinsumExpr represents an einsum expression.
	EinsumExpr struct {
		Src        ast.Expr
		X, Y       Expr
		BatchAxes  [2][]int
		ReduceAxes [2][]int
		Typ        Type
	}

	// RuntimeValue is a value that only exists at runtime, that is
	// when GX code is being interpreted.
	RuntimeValue interface {
		Type() Type
	}

	// RuntimeValueExpr is an expression representing a runtime value.
	// Such expression will never be present in a compiled package.
	// This type is used by the interpreter to represent a GX value
	// as an expression.
	RuntimeValueExpr interface {
		Expr
		Value(Expr) RuntimeValue
	}

	// RuntimeValueExprT is a runtime expression specialised for a value type.
	// The Src field can be nil if the value was not computed from an expression
	// in the source code (e.g. a value given as a static value programmatically).
	RuntimeValueExprT[T RuntimeValue] struct {
		Src ast.Expr
		Typ Type
		Val T
	}

	// Tuple is a group of expression.
	Tuple struct {
		Exprs []Expr
	}
)

var (
	_ Number           = (*NumberFloat)(nil)
	_ Number           = (*NumberInt)(nil)
	_ Expr             = (*NumberCastExpr)(nil)
	_ AtomicValue      = (*AtomicValueT[int32])(nil)
	_ Expr             = (*StringLiteral)(nil)
	_ Expr             = (*ArrayLitExpr)(nil)
	_ Expr             = (*SliceLitExpr)(nil)
	_ Expr             = (*StructLitExpr)(nil)
	_ StorageWithValue = (*FieldLit)(nil)
	_ Expr             = (*UnaryExpr)(nil)
	_ Expr             = (*ParenExpr)(nil)
	_ WithStore        = (*ParenExpr)(nil)
	_ Expr             = (*BinaryExpr)(nil)
	_ CallExpr         = (*FuncCallExpr)(nil)
	_ Expr             = (*CallResultExpr)(nil)
	_ TypeCastExpr     = (*CastExpr)(nil)
	_ TypeCastExpr     = (*TypeAssertExpr)(nil)
	_ Expr             = (*ValueRef)(nil)
	_ WithStore        = (*ValueRef)(nil)
	_ Expr             = (*PackageRef)(nil)
	_ Expr             = (*SelectorExpr)(nil)
	_ WithStore        = (*SelectorExpr)(nil)
	_ Expr             = (*IndexExpr)(nil)
	_ Expr             = (*IndexListExpr)(nil)
	_ Expr             = (*EinsumExpr)(nil)
	_ RuntimeValueExpr = (*RuntimeValueExprT[RuntimeValue])(nil)
	_ Node             = (*Tuple)(nil)

	_ Expr      = (*FuncValExpr)(nil)
	_ Callee    = (*FuncValExpr)(nil)
	_ Expr      = (*TypeValExpr)(nil) // Use Expr to store a type in a field (in function type params).
	_ WithStore = (*TypeValExpr)(nil)
)

func (s *NumberFloat) node()       {}
func (s *NumberFloat) numberExpr() {}

// Expr returns the expression AST.
func (s *NumberFloat) Expr() ast.Expr { return s.Src }

// Zero returns a zero float number.
func (s *NumberFloat) Zero() Expr {
	return zeroFloat
}

// Node returns the node in the AST tree.
func (s *NumberFloat) Node() ast.Node { return s.Src }

// Type returns the type returned by the function call.
func (s *NumberFloat) Type() Type { return NumberFloatType() }

// DefaultType returns the default type of the number.
func (s NumberFloat) DefaultType() Type { return TypeFromKind(irkind.Float64) }

func (s *NumberInt) node()       {}
func (s *NumberInt) numberExpr() {}

// Expr returns the expression AST.
func (s *NumberInt) Expr() ast.Expr { return s.Src }

// Zero returns a zero float number.
func (s *NumberInt) Zero() Expr {
	return zero
}

// Node returns the node in the AST tree.
func (s *NumberInt) Node() ast.Node { return s.Src }

// Type returns the type returned by the function call.
func (s *NumberInt) Type() Type { return NumberIntType() }

// DefaultType returns the default type of the number.
func (s NumberInt) DefaultType() Type { return TypeFromKind(irkind.Int64) }

func (s *NumberCastExpr) node()        {}
func (s *NumberCastExpr) staticValue() {}

// Node returns the node in the AST tree.
func (s *NumberCastExpr) Node() ast.Node { return s.X.Node() }

// Expr returns the expression AST.
func (s *NumberCastExpr) Expr() ast.Expr { return s.X.Expr() }

// Type returns the type returned by the function call.
func (s *NumberCastExpr) Type() Type { return s.Typ }

// String representation of the number.
func (s *NumberCastExpr) String() string {
	return s.X.String()
}

func (s *StringLiteral) node()        {}
func (s *StringLiteral) staticValue() {}

// Expr returns the AST expression.
func (s *StringLiteral) Expr() ast.Expr { return s.Src }

// Node returns the node in the AST tree.
func (s *StringLiteral) Node() ast.Node { return s.Src }

// Type returns the type returned by the function call.
func (s *StringLiteral) Type() Type { return StringType() }

// String representation of the number.
func (s *StringLiteral) String() string { return s.Src.Value }

func (s *AtomicValueT[T]) node()        {}
func (s *AtomicValueT[T]) atomicValue() {}

// Expr returns the AST expression.
func (s *AtomicValueT[T]) Expr() ast.Expr { return s.Src }

// Node returns the node in the AST tree.
func (s *AtomicValueT[T]) Node() ast.Node { return s.Src }

// Type returns the type returned by the function call.
func (s *AtomicValueT[T]) Type() Type { return s.Typ }

// String representation.
func (s *AtomicValueT[T]) String() string { return fmt.Sprint(s.Val) }

func (s *ArrayLitExpr) node() {}

// Node returns the node in the AST tree.
func (s *ArrayLitExpr) Node() ast.Node {
	return s.Src
}

// Type returns the type returned by the function call.
func (s *ArrayLitExpr) Type() Type { return s.Typ }

// Expr returns the AST expression.
func (s *ArrayLitExpr) Expr() ast.Expr { return s.Src }

// Values returns the expressions defining the values of the array.
func (s *ArrayLitExpr) Values() []Expr { return s.Elts }

// NewFromValues returns a new literal of the same type from a slice of values.
func (s *ArrayLitExpr) NewFromValues(elts []Expr) *ArrayLitExpr {
	return &ArrayLitExpr{
		Typ:  s.Typ,
		Elts: elts,
	}
}

func (s *StructLitExpr) node() {}

// Node returns the node in the AST tree.
func (s *StructLitExpr) Node() ast.Node { return s.Src }

// Type returns the type returned by the function call.
func (s *StructLitExpr) Type() Type { return s.Typ }

// Expr returns the AST expression.
func (s *StructLitExpr) Expr() ast.Expr { return s.Src }

// String representation.
func (s *StructLitExpr) String() string { return s.Type().String() }

func (s *SliceLitExpr) node() {}

// Node returns the node in the AST tree.
func (s *SliceLitExpr) Node() ast.Node { return s.Src }

// Type returns the type returned by the function call.
func (s *SliceLitExpr) Type() Type { return s.Typ }

// Expr returns the AST expression.
func (s *SliceLitExpr) Expr() ast.Expr { return s.Src }

func (s *UnaryExpr) node() {}

// Node returns the node in the AST tree.
func (s *UnaryExpr) Node() ast.Node { return s.Src }

// Type returns the type returned by the expression.
func (s *UnaryExpr) Type() Type { return s.X.Type() }

// Expr returns the expression in the source code.
func (s *UnaryExpr) Expr() ast.Expr { return s.Src }

// String representation.
func (s *UnaryExpr) String() string { return s.Src.Op.String() + s.X.String() }

func (s *ParenExpr) node() {}

// Node returns the node in the AST tree.
func (s *ParenExpr) Node() ast.Node { return s.Src }

// Type returns the type returned by the expression.
func (s *ParenExpr) Type() Type { return s.X.Type() }

// Expr returns the expression in the source code.
func (s *ParenExpr) Expr() ast.Expr { return s.Src }

// Store returns the storage referenced by this expression.
func (s *ParenExpr) Store() Storage {
	withStore, ok := s.X.(WithStore)
	if !ok {
		return nil
	}
	return withStore.Store()
}

// String representation.
func (s *ParenExpr) String() string { return "(" + s.X.String() + ")" }

func (s *BinaryExpr) node() {}

// Node returns the node in the AST tree.
func (s *BinaryExpr) Node() ast.Node { return s.Src }

// Type returned by the expression.
func (s *BinaryExpr) Type() Type { return s.Typ }

// Expr returns the expression in the source code.
func (s *BinaryExpr) Expr() ast.Expr { return s.Src }

// TypeExpr builds an expression given a type.
func TypeExpr(x Expr, t Type) *TypeValExpr {
	if x == nil {
		x = &ValueRef{
			Stor: t,
			Src:  &ast.Ident{Name: t.String()},
		}
	}
	return &TypeValExpr{x: x, val: t}
}

func (s *TypeValExpr) node()     {}
func (s *TypeValExpr) typeExpr() {}

// Node returns the node in the AST tree.
func (s *TypeValExpr) Node() ast.Node { return s.Expr() }

// Expr returns the AST expression.
func (s *TypeValExpr) Expr() ast.Expr { return s.x.Expr() }

// Type of the function being called.
func (s *TypeValExpr) Type() Type {
	// The value of the reference is stored in the Typ field.
	// The type of that value is a meta-type.
	return MetaType()
}

// X returns the IR expression representing the type.
func (s *TypeValExpr) X() Expr {
	return s.x
}

// Val returns the type referenced by the expression.
func (s *TypeValExpr) Val() Type {
	return s.val
}

// Store storing the type.
func (s *TypeValExpr) Store() Storage { return s.val }

// String representation.
func (s *TypeValExpr) String() string { return s.X().String() }

// NewFuncValExpr returns a new expression referencing a function.
func NewFuncValExpr(x Expr, fn Func) *FuncValExpr {
	return &FuncValExpr{x: x, f: fn, t: fn.FuncType()}
}

func (s *FuncValExpr) node() {}

// X returns the expression referencing the function.
func (s *FuncValExpr) X() Expr {
	return s.x
}

// Node returns the node in the AST tree.
func (s *FuncValExpr) Node() ast.Node { return s.Expr() }

// Expr returns the AST expression.
func (s *FuncValExpr) Expr() ast.Expr { return s.x.Expr() }

// Func returns the function being called.
func (s *FuncValExpr) Func() Func {
	return s.f
}

// FuncType returns type of the function being.
// If the function is generic, the type is specialised.
func (s *FuncValExpr) FuncType() *FuncType {
	return s.t
}

// NewFType returns the expression for a new function type (e.g. after the function has been specialised).
func (s *FuncValExpr) NewFType(t *FuncType) *FuncValExpr {
	return &FuncValExpr{x: s.x, f: s.f, t: t}
}

// Type of the function being called.
func (s *FuncValExpr) Type() Type {
	return s.t
}

// ShortString representing the function being called.
func (s *FuncValExpr) ShortString() string {
	return s.x.String()
}

// SourceString returns GX code representing the call to the function.
func (s *FuncValExpr) SourceString() string {
	return s.String()
}

// String representation.
func (s *FuncValExpr) String() string {
	callee := s.x.String()
	spec := ""
	fType := s.t
	numTypeParamValues := len(fType.TypeParamsValues)
	if numTypeParamValues > 0 {
		types := make([]string, numTypeParamValues)
		for i, val := range fType.TypeParamsValues {
			types[i] = val.Typ.String()
		}
		spec = "[" + strings.Join(types, ",") + "]"
	}
	return callee + spec
}

func (s *FuncCallExpr) node() {}

// Node returns the node in the AST tree.
func (s *FuncCallExpr) Node() ast.Node { return s.Call() }

// Call returns the source of the call.
func (s *FuncCallExpr) Call() *ast.CallExpr { return s.Src }

// Type returns the type returned by the function call.
// Use CallExpr.Func.Type to get the type of the function being called.
func (s *FuncCallExpr) Type() Type {
	if s.Callee == nil {
		return InvalidType()
	}
	return s.Callee.FuncType().Results.Type()
}

// FuncCall returns the function call from the call.
func (s *FuncCallExpr) FuncCall() *FuncCallExpr {
	return s
}

// Expr returns the expression in the source code.
func (s *FuncCallExpr) Expr() ast.Expr { return s.Src }

// ExprFromResult returns an expression pointing to the ith result of a function call.
func (s *FuncCallExpr) ExprFromResult(i int) *CallResultExpr {
	return &CallResultExpr{
		Index: i,
		Call:  s,
	}
}

func (s *CallResultExpr) node() {}

// Node returns the node in the AST tree.
func (s *CallResultExpr) Node() ast.Node { return s.Call.Node() }

// Type returns the type returned by the function call.
// Use CallExpr.Func.Type to get the type of the function being called.
func (s *CallResultExpr) Type() Type {
	return s.Call.Callee.FuncType().Results.Fields()[s.Index].Type()
}

// Expr returns the expression in the source code.
func (s *CallResultExpr) Expr() ast.Expr { return s.Call.Expr() }

// String representation.
func (s *CallResultExpr) String() string { return fmt.Sprintf("%T", s) }

func (s *CastExpr) node() {}

// Node returns the node in the AST tree.
func (s *CastExpr) Node() ast.Node { return s.Src }

// Type returns the target type of the cast.
func (s *CastExpr) Type() Type { return s.Typ }

// Expr returns the expression in the source code.
func (s *CastExpr) Expr() ast.Expr { return s.Src }

// Orig returns the expression being casted.
func (s *CastExpr) Orig() Expr { return s.X }

func (s *TypeAssertExpr) node() {}

// Node returns the node in the AST tree.
func (s *TypeAssertExpr) Node() ast.Node { return s.Src }

// Type returns the target type of the cast.
func (s *TypeAssertExpr) Type() Type { return s.Typ }

// Expr returns the expression in the source code.
func (s *TypeAssertExpr) Expr() ast.Expr { return s.Src }

// Orig returns the expression being casted.
func (s *TypeAssertExpr) Orig() Expr { return s.X }

func (s *ValueRef) node() {}

// Node returns the node in the AST tree.
func (s *ValueRef) Node() ast.Node { return s.Src }

// Type returns the type returned by the function call.
// Use CallExpr.Func.Type to get the type of the function being called.
func (s *ValueRef) Type() Type { return s.Stor.Type() }

// Expr returns the expression in the source code.
func (s *ValueRef) Expr() ast.Expr { return s.Src }

// Store returns the storage referenced by this expression.
func (s *ValueRef) Store() Storage { return s.Stor }

// String representation.
func (s *ValueRef) String() string { return s.Src.Name }

func (s *PackageRef) node() {}

// Node returns the node in the AST tree.
func (s *PackageRef) Node() ast.Node { return s.Expr() }

// Expr returns the AST expression.
func (s *PackageRef) Expr() ast.Expr { return s.X.Expr() }

// Type returns the package type.
func (s *PackageRef) Type() Type { return PackageType() }

// String representation.
func (s *PackageRef) String() string { return s.X.String() }

func (*FieldLit) node()         {}
func (*FieldLit) storageValue() {}

// Value returns the value stored in the field.
func (lit *FieldLit) Value(Expr) Expr {
	return lit.X
}

// Same returns true if the other storage is this storage.
func (lit *FieldLit) Same(o Storage) bool {
	return Storage(lit) == o
}

func (s *SelectorExpr) node() {}

// Node returns the node in the AST tree.
func (s *SelectorExpr) Node() ast.Node { return s.Src }

// Type returns the type returned after the selection.
func (s *SelectorExpr) Type() Type { return s.Stor.Type() }

// Expr returns the AST expression.
func (s *SelectorExpr) Expr() ast.Expr { return s.Src }

// Select finds a method on a named type or a field in a structure.
func (s *SelectorExpr) Select(typ Type) (method PkgFunc, field *Field) {
	named, ok := typ.(*NamedType)
	if ok {
		method = named.MethodByName(s.Src.Sel.Name)
	}
	if method != nil {
		return
	}
	under := Underlying(typ)
	strct, ok := under.(*StructType)
	if ok {
		field = strct.Fields.FindField(s.Src.Sel.Name)
		return
	}
	return
}

// Store returns the storage referenced by this expression.
func (s *SelectorExpr) Store() Storage {
	return s.Stor
}

func (s *IndexExpr) node() {}

// Node returns the node in the AST tree.
func (s *IndexExpr) Node() ast.Node { return s.Src }

// Type returns the type of the indexed element.
func (s *IndexExpr) Type() Type { return s.Typ }

// Expr returns the AST expression.
func (s *IndexExpr) Expr() ast.Expr { return s.Src }

func (s *IndexListExpr) node() {}

// Node returns the node in the AST tree.
func (s *IndexListExpr) Node() ast.Node { return s.Src }

// Type returns the type of the indexed element.
func (s *IndexListExpr) Type() Type { return s.Typ }

// Expr returns the AST expression.
func (s *IndexListExpr) Expr() ast.Expr { return s.Src }

// String representation.
func (s *IndexListExpr) String() string { return fmt.Sprintf("%T", s) }

func (s *EinsumExpr) node() {}

// Node returns the node in the AST tree.
func (s *EinsumExpr) Node() ast.Node { return s.Src }

// Type returns the type returned by the einsum.
func (s *EinsumExpr) Type() Type { return s.Typ }

// Expr returns the expression in the source code.
func (s *EinsumExpr) Expr() ast.Expr { return s.Src }

// String representation.
func (s *EinsumExpr) String() string { return fmt.Sprintf("%T", s) }

func (s *RuntimeValueExprT[T]) node() {}

// Node returns the node in the AST tree.
func (s *RuntimeValueExprT[T]) Node() ast.Node { return s.Src }

// Expr returns the expression in the source code.
func (s *RuntimeValueExprT[T]) Expr() ast.Expr { return s.Src }

// Type returns the type referenced by the expression.
func (s *RuntimeValueExprT[T]) Type() Type { return s.Typ }

// Value returns the runtime value stored in the expression.
func (s *RuntimeValueExprT[T]) Value(Expr) RuntimeValue { return s.Val }

// String representation of the type.
func (s *RuntimeValueExprT[T]) String() string {
	return fmt.Sprintf("runtime(%T)", s.Val)
}

func (s *Tuple) node() {}

// Type returns the tuple type.
func (s *Tuple) Type() Type {
	typs := make([]Type, len(s.Exprs))
	for i, expr := range s.Exprs {
		typs[i] = expr.Type()
	}
	var src ast.Expr
	if len(s.Exprs) > 0 {
		src = s.Exprs[0].Node().(ast.Expr)
	}
	return &TupleType{
		BaseType: BaseType[ast.Expr]{Src: src},
		Types:    typs,
	}
}

// Node returns the node in the AST tree.
func (s *Tuple) Node() ast.Node {
	if len(s.Exprs) == 0 {
		return nil
	}
	return s.Exprs[0].Node()
}

// ----------------------------------------------------------------------------
// Storage.

type (
	// AnonymousStorage stores values that have not been labelled
	// (for example, the axis length in [_]float32).
	AnonymousStorage struct {
		Src *ast.Ident
		Typ Type
	}

	// LocalVarStorage is a local variable to which values can be assigned to.
	LocalVarStorage struct {
		Src *ast.Ident
		Typ Type
	}

	// StructFieldStorage is a field in a structure.
	StructFieldStorage struct {
		Sel *SelectorExpr
	}

	// FieldStorage is a field to which values can be assigned to.
	FieldStorage struct {
		Field *Field
	}
)

var (
	_ Storage = (*AnonymousStorage)(nil)
	_ Storage = (*LocalVarStorage)(nil)
	_ Storage = (*StructFieldStorage)(nil)
	_ Storage = (*FieldStorage)(nil)
)

func (*AnonymousStorage) node()    {}
func (*AnonymousStorage) storage() {}

// Node returns the node in the AST tree.
func (s *AnonymousStorage) Node() ast.Node { return s.Src }

// Expr returns the expression in the AST tree.
func (s *AnonymousStorage) Expr() ast.Expr { return s.Src }

// Type of the destination of the assignment.
func (s *AnonymousStorage) Type() Type { return s.Typ }

// NameDef returns the identifier identifying the storage.
func (s *AnonymousStorage) NameDef() *ast.Ident { return s.Src }

// Same returns true if the other storage is this storage.
func (s *AnonymousStorage) Same(o Storage) bool {
	return Storage(s) == o
}

func (*LocalVarStorage) node()    {}
func (*LocalVarStorage) storage() {}

// Node returns the node in the AST tree.
func (s *LocalVarStorage) Node() ast.Node { return s.Src }

// Expr returns the expression in the AST tree.
func (s *LocalVarStorage) Expr() ast.Expr { return s.Src }

// Type of the destination of the assignment.
func (s *LocalVarStorage) Type() Type { return s.Typ }

// NameDef returns the identifier identifying the storage.
func (s *LocalVarStorage) NameDef() *ast.Ident { return s.Src }

// Same returns true if the other storage is this storage.
func (s *LocalVarStorage) Same(o Storage) bool {
	return Storage(s) == o
}

func (*StructFieldStorage) node()    {}
func (*StructFieldStorage) storage() {}

// Node returns the node in the AST tree.
func (s *StructFieldStorage) Node() ast.Node { return s.Expr() }

// Expr returns the expression in the AST tree.
func (s *StructFieldStorage) Expr() ast.Expr { return s.Sel.Expr() }

// Type of the destination of the assignment.
func (s *StructFieldStorage) Type() Type { return s.Sel.Type() }

// NameDef returns the identifier identifying the storage.
func (s *StructFieldStorage) NameDef() *ast.Ident { return s.Sel.Src.Sel }

// Same returns true if the other storage is this storage.
func (s *StructFieldStorage) Same(o Storage) bool {
	return Storage(s) == o
}

func (*FieldStorage) node()    {}
func (*FieldStorage) storage() {}

// Node returns the node in the AST tree.
func (s *FieldStorage) Node() ast.Node {
	return s.Field.Node()
}

// Type of the destination of the assignment.
func (s *FieldStorage) Type() Type { return s.Field.Type() }

// NameDef returns the identifier identifying the storage.
func (s *FieldStorage) NameDef() *ast.Ident { return s.Field.Name }

// Same returns true if the other storage is this storage.
func (s *FieldStorage) Same(o Storage) bool {
	oT, ok := o.(*FieldStorage)
	if !ok {
		return false
	}
	return s.Field == oT.Field
}

// ----------------------------------------------------------------------------
// Statements.
type (
	// BlockStmt is a braced statement list.
	BlockStmt struct {
		Src  *ast.BlockStmt
		List []Stmt
	}

	// Stmt is a statement that performs an action.
	// No value is being returned.
	Stmt interface {
		Node
		// stmtNode marks a structure as a statement structure.
		stmtNode()
		String() string
	}

	// ReturnStmt is a return statement in a function.
	ReturnStmt struct {
		Src     *ast.ReturnStmt
		Results []Expr
	}

	// CallExpr is a call expression such as a function or a macro.
	CallExpr interface {
		Expr
		Call() *ast.CallExpr
		FuncCall() *FuncCallExpr
	}

	// AssignCallStmt assigns the results of a function call returning more than one value to variables.
	AssignCallStmt struct {
		Src  *ast.AssignStmt
		Call CallExpr
		List []*AssignCallResult
	}

	// AssignExpr assigns an expression to a Assignable node.
	AssignExpr struct {
		Storage
		X Expr
	}

	// AssignCallResult assigns the result of a function call.
	AssignCallResult struct {
		Storage
		Call        CallExpr
		ResultIndex int
	}

	// AssignExprStmt assigns the results of expressions (possibly functions returning one value) to variables.
	AssignExprStmt struct {
		Src  *ast.AssignStmt
		List []*AssignExpr
	}

	// RangeStmt is a range statement in for loops.
	RangeStmt struct {
		Src        *ast.RangeStmt
		Key, Value Storage
		X          Expr
		Body       *BlockStmt
	}

	// IfStmt is a if Statements.
	IfStmt struct {
		Src  *ast.IfStmt
		Init Stmt
		Cond Expr
		Body *BlockStmt
		Else Stmt
	}

	// ExprStmt is a statement evaluating an expression with no return result.
	ExprStmt struct {
		Src *ast.ExprStmt
		X   Expr
	}

	// DeclStmt represents a statement that contains `var` declarations.
	DeclStmt struct {
		Src   ast.Node
		Decls []*VarSpec
	}
)

var (
	_ Stmt = (*BlockStmt)(nil)
	_ Stmt = (*ReturnStmt)(nil)
	_ Stmt = (*AssignCallStmt)(nil)
	_ Stmt = (*AssignExprStmt)(nil)
	_ Stmt = (*RangeStmt)(nil)
	_ Stmt = (*IfStmt)(nil)
	_ Stmt = (*ExprStmt)(nil)
	_ Stmt = (*DeclStmt)(nil)

	_ StorageWithValue = (*AssignExpr)(nil)
	_ StorageWithValue = (*AssignCallResult)(nil)
)

func (*BlockStmt) stmtNode() {}
func (*BlockStmt) node()     {}

// Node returns the node in the AST tree.
func (s *BlockStmt) Node() ast.Node { return s.Src }

func (*ReturnStmt) stmtNode() {}
func (*ReturnStmt) node()     {}

// Node returns the node in the AST tree.
func (s *ReturnStmt) Node() ast.Node { return s.Src }

// Type of the result being returned.
func (s *ReturnStmt) Type() Type {
	types := make([]Type, len(s.Results))
	for i, expr := range s.Results {
		types[i] = expr.Type()
	}
	return &TupleType{
		BaseType: BaseType[ast.Expr]{},
		Types:    types,
	}
}

func (*AssignCallStmt) stmtNode() {}
func (*AssignCallStmt) node()     {}

// Node returns the node in the AST tree.
func (s *AssignCallStmt) Node() ast.Node { return s.Src }

func (*AssignExprStmt) stmtNode() {}
func (*AssignExprStmt) node()     {}

// Node returns the node in the AST tree.
func (s *AssignExprStmt) Node() ast.Node { return s.Src }

func (*RangeStmt) stmtNode() {}
func (*RangeStmt) node()     {}

// Node returns the node in the AST tree.
func (s *RangeStmt) Node() ast.Node { return s.Src }

func (*IfStmt) stmtNode() {}
func (*IfStmt) node()     {}

// Node returns the node in the AST tree.
func (s *IfStmt) Node() ast.Node { return s.Src }

func (*ExprStmt) stmtNode() {}
func (*ExprStmt) node()     {}

// Node returns the node in the AST tree.
func (s *ExprStmt) Node() ast.Node { return s.Src }

func (*DeclStmt) node() {}

// Node returns the original source node, satisfying the Node interface part of Stmt.
func (s *DeclStmt) Node() ast.Node { return s.Src }

func (*DeclStmt) stmtNode() {}

func (*AssignExpr) storageValue() {}

// Value returns the expression stored in the storage.
func (a *AssignExpr) Value(Expr) Expr {
	return a.X
}

func (*AssignCallResult) storageValue() {}

// Value returns the expression stored in the storage.
func (a *AssignCallResult) Value(Expr) Expr {
	return &IndexExpr{
		Src: &ast.IndexExpr{X: a.Call.Call()},
		X:   a.Call,
		Typ: a.Type(),
		Index: &AtomicValueT[int64]{
			Src: &ast.BasicLit{},
			Val: int64(a.ResultIndex),
			Typ: Int64Type(),
		},
	}
}

// Blank characters when an identifier is not needed.
const Blank = "_"

// ValidName returns true if a name is a valid identifier.
func ValidName(n string) bool {
	return n != Blank && n != ""
}

// ValidIdent returns true if the ident points to a valid identifier.
func ValidIdent(ident *ast.Ident) bool {
	return ident != nil && ValidName(ident.Name)
}

// Shape returns the data type and axes of an array.
// An invalid data type is returned if the type is not a container.
// A slice returns a nil rank.
func Shape(tp Type) (ArrayRank, Type) {
	switch tpT := tp.(type) {
	case ArrayType:
		return tpT.Rank(), tpT.DataType()
	case *NamedType:
		return Shape(tpT.Underlying.Val())
	case *SliceType:
		return nil, tpT.DType.Val()
	default:
		return nil, InvalidType()
	}
}

// IsBoolOp returns true if op is an operator returning a boolean.
func IsBoolOp(op token.Token) bool {
	switch op {
	case token.EQL, token.NEQ, token.LEQ, token.GEQ, token.LSS, token.GTR:
		return true
	}
	return false
}

// TODO(b/400359274): disable for now because of a bug in type inference.
const enforceArrayChecks = false

// ToArrayTypeGivenShape converts a type to the underlying array type.
// Returns an error with the array type does not match the shape.
func ToArrayTypeGivenShape(typ Type, sh *shape.Shape) (ArrayType, error) {
	under := Underlying(typ)
	array, ok := under.(ArrayType)
	if !ok {
		return nil, errors.Errorf("cannot create array value: type %s has underlying type %s but want array type", typ.String(), under.String())
	}
	if !enforceArrayChecks {
		return array, nil
	}
	arrayElementKind := array.DataType().Kind()
	if arrayElementKind == irkind.Unknown {
		// TODO(396628508): result of a generic that has not been resolved by the compiler. Temporary until will support generics correctly.
		return array, nil
	}
	dt := arrayElementKind.DType()
	if dt != sh.DType {
		return array, errors.Errorf("cannot create array value: type %s has data type %s but shape has data type %s", typ.String(), dt, sh.DType.String())
	}
	return array, nil
}

func fullName(f PkgFunc) string {
	return f.File().Package.FullName() + "." + f.Name()
}
