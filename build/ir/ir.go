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

	"github.com/pkg/errors"
	"github.com/gx-org/backend/dtype"
	"github.com/gx-org/backend/shape"
)

// ----------------------------------------------------------------------------
// Types of node in the tree.
type (
	// Node in the tree.
	Node interface {
		// node marks a structure as a node structure.
		// It prevents external implementations of the interface.
		// It prevents using arbitrary structure in this package to be used as nodes.
		node()
	}

	// SourceNode is a node with a position in GX source code.
	SourceNode interface {
		Node
		Source() ast.Node
	}

	// Storage is a node that can store a value.
	// (e.g. a field in a function)
	Storage interface {
		SourceNode
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
		Value(Expr) AssignableExpr
	}

	// Value is a value used by the compiler. It can be any thing with a type (package, constant, ...).
	// This is a superset of AssignableExpr.
	Value interface {
		SourceNode
		Type() Type
	}

	// AssignableExpr is an expression which can be assigned to variable.
	// This excludes package or types.
	AssignableExpr interface {
		assignable()
		Expr
	}

	// WithStore is an expression which points to a storage.
	WithStore interface {
		Expr
		Store() Storage
	}
)

// ----------------------------------------------------------------------------
// Types definition.
type (
	// Type of a value.
	Type interface {
		StorageWithValue

		// Kind of the type.
		Kind() Kind

		// Equal returns true if other is the same type.
		Equal(Fetcher, Type) (bool, error)

		// AssignableTo reports whether a value of the type can be assigned to another.
		AssignableTo(Fetcher, Type) (bool, error)

		// ConvertibleTo reports whether a value of the type can be converted to another
		// (using static type casting).
		ConvertibleTo(Fetcher, Type) (bool, error)

		// String representation of the type.
		String() string
	}

	// Zeroer is a type able to create a zero value of the type as an expression.
	Zeroer interface {
		Type
		Zero() AssignableExpr
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
		SourceNode
		SlicerType

		// ArrayType returns the source code defining the array type.
		// May be nil.
		ArrayType() *ast.ArrayType

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

	atomicType struct {
		BaseType[ast.Expr]

		Knd Kind
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

	// TypeParamValue assigns a type to a field of a more generic type.
	TypeParamValue struct {
		Field *Field
		Typ   Type
	}

	// FuncType defines a function signature.
	FuncType struct {
		BaseType[*ast.FuncType]

		Receiver   *FieldList
		TypeParams *FieldList
		Params     *FieldList
		Results    *FieldList

		TypeParamsValues []TypeParamValue

		// CompEval is set to true if the function can be called at compilation time.
		CompEval bool
	}

	// SliceType defines the type for a slice.
	SliceType struct {
		BaseType[*ast.ArrayType]

		DType *TypeValExpr
		Rank  int
	}

	// arrayType defines the type of an array from code.
	arrayType struct {
		BaseType[*ast.ArrayType]

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
	_ Type             = (*FuncType)(nil)
	_ SlicerType       = (*SliceType)(nil)
	_ Type             = (*TupleType)(nil)
	_ ArrayType        = (*atomicType)(nil)
	_ ArrayType        = (*arrayType)(nil)
	_ Type             = (*TypeParam)(nil)
)

// DefaultFloatType is the default type used for a scalar.
var DefaultFloatType = Float32Type()

var (
	// DefaultIntKind is the default kind for integer.
	DefaultIntKind = Int64Kind

	// DefaultIntType is the default type used for an integer.
	DefaultIntType = TypeFromKind(Int64Kind)
)

// Int is the default integer for indices, for loops, etc.
// It needs to match DefaultIntKind above
type Int = int64

func (*TupleType) node() {}

// Kind returns the scalar kind.
func (s *TupleType) Kind() Kind { return TupleKind }

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
func (s *TupleType) Value(x Expr) AssignableExpr {
	return &TypeValExpr{X: x, Typ: s}
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
func (s *InterfaceType) Kind() Kind { return InterfaceKind }

// Equal returns true if other is the same type.
func (s *InterfaceType) Equal(Fetcher, Type) (bool, error) { return false, nil }

// AssignableTo reports whether a value of the type can be assigned to another.
// Always returns false.
func (s *InterfaceType) AssignableTo(Fetcher, Type) (bool, error) { return false, nil }

// ConvertibleTo reports whether a value of the type can be converted to another
// (using static type casting). Always returns false.
func (s *InterfaceType) ConvertibleTo(Fetcher, Type) (bool, error) { return false, nil }

// Value returns a value pointing to the receiver.
func (s *InterfaceType) Value(x Expr) AssignableExpr {
	return &TypeValExpr{X: x, Typ: s}
}

// String representation of the type.
func (s *InterfaceType) String() string { return s.Kind().String() }

// IsValid returns true if the type is valid.
func IsValid(tp Type) bool {
	return tp.Kind() != InvalidKind
}

func (*BuiltinType) node() {}

// Kind returns the scalar kind.
func (s *BuiltinType) Kind() Kind { return BuiltinKind }

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
func (s *BuiltinType) Value(x Expr) AssignableExpr {
	return &TypeValExpr{X: x, Typ: s}
}

// String representation of the type.
func (s *BuiltinType) String() string {
	return fmt.Sprint(s.Impl)
}

func (*atomicType) node()   {}
func (*atomicType) atomic() {}

// Kind returns the scalar kind.
func (s *atomicType) Kind() Kind { return s.Knd }

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
		if s.Knd == NumberIntKind && IsInteger(target) {
			return true, nil
		}
		if s.Knd == NumberFloatKind && IsFloat(target) {
			return true, nil
		}
		return s.equalAtomic(targetT)
	}
	dtypeEq, err := s.Equal(fetcher, targetT.DataType())
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
	if targetT.Rank().IsAtomic() {
		return s.Kind() == target.Kind() || SupportOperators(s) == SupportOperators(target), nil
	}
	return s.Rank().Equal(fetcher, targetT.Rank())
}

// Source returns the source code defining the type.
// Always returns nil.
func (s *atomicType) Source() ast.Node {
	return s.ArrayType()
}

// Value returns a value pointing to the receiver.
func (s *atomicType) Value(x Expr) AssignableExpr {
	return &TypeValExpr{X: x, Typ: s}
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
func (s *atomicType) ArrayType() *ast.ArrayType {
	return &ast.ArrayType{}
}

var (
	zero      = &NumberInt{Src: &ast.BasicLit{Value: "0"}, Val: big.NewInt(0)}
	zeroFloat = &NumberFloat{Src: &ast.BasicLit{Value: "0.0"}, Val: big.NewFloat(0)}
)

// Zero returns a zero expression of the same type.
func (s *atomicType) Zero() AssignableExpr {
	return &NumberCastExpr{
		X:   zero,
		Typ: s,
	}
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
func (s *NamedType) Kind() Kind { return s.Underlying.Typ.Kind() }

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
	return s.Underlying.Typ.Equal(fetcher, otherT.Underlying.Typ)
}

// AssignableTo reports if the type can be assigned to other.
func (s *NamedType) AssignableTo(fetcher Fetcher, target Type) (bool, error) {
	return s.Equal(fetcher, target)
}

// AssignableFrom reports whether a given source type is assignable to a named type.
func (s *NamedType) assignableFrom(fetcher Fetcher, source Type) (bool, error) {
	if target, ok := s.Underlying.Typ.(assignsFrom); ok {
		return target.assignableFrom(fetcher, source)
	}
	return s.Equal(fetcher, source)
}

// ConvertibleTo reports whether a value of the type can be converted to another
// (using static type casting).
func (s *NamedType) ConvertibleTo(fetcher Fetcher, target Type) (bool, error) {
	switch targetT := target.(type) {
	case *NamedType:
		return s.Equal(fetcher, targetT)
	case *TypeParam:
		return s.ConvertibleTo(fetcher, targetT.Field.Group.Type.Typ)
	}
	return s.Underlying.Typ.ConvertibleTo(fetcher, target)
}

func (s *NamedType) convertibleFrom(fetcher Fetcher, source Type) (bool, error) {
	return s.Underlying.Typ.Equal(fetcher, source)
}

// FullName returns the full name of the type, that is the full package path and the type name.
func (s *NamedType) FullName() string {
	return s.File.Package.Path + "." + s.Name()
}

// Source returns the node in the AST tree.
func (s *NamedType) Source() ast.Node { return s.Src }

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
func (s *NamedType) Value(x Expr) AssignableExpr {
	return &TypeValExpr{
		X:   x,
		Typ: s,
	}
}

// Same returns true if the other storage is this storage.
func (s *NamedType) Same(o Storage) bool {
	return Storage(s) == o
}

// String representation of the type.
func (s *NamedType) String() string {
	if s.File == nil {
		return s.Name()
	}
	return s.Package().Name.Name + "." + s.Name()
}

// Package returns the package to which the type belongs to.
func (s *NamedType) Package() *Package {
	return s.File.Package
}

// Underlying returns the underlying type of a type.
func Underlying(typ Type) Type {
	switch typT := typ.(type) {
	case *NamedType:
		return Underlying(typT.Underlying.Typ)
	case *TypeParam:
		return Underlying(typT.Field.Type())
	default:
		return typ
	}
}

func (*StructType) node() {}

// Kind returns the structure kind.
func (s *StructType) Kind() Kind { return StructKind }

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

// Source returns the node in the AST tree.
func (s *StructType) Source() ast.Node { return s.Src }

// Value returns a value pointing to the receiver.
func (s *StructType) Value(x Expr) AssignableExpr {
	return &TypeValExpr{X: x, Typ: s}
}

// String representation of the type.
func (s *StructType) String() string {
	return s.Kind().String()
}

// NumFields returns the number of fields in a structure.
func (s *StructType) NumFields() int {
	n := 0
	for _, group := range s.Fields.List {
		n += group.NumFields()
	}
	return n
}

func (*FuncType) node() {}

// Kind returns the function kind.
func (s *FuncType) Kind() Kind { return FuncKind }

// Equal returns true if other is the same type.
func (s *FuncType) Equal(fetcher Fetcher, other Type) (bool, error) {
	otherT, ok := other.(*FuncType)
	if !ok {
		return false, nil
	}
	return s.equal(fetcher, otherT)
}

// Equal returns true if other is the same type.
func (s *FuncType) equal(fetcher Fetcher, other *FuncType) (bool, error) {
	if s == other {
		return true, nil
	}
	recvOk, err := s.Receiver.Type().Equal(fetcher, other.Receiver.Type())
	if err != nil {
		return false, err
	}
	paramsOk, err := s.Params.Type().Equal(fetcher, other.Params.Type())
	if err != nil {
		return false, err
	}
	resultsOk, err := s.Results.Type().Equal(fetcher, other.Results.Type())
	if err != nil {
		return false, err
	}
	return recvOk && paramsOk && resultsOk, nil
}

// ReceiverField returns a field representing the receiver of the function, or nil if the function has no receiver.
func (s *FuncType) ReceiverField() *Field {
	if s == nil || s.Receiver == nil { // The function type can be nil for builtin functions.
		return nil
	}
	grp := s.Receiver.List[0]
	if len(grp.Fields) == 0 {
		return &Field{Group: grp}
	}
	return grp.Fields[0]
}

// AssignableTo reports if the type can be assigned to other.
func (s *FuncType) AssignableTo(fetcher Fetcher, other Type) (bool, error) {
	otherT, ok := other.(*FuncType)
	if ok {
		return s.equal(fetcher, otherT)
	}
	aFrom, ok := other.(assignsFrom)
	if !ok {
		return false, nil
	}
	return aFrom.assignableFrom(fetcher, s)
}

// ConvertibleTo reports whether a value of the type can be converted to another
// (using static type casting).
func (s *FuncType) ConvertibleTo(fetcher Fetcher, target Type) (bool, error) {
	return s.Equal(fetcher, target)
}

// Value returns a value pointing to the receiver.
func (s *FuncType) Value(x Expr) AssignableExpr {
	return &TypeValExpr{X: x, Typ: s}
}

// Source returns the node in the AST tree.
func (s *FuncType) Source() ast.Node { return s.Src }

func (*SliceType) node() {}

// Kind returns the slide kind.
func (s *SliceType) Kind() Kind { return SliceKind }

// Equal returns true if other is the same type.
func (s *SliceType) Equal(fetcher Fetcher, other Type) (bool, error) {
	otherT, ok := other.(*SliceType)
	if !ok {
		return false, nil
	}
	if s.Rank != otherT.Rank {
		return false, nil
	}
	return otherT.DType.Typ.Equal(fetcher, s.DType.Typ)
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
func (s *SliceType) Value(x Expr) AssignableExpr {
	return &TypeValExpr{X: x, Typ: s}
}

// Source returns the node in the AST tree.
func (s *SliceType) Source() ast.Node { return s.Src }

// ElementType returns the type of an element in that slice.
func (s *SliceType) ElementType() (Type, bool) {
	if s.Rank <= 0 {
		return InvalidType(), false
	}
	if s.Rank == 1 {
		return s.DType.Typ, true
	}
	return &SliceType{
		BaseType: s.BaseType,
		DType:    s.DType,
		Rank:     s.Rank - 1,
	}, true
}

// String representation of the type.
func (s *SliceType) String() string {
	dtype := "nil"
	if s.DType != nil {
		dtype = s.DType.String()
	}
	rank := "[?]"
	if s.Rank > 0 {
		rank = strings.Repeat("[]", s.Rank)
	}
	return rank + dtype
}

func (*arrayType) node() {}

// NewArrayType returns a new array from a data type and a rank.
func NewArrayType(src *ast.ArrayType, dtype Type, rank ArrayRank) ArrayType {
	if dtype == nil {
		dtype = UnknownType()
	}
	if rank == nil {
		rank = &RankInfer{}
	}
	return &arrayType{
		BaseType: BaseType[*ast.ArrayType]{Src: src},
		DTypeF:   dtype,
		RankF:    rank,
	}
}

// Kind returns the tensor kind.
func (s *arrayType) Kind() Kind {
	if s.RankF == nil {
		return InvalidKind
	}
	if s.RankF.IsAtomic() {
		return s.DTypeF.Kind()
	}
	return ArrayKind
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
		BaseType: BaseType[*ast.ArrayType]{
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

// Value returns a value pointing to the receiver.
func (s *arrayType) Value(x Expr) AssignableExpr {
	return &TypeValExpr{X: x, Typ: s}
}

// Source returns the node in the AST tree.
func (s *arrayType) Source() ast.Node { return s.ArrayType() }

// ArrayType returns the source code defining the type.
func (s *arrayType) ArrayType() *ast.ArrayType {
	return s.Src
}

// Zero returns a zero literal of this type
func (s *arrayType) Zero() AssignableExpr {
	cst := &NumberCastExpr{
		X:   zero,
		Typ: s.DataType(),
	}
	if s.RankF.IsAtomic() {
		return cst
	}
	return &CastExpr{
		Typ: s,
		X:   cst,
	}
}

// String representation of the tensor type.
func (s *arrayType) String() string {
	dtype := "dtype"
	if s.DTypeF != nil {
		dtype = s.DTypeF.String()
	}
	rank := "[invalid]"
	if s.RankF != nil {
		rank = s.RankF.String()
	}
	return rank + dtype
}

func (*TypeParam) node() {}

// Kind of the type.
func (s *TypeParam) Kind() Kind {
	return s.Field.Type().Kind()
}

// Equal returns true if other is the same type.
func (s *TypeParam) Equal(fetcher Fetcher, typ Type) (bool, error) {
	switch typT := typ.(type) {
	case *TypeParam:
		return typT == s, nil
	case ArrayType:
		if !typT.Rank().IsAtomic() {
			return false, nil
		}
		return s.Equal(fetcher, typT.DataType())
	}
	return false, nil
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

// Source defining the type.
func (s *TypeParam) Source() ast.Node {
	return s.Field.Type().Source()
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
func (s *TypeParam) Value(x Expr) AssignableExpr {
	return &TypeValExpr{X: x, Typ: s}
}

// String representation of the type.
func (s *TypeParam) String() string {
	return s.Field.Type().String()
}

// IsStatic return true if the type is static, that is if the instance of the type
// can be evaluated when the graph is being constructed.
func IsStatic(tp Type) bool {
	switch tp.Kind() {
	case SliceKind:
		return IsStatic(tp.(*SliceType).DType.Typ)
	case IntIdxKind, IntLenKind:
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
		// BuildFuncType builds the type of a function given how it is called.
		BuildFuncType(fetcher Fetcher, call *CallExpr) (*FuncType, error)

		// Implementation of the function, provided by the backend.
		Implementation() any
	}

	// Func is a callable GX function.
	Func interface {
		Expr

		// Name of the function (without the package name).
		Name() string

		// Doc returns associated documentation or nil.
		Doc() *ast.CommentGroup

		// File owning the function.
		File() *File

		// Type of the function.
		// If the function is generic, then the type will have been inferred by the compiler from the
		// types passed as args. Note that both FuncType() and Type() must return the same underlying
		// value, though FuncType returns the concrete type.
		FuncType() *FuncType
	}

	// PkgFunc is a function declared at the package level.
	PkgFunc interface {
		pkgFunc()
		StorageWithValue
		Func
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
	}

	// FuncBuiltin is a function provided by a backend.
	// The implementation is opaque to GX.
	FuncBuiltin struct {
		FFile *File
		Src   *ast.FuncDecl
		FType *FuncType
		Impl  FuncImpl
	}

	// Macro is a function that takes a set of IR nodes as an input
	// and returns a new set of IR nodes (or a compiler error).
	// An example of such function is math/graph.Grad.
	// The implementation is opaque to GX.
	Macro struct {
		FFile *File
		Src   *ast.FuncDecl
		FType *FuncType

		BuildSynthetic any
	}

	// FuncLit is a function literal.
	FuncLit struct {
		Src   *ast.FuncLit
		FType *FuncType
		FFile *File
		Body  *BlockStmt
	}

	// ImportDecl imports a package.
	ImportDecl struct {
		Src  *ast.ImportSpec
		Path string
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
		Val   AssignableExpr
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
	_ Node             = (*Package)(nil)
	_ StorageWithValue = (*ImportDecl)(nil)
	_ Node             = (*ConstSpec)(nil)
	_ StorageWithValue = (*ConstExpr)(nil)
	_ Node             = (*VarSpec)(nil)
	_ Storage          = (*VarExpr)(nil)
	_ PkgFunc          = (*FuncBuiltin)(nil)
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
	first := name[:1]
	return strings.ToUpper(first) == first
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

// Source returns the node in the AST tree.
func (s *FuncDecl) Source() ast.Node { return s.Src }

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
func (s *FuncDecl) Value(x Expr) AssignableExpr {
	return &FuncValExpr{
		X: x,
		F: s,
		T: s.FType,
	}
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

func (*FuncBuiltin) node()         {}
func (*FuncBuiltin) staticValue()  {}
func (*FuncBuiltin) storage()      {}
func (*FuncBuiltin) storageValue() {}
func (*FuncBuiltin) pkgFunc()      {}

// Source returns the node in the AST tree.
func (s *FuncBuiltin) Source() ast.Node { return s.Src }

// Name of the function. Returns an empty string if the function is anonymous.
func (s *FuncBuiltin) Name() string {
	return s.NameDef().Name
}

// NameDef is the name definition of the function.
func (s *FuncBuiltin) NameDef() *ast.Ident {
	return s.Src.Name
}

// Value returns a reference to the function.
func (s *FuncBuiltin) Value(x Expr) AssignableExpr {
	return &FuncValExpr{
		X: x,
		F: s,
		T: s.FType,
	}
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

func (*FuncLit) node()         {}
func (*FuncLit) staticValue()  {}
func (*FuncLit) assignable()   {}
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
func (s *FuncLit) Value(x Expr) AssignableExpr {
	return &FuncValExpr{
		X: x,
		F: s,
		T: s.FType,
	}
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

// Source returns the node in the AST tree.
func (s *FuncLit) Source() ast.Node { return s.Src }

// Expr returns the expression in the source code.
func (s *FuncLit) Expr() ast.Expr { return s.Src }

// String representation of the literal.
func (s *FuncLit) String() string {
	return "func {...}"
}

func (*Macro) node()         {}
func (*Macro) staticValue()  {}
func (*Macro) storage()      {}
func (*Macro) storageValue() {}
func (*Macro) pkgFunc()      {}

// Name of the function.
func (s *Macro) Name() string {
	return s.Src.Name.Name
}

// NameDef is the name definition of the function.
func (s *Macro) NameDef() *ast.Ident {
	return s.Src.Name
}

// Value returns a reference to the function.
func (s *Macro) Value(x Expr) AssignableExpr {
	return &FuncValExpr{
		X: x,
		F: s,
		T: s.FuncType(),
	}
}

// Doc returns associated documentation or nil.
func (s *Macro) Doc() *ast.CommentGroup {
	return nil
}

// File declaring the function literal.
func (s *Macro) File() *File {
	return s.FFile
}

// FuncType returns the concrete type of the function.
func (s *Macro) FuncType() *FuncType {
	return s.FType
}

// Type returns the type of the function.
func (s *Macro) Type() Type {
	return s.FuncType()
}

// Source returns the node in the AST tree.
func (s *Macro) Source() ast.Node {
	return s.Src
}

// Same returns true if the other storage is this storage.
func (s *Macro) Same(o Storage) bool {
	return Storage(s) == o
}

// String representation of the literal.
func (s *Macro) String() string {
	return fmt.Sprintf("metafunc %s", s.Name())
}

func (*ImportDecl) node()         {}
func (*ImportDecl) staticValue()  {}
func (*ImportDecl) storage()      {}
func (*ImportDecl) storageValue() {}

// Source returns the node in the AST tree.
func (s *ImportDecl) Source() ast.Node {
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
func (s *ImportDecl) Value(x Expr) AssignableExpr {
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

// Source returns the node in the AST tree.
func (cst *ConstExpr) Source() ast.Node {
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
	return cst.Val.Type()
}

// Value assigned to the constant.
func (cst *ConstExpr) Value(Expr) AssignableExpr {
	return cst.Val
}

func (*VarSpec) node() {}

func (*VarExpr) node()    {}
func (*VarExpr) storage() {}

// Source returns the node in the AST tree.
func (vr *VarExpr) Source() ast.Node {
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
// Array axes specification

const (
	// DefineAxisGroup is the prefix used in parameters to define an axis group.
	DefineAxisGroup = "___"
	// DefineAxisLength is the prefix used in parameters to define an axis length.
	DefineAxisLength = "_"
)

type (
	// ArrayRank of an array.
	ArrayRank interface {
		Expr
		nodeRank()

		// IsAtomic returns true if the rank has no axes.
		IsAtomic() bool

		// Axes returns all axis in the rank.
		Axes() []AxisLengths

		// Equal returns true if two ranks are equal.
		Equal(Fetcher, ArrayRank) (bool, error)

		// AssignableTo returns true if this rank can be assigned to the destination rank.
		AssignableTo(Fetcher, ArrayRank) (bool, error)

		// ConvertibleTo returns true if this rank can be converted to the destination rank.
		ConvertibleTo(Fetcher, ArrayRank) (bool, error)

		// SubRank returns the rank with the top-axis removed.
		SubRank() (ArrayRank, bool)

		// String representation of the rank.
		String() string
	}

	// Rank with a known number of axes.
	// The length of some axes may still be unknown.
	Rank struct {
		Src *ast.ArrayType
		Ax  []AxisLengths
	}

	// RankInfer is a rank determined at compile time
	// (specified using ___ or ...).
	RankInfer struct {
		Src *ast.ArrayType
		Rnk ArrayRank
	}

	// AxisLengths specification of an array.
	AxisLengths interface {
		axExprString() string
		Expr

		// AxisValue returns the value associated with the axis.
		// Can return nil if the axis has not been resolved.
		AxisValue() AssignableExpr

		// Equal returns true if two axis lengths have been resolved and are equal.
		// Returns an error if one of the axis has not been resolved.
		Equal(Fetcher, AxisLengths) (bool, error)

		// AssignableTo returns true if this axis length can be assigned to another.
		AssignableTo(Fetcher, AxisLengths) (bool, error)

		// String representation of the axis length.
		String() string
	}

	// AxisInfer is an array axis specified as "_" and inferred by the compiler.
	AxisInfer struct {
		Src *ast.Ident
		X   AxisLengths
	}

	// AxisExpr is an array axis specified using an expression.
	AxisExpr struct {
		// Source of the axis expression.
		// May be different from the source of the expression, for example
		// the expression is formed from a function call.
		Src ast.Expr
		// X computes the size of the axis.
		X AssignableExpr
	}
)

var (
	_ ArrayRank   = (*Rank)(nil)
	_ ArrayRank   = (*RankInfer)(nil)
	_ AxisLengths = (*AxisExpr)(nil)
	_ AxisLengths = (*AxisInfer)(nil)
)

func (*Rank) node()     {}
func (*Rank) nodeRank() {}

// NewRank returns a new rank from a slice of axis lengths.
func NewRank(axlens []int) *Rank {
	axes := make([]AxisLengths, len(axlens))
	for i, al := range axlens {
		axes[i] = &AxisExpr{
			X: &NumberCastExpr{
				X:   &NumberInt{Val: big.NewInt(int64(al))},
				Typ: IntLenType(),
			},
		}
	}
	return &Rank{Ax: axes}
}

// Type returns the rank type.
func (r *Rank) Type() Type { return RankType() }

// Source returns the source node defining the rank.
func (r *Rank) Source() ast.Node { return r.Src }

// Axes returns all axis in the rank.
func (r *Rank) Axes() []AxisLengths { return r.Ax }

// Equal returns true if other has the same number of axes and each axis has the same length.
func (r *Rank) Equal(fetcher Fetcher, other ArrayRank) (bool, error) {
	switch otherT := other.(type) {
	case *RankInfer:
		if otherT.Rnk == nil {
			return false, errors.Errorf("rank not inferred")
		}
		return r.equalRank(fetcher, otherT.Rnk)
	case *Rank:
		return r.equalRank(fetcher, otherT)
	default:
		return false, errors.Errorf("rank type %T not supported", otherT)
	}
}

func (r *Rank) equalRank(fetcher Fetcher, other ArrayRank) (bool, error) {
	// Check the number of axes.
	otherAx := other.Axes()
	if len(r.Ax) != len(otherAx) {
		return false, nil
	}
	// Check each axis.
	for i, dimX := range r.Ax {
		dimEq, err := dimX.Equal(fetcher, otherAx[i])
		if err != nil {
			return false, err
		}
		if !dimEq {
			return false, nil
		}
	}
	return true, nil
}

func (r *Rank) assignableTo(fetcher Fetcher, dst ArrayRank) (bool, error) {
	dstAx := dst.Axes()
	if len(r.Ax) != len(dstAx) {
		return false, nil
	}
	for i, dimThis := range r.Ax {
		ok, err := dimThis.AssignableTo(fetcher, dstAx[i])
		if !ok || err != nil {
			return ok, err
		}
	}
	return true, nil
}

// AssignableTo returns true if this rank can be assigned to the destination rank.
func (r *Rank) AssignableTo(fetcher Fetcher, dst ArrayRank) (bool, error) {
	switch dstT := dst.(type) {
	case *RankInfer:
		if dstT.Rnk == nil {
			return true, nil
		}
		return r.assignableTo(fetcher, dstT.Rnk)
	case *Rank:
		return r.assignableTo(fetcher, dstT)
	default:
		return false, errors.Errorf("rank type %T not supported", dstT)
	}
}

// ConvertibleTo returns true if this rank can be converted to the destination rank.
func (r *Rank) ConvertibleTo(fetcher Fetcher, dst ArrayRank) (bool, error) {
	var dstR ArrayRank
	switch dstT := dst.(type) {
	case *RankInfer:
		if dstT.Rnk == nil {
			return true, nil
		}
		dstR = dstT.Rnk
	case *Rank:
		dstR = dstT
	default:
		return false, errors.Errorf("rank type %T not supported", dstT)
	}
	thisSize := RankSize(r)
	otherSize := RankSize(dstR)
	return areEqual(fetcher, thisSize, otherSize)
}

var oneSize = &NumberCastExpr{
	X:   &NumberInt{Val: big.NewInt(1)},
	Typ: IntLenType(),
}

func exprSource(n SourceNode) ast.Expr {
	if n == nil {
		return nil
	}
	src := n.Source()
	if src == nil {
		return nil
	}
	expr, _ := src.(ast.Expr)
	return expr
}

// RankSize is the total number of elements across all axes.
func RankSize(r ArrayRank) Expr {
	var expr AssignableExpr = oneSize
	for _, axis := range r.Axes() {
		expr = &BinaryExpr{
			Src: &ast.BinaryExpr{
				Op: token.MUL,
				X:  exprSource(expr),
				Y:  exprSource(axis),
			},
			X:   expr,
			Y:   axis.AxisValue(),
			Typ: IntLenType(),
		}
	}
	return expr
}

// IsAtomic returns true if the rank is equals to zero
// (that is it has no axes).
func (r *Rank) IsAtomic() bool {
	return len(r.Ax) == 0
}

// AxisLengths of the rank.
func (r *Rank) AxisLengths() []AxisLengths {
	return r.Ax
}

// SubRank returns the rank with the top-axis removed
// or nil if the rank is already 0.
func (r *Rank) SubRank() (ArrayRank, bool) {
	if len(r.Ax) == 0 {
		return nil, false
	}
	return &Rank{Ax: append([]AxisLengths{}, r.Ax[1:]...)}, true
}

func (*RankInfer) node()     {}
func (*RankInfer) nodeRank() {}

// Source returns the node defining the rank.
func (r *RankInfer) Source() ast.Node { return r.Src }

// Type returns the rank type.
func (r *RankInfer) Type() Type { return RankType() }

// Axes returns all axis in the rank.
func (r *RankInfer) Axes() []AxisLengths {
	if r.Rnk == nil {
		return nil
	}
	return r.Rnk.Axes()
}

// Equal returns true if other has the same rank and dimensions.
func (r *RankInfer) Equal(fetcher Fetcher, other ArrayRank) (bool, error) {
	if r.Rnk != nil {
		return r.Rnk.Equal(fetcher, other)
	}
	switch otherT := other.(type) {
	case *RankInfer:
		return otherT.Rnk == nil, nil
	case *Rank:
		return false, nil
	default:
		return false, errors.Errorf("rank type %T not supported", otherT)
	}
}

// AssignableTo returns true if this rank can be assigned to the destination rank.
func (r *RankInfer) AssignableTo(fetcher Fetcher, dst ArrayRank) (bool, error) {
	if r.Rnk != nil {
		return r.Rnk.AssignableTo(fetcher, dst)
	}
	return true, nil
}

// ConvertibleTo returns true if this rank can be converted to the destination rank.
func (r *RankInfer) ConvertibleTo(fetcher Fetcher, dst ArrayRank) (bool, error) {
	if r.Rnk != nil {
		return r.Rnk.ConvertibleTo(fetcher, dst)
	}
	return true, nil
}

// IsAtomic returns true if the rank has no axes.
func (r *RankInfer) IsAtomic() bool {
	return false
}

// SubRank returns the rank with the top-axis removed.
func (r *RankInfer) SubRank() (ArrayRank, bool) {
	if r.Rnk == nil {
		return &RankInfer{}, true
	}
	return r.Rnk.SubRank()
}

func (*AxisInfer) node() {}

// Source returns the source expression specifying the dimension.
func (dm *AxisInfer) Source() ast.Node { return dm.Src }

// Type of the expression.
func (dm *AxisInfer) Type() Type { return IntLenType() }

// Expr returns how to compute the dimension as an expression.
func (dm *AxisInfer) Expr() ast.Expr { return dm.Src }

// Equal returns true if other has the axis length.
func (dm *AxisInfer) Equal(fetcher Fetcher, other AxisLengths) (bool, error) {
	switch otherT := other.(type) {
	case *AxisExpr:
		if dm.X == nil {
			return false, errors.Errorf("unresolved axis length")
		}
		return dm.X.Equal(fetcher, otherT)
	case *AxisInfer:
		if dm.X == nil && otherT.X == nil {
			return true, nil
		}
		if dm.X != nil && otherT.X != nil {
			return dm.X.Equal(fetcher, otherT.X)
		}
		return false, nil
	default:
		return false, errors.Errorf("cannot compare with axis type %T: not supported", otherT)
	}
}

// AssignableTo returns true if a dimension can be assigned to another.
func (dm *AxisInfer) AssignableTo(fetcher Fetcher, dst AxisLengths) (bool, error) {
	if dm.X != nil {
		return dm.X.AssignableTo(fetcher, dst)
	}
	return true, nil
}

// AxisValue returns the value assigned to the axis.
func (dm *AxisInfer) AxisValue() AssignableExpr {
	if dm.X == nil {
		return nil
	}
	return dm.X.AxisValue()
}

// Value of the axis.
func (dm *AxisInfer) Value(Expr) AssignableExpr {
	return dm.AxisValue()
}

func (*AxisExpr) node() {}

// Source returns the source expression specifying the dimension.
func (dm *AxisExpr) Source() ast.Node { return dm.X.Source() }

// NumAxes returns the number of axis represented by the group.
func (dm *AxisExpr) NumAxes() int { return 1 }

// AssignableTo returns true if a dimension can be assigned to another.
func (dm *AxisExpr) AssignableTo(fetcher Fetcher, dst AxisLengths) (bool, error) {
	switch dstT := dst.(type) {
	case *AxisExpr:
		return dm.Equal(fetcher, dst)
	case *AxisInfer:
		if dstT.X == nil {
			return true, nil
		}
		return dm.Equal(fetcher, dstT.X)
	default:
		return false, errors.Errorf("assigning an axis to an axis type %T not supported", dstT)
	}
}

// Equal returns true if other has the axis length.
func (dm *AxisExpr) Equal(fetcher Fetcher, other AxisLengths) (bool, error) {
	switch otherT := other.(type) {
	case *AxisExpr:
		return areEqual(fetcher, dm.X, otherT.X)
	case *AxisInfer:
		if otherT.X == nil {
			return false, errors.Errorf("cannot compare with an unresolved axis length")
		}
		return dm.Equal(fetcher, otherT.X)
	default:
		return false, errors.Errorf("cannot compare with axis type %T: not supported", otherT)
	}
}

// AxisValue returns the value assigned to the axis.
func (dm *AxisExpr) AxisValue() AssignableExpr { return dm.X }

// Value of the axis.
func (dm *AxisExpr) Value(Expr) AssignableExpr { return dm.AxisValue() }

// Type of the expression.
func (dm *AxisExpr) Type() Type { return dm.X.Type() }

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
	_ SourceNode = (*FieldList)(nil)
	_ SourceNode = (*FieldGroup)(nil)
	_ Expr       = (*Field)(nil)
)

func (*FieldList) node() {}

// Source returns the source defining the field.
func (s *FieldList) Source() ast.Node {
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
				Name:  &ast.Ident{Name: "_"},
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

// Source returns the node in the AST tree.
func (s *FieldGroup) Source() ast.Node {
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

// Source returns the source defining the field.
func (s *Field) Source() ast.Node {
	if s.Name == nil {
		return s.Group.Source()
	}
	return s.Name
}

// Storage returns a storage pointing to the field.
func (s *Field) Storage() *FieldStorage {
	return &FieldStorage{Field: s}
}

// Type returns the type of the field.
func (s *Field) Type() Type {
	return s.Group.Type.Typ
}

// String returns a string representation of the field.
func (s *Field) String() string {
	return fmt.Sprintf("%s:%v", s.Name.Name, s.Group.Type)
}

// ----------------------------------------------------------------------------
// Expressions.
type (
	// Expr is an expression that returns a (typed) result.
	Expr interface {
		SourceNode
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
		AssignableExpr
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
		Elts []AssignableExpr
	}

	// SliceLitExpr is a slice literal.
	SliceLitExpr struct {
		Src  ast.Expr
		Typ  Type
		Elts []AssignableExpr
	}

	// FieldLit assigns a value to a field in a structure literal.
	FieldLit struct {
		*FieldStorage
		X AssignableExpr
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
		X   AssignableExpr
	}

	// ParenExpr is a parenthesized expression.
	ParenExpr struct {
		Src *ast.ParenExpr
		X   Expr
	}

	// BinaryExpr is an operator with two arguments.
	BinaryExpr struct {
		Src  *ast.BinaryExpr
		X, Y AssignableExpr
		Typ  Type
	}

	// TypeValExpr is an expression pointing to a type.
	TypeValExpr struct {
		X   Expr
		Typ Type
	}

	// FuncValExpr is an expression pointing to a function.
	FuncValExpr struct {
		X Expr
		F Storage
		// T is the function type if the function has been specialised by the call
		// (in the context of generic functions).
		T *FuncType
	}

	// CallExpr is an expression calling a function.
	CallExpr struct {
		Src    *ast.CallExpr
		Args   []AssignableExpr
		Callee *FuncValExpr
	}

	// CallResultExpr represents the ith result of a function call as an expression.
	CallResultExpr struct {
		Index int
		Call  *CallExpr
	}

	// TypeCastExpr is an abstract type conversion.
	TypeCastExpr interface {
		AssignableExpr
		Orig() Expr
	}

	// CastExpr casts a type to another.
	CastExpr struct {
		Src *ast.CallExpr
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
	_ AssignableExpr   = (*NumberCastExpr)(nil)
	_ AtomicValue      = (*AtomicValueT[int32])(nil)
	_ AssignableExpr   = (*StringLiteral)(nil)
	_ AssignableExpr   = (*ArrayLitExpr)(nil)
	_ AssignableExpr   = (*SliceLitExpr)(nil)
	_ AssignableExpr   = (*StructLitExpr)(nil)
	_ StorageWithValue = (*FieldLit)(nil)
	_ AssignableExpr   = (*UnaryExpr)(nil)
	_ AssignableExpr   = (*ParenExpr)(nil)
	_ WithStore        = (*ParenExpr)(nil)
	_ AssignableExpr   = (*BinaryExpr)(nil)
	_ AssignableExpr   = (*CallExpr)(nil)
	_ AssignableExpr   = (*CallResultExpr)(nil)
	_ TypeCastExpr     = (*CastExpr)(nil)
	_ TypeCastExpr     = (*TypeAssertExpr)(nil)
	_ AssignableExpr   = (*ValueRef)(nil)
	_ WithStore        = (*ValueRef)(nil)
	_ AssignableExpr   = (*PackageRef)(nil)
	_ AssignableExpr   = (*SelectorExpr)(nil)
	_ WithStore        = (*SelectorExpr)(nil)
	_ AssignableExpr   = (*IndexExpr)(nil)
	_ AssignableExpr   = (*IndexListExpr)(nil)
	_ Expr             = (*EinsumExpr)(nil)
	_ RuntimeValueExpr = (*RuntimeValueExprT[RuntimeValue])(nil)
	_ Expr             = (*Tuple)(nil)

	_ AssignableExpr = (*FuncValExpr)(nil)
	_ WithStore      = (*FuncValExpr)(nil)
	_ AssignableExpr = (*TypeValExpr)(nil) // Use AssignableExpr to store a type in a field (in function type params).
	_ WithStore      = (*TypeValExpr)(nil)
)

func (s *NumberFloat) node()       {}
func (s *NumberFloat) numberExpr() {}

// assignable: a number is not assignable
// (i.e. it cannot be assigned to a value without a cast)
// but it still supports the AssignableExpr interface to use numbers in binary expressions.
func (s *NumberFloat) assignable() {}

// Zero returns a zero float number.
func (s *NumberFloat) Zero() AssignableExpr {
	return zeroFloat
}

// Expr returns the AST expression.
func (s *NumberFloat) Expr() ast.Expr { return s.Src }

// Source returns the node in the AST tree.
func (s *NumberFloat) Source() ast.Node { return s.Src }

// Type returns the type returned by the function call.
func (s *NumberFloat) Type() Type { return NumberFloatType() }

// DefaultType returns the default type of the number.
func (s NumberFloat) DefaultType() Type { return TypeFromKind(Float64Kind) }

func (s *NumberInt) node()       {}
func (s *NumberInt) numberExpr() {}

// assignable: a number is not assignable
// (i.e. it cannot be assigned to a value without a cast)
// but it still supports the AssignableExpr interface to use numbers in binary expressions.
func (s *NumberInt) assignable() {}

// Zero returns a zero float number.
func (s *NumberInt) Zero() AssignableExpr {
	return zero
}

// Expr returns the AST expression.
func (s *NumberInt) Expr() ast.Expr { return s.Src }

// Source returns the node in the AST tree.
func (s *NumberInt) Source() ast.Node { return s.Src }

// Type returns the type returned by the function call.
func (s *NumberInt) Type() Type { return NumberIntType() }

// DefaultType returns the default type of the number.
func (s NumberInt) DefaultType() Type { return TypeFromKind(Int64Kind) }

func (s *NumberCastExpr) node()        {}
func (s *NumberCastExpr) staticValue() {}
func (s *NumberCastExpr) assignable()  {}

// Source returns the node in the AST tree.
func (s *NumberCastExpr) Source() ast.Node { return s.X.Source() }

// Type returns the type returned by the function call.
func (s *NumberCastExpr) Type() Type { return s.Typ }

// String representation of the number.
func (s *NumberCastExpr) String() string { return s.X.String() }

func (s *StringLiteral) node()        {}
func (s *StringLiteral) staticValue() {}
func (s *StringLiteral) assignable()  {}

// Expr returns the AST expression.
func (s *StringLiteral) Expr() ast.Expr { return s.Src }

// Source returns the node in the AST tree.
func (s *StringLiteral) Source() ast.Node { return s.Src }

// Type returns the type returned by the function call.
func (s *StringLiteral) Type() Type { return StringType() }

// String representation of the number.
func (s *StringLiteral) String() string { return s.Src.Value }

func (s *AtomicValueT[T]) node()        {}
func (s *AtomicValueT[T]) atomicValue() {}
func (s *AtomicValueT[T]) assignable()  {}

// Expr returns the AST expression.
func (s *AtomicValueT[T]) Expr() ast.Expr { return s.Src }

// Source returns the node in the AST tree.
func (s *AtomicValueT[T]) Source() ast.Node { return s.Src }

// Type returns the type returned by the function call.
func (s *AtomicValueT[T]) Type() Type { return s.Typ }

// String representation.
func (s *AtomicValueT[T]) String() string { return fmt.Sprint(s.Val) }

func (s *ArrayLitExpr) node()       {}
func (s *ArrayLitExpr) assignable() {}

// Source returns the node in the AST tree.
func (s *ArrayLitExpr) Source() ast.Node {
	return s.Src
}

// Type returns the type returned by the function call.
func (s *ArrayLitExpr) Type() Type { return s.Typ }

// Expr returns the AST expression.
func (s *ArrayLitExpr) Expr() ast.Expr { return s.Src }

// Values returns the expressions defining the values of the array.
func (s *ArrayLitExpr) Values() []AssignableExpr { return s.Elts }

// NewFromValues returns a new literal of the same type from a slice of values.
func (s *ArrayLitExpr) NewFromValues(elts []AssignableExpr) *ArrayLitExpr {
	return &ArrayLitExpr{
		Typ:  s.Typ,
		Elts: elts,
	}
}

func (s *StructLitExpr) node()       {}
func (s *StructLitExpr) assignable() {}

// Source returns the node in the AST tree.
func (s *StructLitExpr) Source() ast.Node { return s.Src }

// Type returns the type returned by the function call.
func (s *StructLitExpr) Type() Type { return s.Typ }

// Expr returns the AST expression.
func (s *StructLitExpr) Expr() ast.Expr { return s.Src }

// String representation.
func (s *StructLitExpr) String() string { return s.Type().String() }

func (s *SliceLitExpr) node()       {}
func (s *SliceLitExpr) assignable() {}

// Source returns the node in the AST tree.
func (s *SliceLitExpr) Source() ast.Node { return s.Src }

// Type returns the type returned by the function call.
func (s *SliceLitExpr) Type() Type { return s.Typ }

// Expr returns the AST expression.
func (s *SliceLitExpr) Expr() ast.Expr { return s.Src }

func (s *UnaryExpr) node()       {}
func (s *UnaryExpr) assignable() {}

// Source returns the node in the AST tree.
func (s *UnaryExpr) Source() ast.Node { return s.Src }

// Type returns the type returned by the expression.
func (s *UnaryExpr) Type() Type { return s.X.Type() }

// Expr returns the expression in the source code.
func (s *UnaryExpr) Expr() ast.Expr { return s.Src }

// String representation.
func (s *UnaryExpr) String() string { return s.Src.Op.String() + s.X.String() }

func (s *ParenExpr) node()       {}
func (s *ParenExpr) assignable() {}

// Source returns the node in the AST tree.
func (s *ParenExpr) Source() ast.Node { return s.Src }

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

func (s *BinaryExpr) node()       {}
func (s *BinaryExpr) assignable() {}

// Source returns the node in the AST tree.
func (s *BinaryExpr) Source() ast.Node { return s.Src }

// Type returned by the expression.
func (s *BinaryExpr) Type() Type { return s.Typ }

// Expr returns the expression in the source code.
func (s *BinaryExpr) Expr() ast.Expr { return s.Src }

func (s *TypeValExpr) node()       {}
func (s *TypeValExpr) typeExpr()   {}
func (s *TypeValExpr) assignable() {}

// Source returns the node in the AST tree.
func (s *TypeValExpr) Source() ast.Node { return s.X.Source() }

// Type of the function being called.
func (s *TypeValExpr) Type() Type {
	// The value of the reference is stored in the Typ field.
	// The type of that value is a meta-type.
	return MetaType()
}

// Store storing the type.
func (s *TypeValExpr) Store() Storage { return s.Typ }

// String representation.
func (s *TypeValExpr) String() string { return s.Typ.String() }

func (s *FuncValExpr) node()       {}
func (s *FuncValExpr) assignable() {}

// Name returns the name of a function
func (s *FuncValExpr) Name() string {
	return s.F.NameDef().Name
}

// Source returns the node in the AST tree.
func (s *FuncValExpr) Source() ast.Node {
	if s.X != nil {
		return s.X.Source()
	}
	return s.F.Source()
}

// Type of the function being called.
func (s *FuncValExpr) Type() Type {
	return s.T
}

// Store storing the type.
func (s *FuncValExpr) Store() Storage { return s.F }

// String representation.
func (s *FuncValExpr) String() string { return s.X.String() }

func (s *CallExpr) node()       {}
func (s *CallExpr) assignable() {}

// Source returns the node in the AST tree.
func (s *CallExpr) Source() ast.Node { return s.Src }

// Type returns the type returned by the function call.
// Use CallExpr.Func.Type to get the type of the function being called.
func (s *CallExpr) Type() Type { return s.Callee.T.Results.Type() }

// Expr returns the expression in the source code.
func (s *CallExpr) Expr() ast.Expr { return s.Src }

// ExprFromResult returns an expression pointing to the ith result of a function call.
func (s *CallExpr) ExprFromResult(i int) *CallResultExpr {
	return &CallResultExpr{
		Index: i,
		Call:  s,
	}
}

func (s *CallResultExpr) node()       {}
func (s *CallResultExpr) assignable() {}

// Source returns the node in the AST tree.
func (s *CallResultExpr) Source() ast.Node { return s.Call.Source() }

// Type returns the type returned by the function call.
// Use CallExpr.Func.Type to get the type of the function being called.
func (s *CallResultExpr) Type() Type {
	return s.Call.Callee.T.Results.Fields()[s.Index].Type()
}

// Expr returns the expression in the source code.
func (s *CallResultExpr) Expr() ast.Expr { return s.Call.Expr() }

// String representation.
func (s *CallResultExpr) String() string { return fmt.Sprintf("%T", s) }

func (s *CastExpr) node()       {}
func (s *CastExpr) assignable() {}

// Source returns the node in the AST tree.
func (s *CastExpr) Source() ast.Node { return s.Src }

// Type returns the target type of the cast.
func (s *CastExpr) Type() Type { return s.Typ }

// Expr returns the expression in the source code.
func (s *CastExpr) Expr() ast.Expr { return s.Src }

// Orig returns the expression being casted.
func (s *CastExpr) Orig() Expr { return s.X }

func (s *TypeAssertExpr) node()       {}
func (s *TypeAssertExpr) assignable() {}

// Source returns the node in the AST tree.
func (s *TypeAssertExpr) Source() ast.Node { return s.Src }

// Type returns the target type of the cast.
func (s *TypeAssertExpr) Type() Type { return s.Typ }

// Expr returns the expression in the source code.
func (s *TypeAssertExpr) Expr() ast.Expr { return s.Src }

// Orig returns the expression being casted.
func (s *TypeAssertExpr) Orig() Expr { return s.X }

func (s *ValueRef) node()       {}
func (s *ValueRef) assignable() {}

// Source returns the node in the AST tree.
func (s *ValueRef) Source() ast.Node { return s.Src }

// Type returns the type returned by the function call.
// Use CallExpr.Func.Type to get the type of the function being called.
func (s *ValueRef) Type() Type { return s.Stor.Type() }

// Expr returns the expression in the source code.
func (s *ValueRef) Expr() ast.Expr { return s.Src }

// Store returns the storage referenced by this expression.
func (s *ValueRef) Store() Storage { return s.Stor }

// String representation.
func (s *ValueRef) String() string { return s.Src.Name }

func (s *PackageRef) node()       {}
func (s *PackageRef) assignable() {}

// Source returns the node in the AST tree.
func (s *PackageRef) Source() ast.Node { return s.X.Source() }

// Type returns the package type.
func (s *PackageRef) Type() Type { return PackageType() }

// String representation.
func (s *PackageRef) String() string { return s.X.String() }

func (*FieldLit) node()         {}
func (*FieldLit) storageValue() {}

// Value returns the value stored in the field.
func (lit *FieldLit) Value(Expr) AssignableExpr {
	return lit.X
}

// Same returns true if the other storage is this storage.
func (lit *FieldLit) Same(o Storage) bool {
	return Storage(lit) == o
}

func (s *SelectorExpr) node()       {}
func (s *SelectorExpr) assignable() {}

// Source returns the node in the AST tree.
func (s *SelectorExpr) Source() ast.Node { return s.Src }

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

func (s *IndexExpr) node()       {}
func (s *IndexExpr) assignable() {}

// Source returns the node in the AST tree.
func (s *IndexExpr) Source() ast.Node { return s.Src }

// Type returns the type of the indexed element.
func (s *IndexExpr) Type() Type { return s.Typ }

// Expr returns the AST expression.
func (s *IndexExpr) Expr() ast.Expr { return s.Src }

func (s *IndexListExpr) node()       {}
func (s *IndexListExpr) assignable() {}

// Source returns the node in the AST tree.
func (s *IndexListExpr) Source() ast.Node { return s.Src }

// Type returns the type of the indexed element.
func (s *IndexListExpr) Type() Type { return s.Typ }

// Expr returns the AST expression.
func (s *IndexListExpr) Expr() ast.Expr { return s.Src }

// String representation.
func (s *IndexListExpr) String() string { return fmt.Sprintf("%T", s) }

func (s *EinsumExpr) node()       {}
func (s *EinsumExpr) assignable() {}

// Source returns the node in the AST tree.
func (s *EinsumExpr) Source() ast.Node { return s.Src }

// Type returns the type returned by the einsum.
func (s *EinsumExpr) Type() Type { return s.Typ }

// Expr returns the expression in the source code.
func (s *EinsumExpr) Expr() ast.Expr { return s.Src }

// String representation.
func (s *EinsumExpr) String() string { return fmt.Sprintf("%T", s) }

func (s *RuntimeValueExprT[T]) node() {}

// Source returns the node in the AST tree.
func (s *RuntimeValueExprT[T]) Source() ast.Node { return s.Src }

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
		src = s.Exprs[0].Source().(ast.Expr)
	}
	return &TupleType{
		BaseType: BaseType[ast.Expr]{Src: src},
		Types:    typs,
	}
}

// Source returns the node in the AST tree.
func (s *Tuple) Source() ast.Node {
	if len(s.Exprs) == 0 {
		return nil
	}
	return s.Exprs[0].Source()
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

	// SpecialisedFunc is a storage for a function that has been specialised
	SpecialisedFunc struct {
		X Expr
		F *FuncValExpr
		T *FuncType
	}

	// AxLengthName defines a name for the length of an axis or a group of axes.
	AxLengthName struct {
		Src *ast.Ident
		Typ Type
	}
)

var (
	_ Storage          = (*AnonymousStorage)(nil)
	_ Storage          = (*LocalVarStorage)(nil)
	_ Storage          = (*StructFieldStorage)(nil)
	_ Storage          = (*FieldStorage)(nil)
	_ Storage          = (*AxLengthName)(nil)
	_ StorageWithValue = (*SpecialisedFunc)(nil)
)

func (*AnonymousStorage) node()    {}
func (*AnonymousStorage) storage() {}

// Source returns the node in the AST tree.
func (s *AnonymousStorage) Source() ast.Node { return s.Src }

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

// Source returns the node in the AST tree.
func (s *LocalVarStorage) Source() ast.Node { return s.Src }

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

// Source returns the node in the AST tree.
func (s *StructFieldStorage) Source() ast.Node { return s.Expr() }

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

// Source returns the node in the AST tree.
func (s *FieldStorage) Source() ast.Node {
	return s.Field.Source()
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

func (*AxLengthName) node()    {}
func (*AxLengthName) storage() {}

// Source returns the node in the AST tree.
func (s *AxLengthName) Source() ast.Node {
	return s.Src
}

// Type of the destination of the assignment.
func (s *AxLengthName) Type() Type { return s.Typ }

// NameDef returns the identifier identifying the storage.
func (s *AxLengthName) NameDef() *ast.Ident { return s.Src }

// Same returns true if the other storage is this storage.
func (s *AxLengthName) Same(o Storage) bool {
	return Storage(s) == o
}

func (*SpecialisedFunc) node()         {}
func (*SpecialisedFunc) storage()      {}
func (*SpecialisedFunc) storageValue() {}

// Source returns the node in the AST tree.
func (s *SpecialisedFunc) Source() ast.Node {
	return s.X.Source()
}

// Type of the destination of the assignment.
func (s *SpecialisedFunc) Type() Type { return s.T }

// NameDef returns the identifier identifying the storage.
func (s *SpecialisedFunc) NameDef() *ast.Ident { return s.F.F.NameDef() }

// Value returns the function as a value.
func (s *SpecialisedFunc) Value(x Expr) AssignableExpr {
	return &FuncValExpr{
		X: x,
		F: s,
		T: s.T,
	}
}

// Same returns true if the other storage is this storage.
func (s *SpecialisedFunc) Same(o Storage) bool {
	return Storage(s) == o
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
		SourceNode
		// stmtNode marks a structure as a statement structure.
		stmtNode()
		String() string
	}

	// ReturnStmt is a return statement in a function.
	ReturnStmt struct {
		Src     *ast.ReturnStmt
		Results []Expr
	}

	// AssignCallStmt assigns the results of a function call returning more than one value to variables.
	AssignCallStmt struct {
		Src  *ast.AssignStmt
		Call *CallExpr
		List []*AssignCallResult
	}

	// AssignExpr assigns an expression to a Assignable node.
	AssignExpr struct {
		Storage
		X AssignableExpr
	}

	// AssignCallResult assigns the result of a function call.
	AssignCallResult struct {
		Storage
		Call        *CallExpr
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
		X          AssignableExpr
		Body       *BlockStmt
	}

	// IfStmt is a if Statements.
	IfStmt struct {
		Src  *ast.IfStmt
		Init Stmt
		Cond AssignableExpr
		Body *BlockStmt
		Else Stmt
	}

	// ExprStmt is a statement evaluating an expression with no return result.
	ExprStmt struct {
		Src *ast.ExprStmt
		X   Expr
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

	_ StorageWithValue = (*AssignExpr)(nil)
	_ StorageWithValue = (*AssignCallResult)(nil)
)

func (*BlockStmt) stmtNode() {}
func (*BlockStmt) node()     {}

// Source returns the node in the AST tree.
func (s *BlockStmt) Source() ast.Node { return s.Src }

func (*ReturnStmt) stmtNode() {}
func (*ReturnStmt) node()     {}

// Source returns the node in the AST tree.
func (s *ReturnStmt) Source() ast.Node { return s.Src }

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

// Source returns the node in the AST tree.
func (s *AssignCallStmt) Source() ast.Node { return s.Src }

func (*AssignExprStmt) stmtNode() {}
func (*AssignExprStmt) node()     {}

// Source returns the node in the AST tree.
func (s *AssignExprStmt) Source() ast.Node { return s.Src }

func (*RangeStmt) stmtNode() {}
func (*RangeStmt) node()     {}

// Source returns the node in the AST tree.
func (s *RangeStmt) Source() ast.Node { return s.Src }

func (*IfStmt) stmtNode() {}
func (*IfStmt) node()     {}

// Source returns the node in the AST tree.
func (s *IfStmt) Source() ast.Node { return s.Src }

func (*ExprStmt) stmtNode() {}
func (*ExprStmt) node()     {}

// Source returns the node in the AST tree.
func (s *ExprStmt) Source() ast.Node { return s.Src }

func (*AssignExpr) storageValue() {}

// Value returns the expression stored in the storage.
func (a *AssignExpr) Value(Expr) AssignableExpr {
	return a.X
}

func (*AssignCallResult) storageValue() {}

// Value returns the expression stored in the storage.
func (a *AssignCallResult) Value(Expr) AssignableExpr {
	return &IndexExpr{
		Src: &ast.IndexExpr{X: a.Call.Src},
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
		return Shape(tpT.Underlying.Typ)
	case *SliceType:
		return nil, tpT.DType.Typ
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
	if arrayElementKind == UnknownKind {
		// TODO(396628508): result of a generic that has not been resolved by the compiler. Temporary until will support generics correctly.
		return array, nil
	}
	dt := arrayElementKind.DType()
	if dt != sh.DType {
		return array, errors.Errorf("cannot create array value: type %s has data type %s but shape has data type %s", typ.String(), dt, sh.DType.String())
	}
	return array, nil
}
