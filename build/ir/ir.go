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
	"slices"
	"strings"

	"github.com/pkg/errors"
	"github.com/gx-org/backend/dtype"
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
)

// ----------------------------------------------------------------------------
// Types definition.
type (
	// Type of a value.
	Type interface {
		Node

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
		Zero() Expr
	}

	// ArrayType is a type with a rank.
	ArrayType interface {
		Zeroer
		SourceNode

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

	atomicType struct {
		Knd Kind
	}

	// NamedType defines a new type from an existing type.
	NamedType struct {
		NameT      string
		Src        *ast.TypeSpec
		File       *File
		Underlying Type

		Methods []Func

		// ID is the index of the NamedType in the package.
		ID int
	}

	// StructType defines the type of a structure.
	StructType struct {
		Src    *ast.StructType
		Fields *FieldList
	}

	// InterfaceType defines the type of an interface.
	InterfaceType struct{}

	// FuncType defines a function signature.
	FuncType struct {
		Src *ast.FuncType

		Receiver   *NamedType
		TypeParams *FieldList
		Params     *FieldList
		Results    *FieldList
	}

	// SliceType defines the type for a slice.
	SliceType struct {
		Src   *ast.ArrayType
		DType Type
		Rank  int
	}

	// arrayType defines the type of an array from code.
	arrayType struct {
		Src    *ast.ArrayType
		DTypeF Type
		RankF  ArrayRank
	}

	// PackageRef is a reference to a package.
	// PackageRef is an invalid type in GX.
	PackageRef struct {
		Src  *ast.Ident // Identifier of the package in the source.
		Decl *ImportDecl
	}

	// PackageTypeSelector selects a type on an imported package.
	PackageTypeSelector struct {
		Src     *ast.SelectorExpr
		Package *PackageRef
		Typ     *NamedType
	}

	// BuiltinType is an opaque type maintained by the backend.
	// These are the only atomic types that can have a state.
	BuiltinType struct {
		Impl any
	}

	// TupleType is the type of the result of a function that returns more than one value.
	TupleType struct {
		Types []Type
	}

	// InvalidType is a type assigned by the compiler because of a compiling error.
	InvalidType struct{}

	// UnknownType is a temporary unknown type.
	UnknownType struct{}

	// VoidType is a type for representing an absence of value.
	VoidType struct{}
)

var (
	_ Type      = (*NamedType)(nil)
	_ Type      = (*StructType)(nil)
	_ Type      = (*InterfaceType)(nil)
	_ Type      = (*FuncType)(nil)
	_ Type      = (*SliceType)(nil)
	_ Type      = (*PackageRef)(nil)
	_ Type      = (*PackageTypeSelector)(nil)
	_ Type      = (*TupleType)(nil)
	_ Expr      = (*PackageTypeSelector)(nil)
	_ Type      = (*InvalidType)(nil)
	_ ArrayType = (*atomicType)(nil)
	_ ArrayType = (*arrayType)(nil)
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

func (InvalidType) node() {}

// Kind returns the scalar kind.
func (s InvalidType) Kind() Kind { return InvalidKind }

// Equal returns true if other is the same type.
func (s InvalidType) Equal(Fetcher, Type) (bool, error) { return false, nil }

// AssignableTo reports whether a value of the type can be assigned to another.
// Always returns false.
func (s InvalidType) AssignableTo(Fetcher, Type) (bool, error) { return false, nil }

// ConvertibleTo reports whether a value of the type can be converted to another
// (using static type casting). Always returns false.
func (s InvalidType) ConvertibleTo(Fetcher, Type) (bool, error) { return false, nil }

// String representation of the type.
func (s InvalidType) String() string { return s.Kind().String() }

func (UnknownType) node() {}

// Kind returns the unknown kind.
func (s UnknownType) Kind() Kind { return UnknownKind }

// Equal returns true if other is the same type.
func (s UnknownType) Equal(Fetcher, Type) (bool, error) { return false, nil }

// AssignableTo reports whether a value of the type can be assigned to another.
// Always returns false.
func (s UnknownType) AssignableTo(Fetcher, Type) (bool, error) { return false, nil }

// ConvertibleTo reports whether a value of the type can be converted to another
// (using static type casting). Always returns false.
func (s UnknownType) ConvertibleTo(Fetcher, Type) (bool, error) { return false, nil }

// String representation of the type.
func (s UnknownType) String() string { return s.Kind().String() }

func (TupleType) node() {}

// Kind returns the scalar kind.
func (s TupleType) Kind() Kind { return TupleKind }

func (s TupleType) apply(fetcher Fetcher, target Type, f func(Type, Fetcher, Type) (bool, error)) (bool, error) {
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
func (s TupleType) Equal(fetcher Fetcher, target Type) (bool, error) {
	return s.apply(fetcher, target, (Type).Equal)
}

// AssignableTo reports whether a value of the type can be assigned to another.
func (s TupleType) AssignableTo(fetcher Fetcher, target Type) (bool, error) {
	return s.apply(fetcher, target, (Type).AssignableTo)
}

// ConvertibleTo reports whether a value of the type can be converted to another
// (using static type casting). Always returns false.
func (s TupleType) ConvertibleTo(fetcher Fetcher, target Type) (bool, error) {
	return s.apply(fetcher, target, (Type).ConvertibleTo)
}

// String representation of the type.
func (s TupleType) String() string {
	ss := make([]string, len(s.Types))
	for i, typ := range s.Types {
		ss[i] = typ.String()
	}
	return fmt.Sprintf("(%s)", strings.Join(ss, ","))
}

func (VoidType) node() {}

// Kind returns the void kind.
func (s VoidType) Kind() Kind { return VoidKind }

// Equal returns true if other is the same type.
func (s VoidType) Equal(Fetcher, Type) (bool, error) { return false, nil }

// AssignableTo reports whether a value of the type can be assigned to another.
// Always returns false.
func (s VoidType) AssignableTo(Fetcher, Type) (bool, error) { return false, nil }

// ConvertibleTo reports whether a value of the type can be converted to another
// (using static type casting). Always returns false.
func (s VoidType) ConvertibleTo(Fetcher, Type) (bool, error) { return false, nil }

// String representation of the type.
func (s VoidType) String() string { return s.Kind().String() }

func (InterfaceType) node() {}

// Kind returns the interface kind.
func (s InterfaceType) Kind() Kind { return InterfaceKind }

// Equal returns true if other is the same type.
func (s InterfaceType) Equal(Fetcher, Type) (bool, error) { return false, nil }

// AssignableTo reports whether a value of the type can be assigned to another.
// Always returns false.
func (s InterfaceType) AssignableTo(Fetcher, Type) (bool, error) { return false, nil }

// ConvertibleTo reports whether a value of the type can be converted to another
// (using static type casting). Always returns false.
func (s InterfaceType) ConvertibleTo(Fetcher, Type) (bool, error) { return false, nil }

// String representation of the type.
func (s InterfaceType) String() string { return s.Kind().String() }

// IsValid returns true if the type is valid.
func IsValid(tp Type) bool {
	return tp.Kind() != InvalidKind
}

func (BuiltinType) node() {}

// Kind returns the scalar kind.
func (s BuiltinType) Kind() Kind { return BuiltinKind }

// Equal returns true if other is the same type.
func (s BuiltinType) Equal(_ Fetcher, other Type) (bool, error) {
	otherT, ok := other.(BuiltinType)
	if !ok {
		return false, nil
	}
	return s.Impl == otherT.Impl, nil
}

// AssignableTo reports if the type can be assigned to other.
func (s BuiltinType) AssignableTo(fetcher Fetcher, target Type) (bool, error) {
	return s.Equal(fetcher, target)
}

// ConvertibleTo reports whether a value of the type can be converted to another
// (using static type casting).
func (s BuiltinType) ConvertibleTo(fetcher Fetcher, target Type) (bool, error) {
	return s.Equal(fetcher, target)
}

// String representation of the type.
func (s BuiltinType) String() string {
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
	targetT, ok := target.(ArrayType)
	if !ok {
		return false, nil
	}
	if targetT.Rank().IsAtomic() {
		if s.Knd == NumberIntKind && IsInteger(target.Kind()) {
			return true, nil
		}
		if s.Knd == NumberFloatKind && IsInteger(target.Kind()) {
			return false, nil
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
	targetT, ok := target.(ArrayType)
	if !ok {
		return false, nil
	}
	if targetT.Rank().IsAtomic() {
		return SupportOperators(target.Kind()), nil
	}
	return s.Rank().Equal(fetcher, targetT.Rank())
}

// Source returns the source code defining the type.
// Always returns nil.
func (s *atomicType) Source() ast.Node {
	return s.ArrayType()
}

// DataType returns the type of the element.
func (s *atomicType) DataType() Type {
	return s
}

// ArrayType returns the source code defining the type.
// Always returns nil.
func (s *atomicType) ArrayType() *ast.ArrayType {
	return nil
}

// Zero returns a zero expression of the same type.
func (s *atomicType) Zero() Expr {
	return &NumberCastExpr{
		X:   &NumberInt{Src: &ast.BasicLit{Value: "0"}},
		Typ: s,
	}
}

// String representation of the type.
func (s *atomicType) String() string {
	return s.Kind().String()
}

func (*NamedType) node() {}

// MethodByName returns a method given its name, or nil if not method has that name.
func (s *NamedType) MethodByName(name string) Func {
	for _, method := range s.Methods {
		if method.Name() == name {
			return method
		}
	}
	return nil
}

// Kind of the underlying type.
func (s *NamedType) Kind() Kind { return s.Underlying.Kind() }

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
	return s.Underlying.Equal(fetcher, otherT.Underlying)
}

// AssignableTo reports if the type can be assigned to other.
func (s *NamedType) AssignableTo(fetcher Fetcher, target Type) (bool, error) {
	return s.Equal(fetcher, target)
}

// ConvertibleTo reports whether a value of the type can be converted to another
// (using static type casting).
func (s *NamedType) ConvertibleTo(fetcher Fetcher, target Type) (bool, error) {
	return s.Underlying.Equal(fetcher, target)
}

// FullName returns the full name of the type, that is the full package path and the type name.
func (s *NamedType) FullName() string {
	return s.File.Package.Path + "." + s.Name()
}

// Source returns the node in the AST tree.
func (s *NamedType) Source() ast.Node { return s.Src }

// Name of the type.
func (s *NamedType) Name() string {
	return s.NameT
}

// String representation of the type.
func (s *NamedType) String() string {
	return s.Package().Name.Name + "." + s.Name()
}

// Package returns the package to which the type belongs to.
func (s *NamedType) Package() *Package {
	return s.File.Package
}

// Underlying returns the underlying type of a type.
func Underlying(tp Type) Type {
	named, ok := tp.(*NamedType)
	if !ok {
		return tp
	}
	return Underlying(named.Underlying)
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
	if s == otherT {
		return true, nil
	}
	recvOk, err := s.Receiver.Equal(fetcher, otherT.Receiver)
	if err != nil {
		return false, err
	}
	paramsOk, err := s.Params.Type().Equal(fetcher, otherT.Params.Type())
	if err != nil {
		return false, err
	}
	resultsOk, err := s.Results.Type().Equal(fetcher, otherT.Results.Type())
	if err != nil {
		return false, err
	}
	return recvOk && paramsOk && resultsOk, nil
}

// AssignableTo reports if the type can be assigned to other.
func (s *FuncType) AssignableTo(fetcher Fetcher, other Type) (bool, error) {
	return s.Equal(fetcher, other)
}

// ConvertibleTo reports whether a value of the type can be converted to another
// (using static type casting).
func (s *FuncType) ConvertibleTo(fetcher Fetcher, target Type) (bool, error) {
	return s.Equal(fetcher, target)
}

// Source returns the node in the AST tree.
func (s *FuncType) Source() ast.Node { return s.Src }

// String representation of the type.
func (s *FuncType) String() string {
	var b strings.Builder
	b.WriteString("func")
	if s.Receiver != nil {
		b.WriteString(" (")
		b.WriteString(s.Receiver.Name())
		b.WriteString(") ")
	}
	b.WriteString(s.Params.TupleType().String())
	b.WriteRune(' ')
	b.WriteString(s.Results.Type().String())
	return b.String()
}

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
	return otherT.DType.Equal(fetcher, s.DType)
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

// Source returns the node in the AST tree.
func (s *SliceType) Source() ast.Node { return s.Src }

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
		dtype = UnknownType{}
	}
	if rank == nil {
		rank = &GenericRank{}
	}
	return &arrayType{
		Src:    src,
		DTypeF: dtype,
		RankF:  rank,
	}
}

// Kind returns the tensor kind.
func (s *arrayType) Kind() Kind {
	if s.RankF == nil {
		return InvalidKind
	}
	if s.RankF.NumAxes() == 0 {
		return s.DTypeF.Kind()
	}
	return ArrayKind
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

// Source returns the node in the AST tree.
func (s *arrayType) Source() ast.Node { return s.ArrayType() }

// ArrayType returns the source code defining the type.
func (s *arrayType) ArrayType() *ast.ArrayType {
	return s.Src
}

// Zero returns a zero literal of this type
func (s *arrayType) Zero() Expr {
	return &ArrayLitExpr{Typ: s}
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

func (s *PackageRef) node()          {}
func (s *PackageRef) referenceNode() {}

// Source returns the node in the AST tree.
func (s *PackageRef) Source() ast.Node { return s.Src }

// Equal returns true if other is the same type.
func (s *PackageRef) Equal(_ Fetcher, other Type) (bool, error) {
	otherT, ok := other.(*PackageRef)
	if !ok {
		return false, nil
	}
	return s.Decl.Package.Path == otherT.Decl.Package.Path, nil
}

// AssignableTo reports if the type can be assigned to other.
func (s *PackageRef) AssignableTo(Fetcher, Type) (bool, error) {
	return false, nil
}

// ConvertibleTo reports whether a value of the type can be converted to another
// (using static type casting).
func (s *PackageRef) ConvertibleTo(fetcher Fetcher, target Type) (bool, error) {
	return false, nil
}

// Kind always return invalid. A package is not a valid GX type.
func (s *PackageRef) Kind() Kind { return InvalidKind }

// Expr returns the expression in the source code.
func (s *PackageRef) Expr() ast.Expr { return s.Src }

// String representation of the type.
func (s *PackageRef) String() string {
	return s.Decl.Package.String()
}

func (s *PackageTypeSelector) node()          {}
func (s *PackageTypeSelector) referenceNode() {}

// Source returns the node in the AST tree.
func (s *PackageTypeSelector) Source() ast.Node { return s.Src }

// Kind returns the kind of the type being referenced.
func (s *PackageTypeSelector) Kind() Kind { return s.Typ.Kind() }

// Equal returns true if other is the same type.
func (s *PackageTypeSelector) Equal(fetcher Fetcher, other Type) (bool, error) {
	return s.Typ.Equal(fetcher, other)
}

// AssignableTo reports if the type can be assigned to other.
func (s *PackageTypeSelector) AssignableTo(fetcher Fetcher, other Type) (bool, error) {
	return s.Typ.AssignableTo(fetcher, other)
}

// ConvertibleTo reports whether a value of the type can be converted to another
// (using static type casting).
func (s *PackageTypeSelector) ConvertibleTo(fetcher Fetcher, target Type) (bool, error) {
	return false, nil
}

// Expr returns the expression in the source code.
func (s *PackageTypeSelector) Expr() ast.Expr { return s.Src }

// Type returns the type of the expression.
func (s *PackageTypeSelector) Type() Type { return s }

// String representation of the type.
func (s *PackageTypeSelector) String() string {
	return s.Typ.String()
}

// IsStatic return true if the type is static, that is if the instance of the type
// can be evaluated when the graph is being constructed.
func IsStatic(tp Type) bool {
	switch tp.Kind() {
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

		Consts []*ConstDecl
		Funcs  []Func
		Types  []*NamedType
		Vars   []*VarDecl
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
		StaticValue

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
		Type() Type
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
		Package *Package
		FName   string
		FType   *FuncType
		Impl    FuncImpl
	}

	// FuncMetaImpl is a builtin opaque function to produce an IR.
	FuncMetaImpl any

	// MacroImpl is a macro function, that is a function which builds
	// a new function.
	MacroImpl func(fetcher Fetcher, target *FuncDecl, args []Expr) error

	// FuncMeta is a function that takes a set of IR nodes as an input
	// and returns a new set of IR nodes (or a compiler error).
	// An example of such function is math/graph.Grad.
	// The implementation is opaque to GX.
	FuncMeta struct {
		FFile *File
		Src   *ast.FuncDecl
		FType *FuncType
		Impl  FuncMetaImpl
	}

	// FuncLit is a function literal.
	FuncLit struct {
		FFile *File
		Src   *ast.FuncLit
		FType *FuncType
		Body  *BlockStmt
	}

	// ImportDecl imports a package.
	ImportDecl struct {
		Src     *ast.ImportSpec
		Package *Package
	}

	// VarDecl declares a static variable and, optionally, a default value.
	VarDecl struct {
		FFile *File
		Src   *ast.ValueSpec
		VName *ast.Ident
		TypeV Type
	}

	// ConstExpr is a name,expr constant pair.
	ConstExpr struct {
		Decl *ConstDecl

		VName *ast.Ident
		Value Expr
	}

	// ConstDecl declares a package constant.
	ConstDecl struct {
		FFile *File
		Src   *ast.ValueSpec
		Type  Type

		Exprs []*ConstExpr
	}
)

var (
	_ Node       = (*Package)(nil)
	_ Node       = (*ImportDecl)(nil)
	_ Node       = (*ConstDecl)(nil)
	_ StaticExpr = (*ConstExpr)(nil)
	_ Func       = (*FuncBuiltin)(nil)
	_ Func       = (*FuncDecl)(nil)
	_ Func       = (*FuncLit)(nil)
	_ Func       = (*FuncMeta)(nil)
	_ Expr       = (*FuncLit)(nil)
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
func (pkg *Package) TypeByName(name string) *NamedType {
	for _, tp := range pkg.Types {
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
func (pkg *Package) ExportedFuncs() iter.Seq[Func] {
	return func(yield func(Func) bool) {
		for _, fn := range pkg.Funcs {
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
func (pkg *Package) FindFunc(name string) Func {
	for _, fn := range pkg.Funcs {
		if fn.Name() == name {
			return fn
		}
	}
	return nil
}

// ExportedTypes returns the list of exported types of a package.
func (pkg *Package) ExportedTypes() []*NamedType {
	types := []*NamedType{}
	for _, tp := range pkg.Types {
		if !IsExported(tp.Name()) {
			continue
		}
		types = append(types, tp)
	}
	return types
}

// ExportedConsts returns the list of exported constants.
func (pkg *Package) ExportedConsts() []*ConstExpr {
	exprs := []*ConstExpr{}
	for _, csts := range pkg.Consts {
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

func (*FuncDecl) node()        {}
func (*FuncDecl) staticValue() {}

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

// FullyQualifiedName returns a fully qualified function name, that is
// the full package path and the name of the function.
func (s *FuncDecl) FullyQualifiedName() string {
	return s.FFile.Package.FullName() + "." + s.Name()
}

// Doc returns associated documentation or nil.
func (s *FuncDecl) Doc() *ast.CommentGroup {
	return s.Src.Doc
}

// ReceiverField returns a field representing the receiver of the function, or nil if the function has no receiver.
func (s *FuncDecl) ReceiverField() *Field {
	if s.FType.Receiver == nil {
		return nil
	}
	groupSrc := s.Src.Recv.List[0]
	field := &Field{
		Group: &FieldGroup{
			Src:  groupSrc,
			Type: s.FType.Receiver,
		},
		Name: groupSrc.Names[0],
	}
	field.Group.Fields = []*Field{field}
	return field
}

// Type returns the type of the function.
func (s *FuncDecl) Type() Type {
	return s.FType
}

// FuncType returns the concrete type of the function.
func (s *FuncDecl) FuncType() *FuncType {
	return s.FType
}

// File declaring the function.
func (s *FuncDecl) File() *File {
	return s.FFile
}

func (*FuncBuiltin) node()        {}
func (*FuncBuiltin) staticValue() {}

// Name of the function. Returns an empty string if the function is anonymous.
func (s *FuncBuiltin) Name() string {
	return s.FName
}

// Doc returns associated documentation or nil.
func (s *FuncBuiltin) Doc() *ast.CommentGroup {
	return nil
}

// File declaring the builtin function.
// By definition, the function has not been declared in a file.
// Consequently, the file returned has no source file.
func (s *FuncBuiltin) File() *File {
	return &File{Package: s.Package}
}

// Type returns the type of the function.
func (s *FuncBuiltin) Type() Type {
	return s.FType
}

// FuncType returns the concrete type of the function.
func (s *FuncBuiltin) FuncType() *FuncType {
	return s.FType
}

func (*FuncLit) node()        {}
func (*FuncLit) staticValue() {}

// Name of the function. Returns an empty string since literals are always anonymous.
func (s *FuncLit) Name() string {
	return ""
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

// Source returns the node in the AST tree.
func (s *FuncLit) Source() ast.Node { return s.Src }

// Expr returns the expression in the source code.
func (s *FuncLit) Expr() ast.Expr { return s.Src }

// String representation of the literal.
func (s *FuncLit) String() string {
	return "func {...}"
}

func (*FuncMeta) node()        {}
func (*FuncMeta) staticValue() {}

// Name of the function.
func (s *FuncMeta) Name() string {
	return s.Src.Name.Name
}

// Doc returns associated documentation or nil.
func (s *FuncMeta) Doc() *ast.CommentGroup {
	return nil
}

// File declaring the function literal.
func (s *FuncMeta) File() *File {
	return s.FFile
}

// Type returns the type of the function.
func (s *FuncMeta) Type() Type {
	return s.FType
}

// FuncType returns the concrete type of the function.
func (s *FuncMeta) FuncType() *FuncType {
	return s.FType
}

// Source returns the node in the AST tree.
func (s *FuncMeta) Source() ast.Node { return s.Src }

// String representation of the literal.
func (s *FuncMeta) String() string {
	return "func {...}"
}

func (*ImportDecl) node() {}

// Name returns the identifier naming the package in the import.
func (s *ImportDecl) Name() *ast.Ident {
	return s.Src.Name
}

// Source returns the node in the AST tree.
func (s *ImportDecl) Source() ast.Node {
	return s.Src
}

func (*VarDecl) node() {}

// Source returns the node in the AST tree.
func (s *VarDecl) Source() ast.Node {
	return s.Src
}

// Name of the variable.
func (s *VarDecl) Name() *ast.Ident {
	return s.VName
}

// Type of the variable.
func (s *VarDecl) Type() Type {
	return s.TypeV
}

func (*ConstDecl) node() {}

func (*ConstExpr) node()        {}
func (*ConstExpr) staticValue() {}

// Source returns the node in the AST tree.
func (cst *ConstExpr) Source() ast.Node {
	return cst.VName
}

// Type returns the type of an expression.
func (cst *ConstExpr) Type() Type {
	return cst.Decl.Type
}

// Expr returns the source of the expression.
func (cst *ConstExpr) Expr() ast.Expr {
	return cst.Value.Expr()
}

// String representation of the constant.
func (cst *ConstExpr) String() string {
	return fmt.Sprintf("const(%s)", cst.VName)
}

// ----------------------------------------------------------------------------
// Array axes specification

type (
	// ArrayRank of an array.
	ArrayRank interface {
		Node
		nodeRank()

		// IsGeneric returns true if some axes are unknown.
		IsGeneric() bool

		// IsAtomic returns true if the rank has no axes.
		IsAtomic() bool

		// NumAxes returns the number of axes or -1 if unknown.
		NumAxes() int

		// Equal returns true if two ranks are equal.
		Equal(Fetcher, ArrayRank) (bool, error)

		// AssignableTo returns true if an array can be assigned to another.
		AssignableTo(Fetcher, ArrayRank) (bool, error)

		// ConvertibleTo returns true if other can be converted to the receiver.
		ConvertibleTo(Fetcher, ArrayRank) (bool, error)

		// String representation of the rank.
		String() string
	}

	// ResolvedRank is an ArrayRank that can or has been resolved by the compiler.
	ResolvedRank interface {
		ArrayRank
		// Resolved returns the resolved rank or nil if the rank has not been resolved.
		Resolved() *Rank
	}

	// Rank with a known number of axes.
	// The length of some axes may still be unknown.
	Rank struct {
		Src  *ast.ArrayType
		Axes []AxisLength
	}

	// GenericRank is a rank determined at compile time.
	GenericRank struct {
		Src *ast.ArrayType
		Rnk *Rank

		// Name is an optional rank parameter identifier.
		Name *ast.Ident
	}

	// AnyRank is a rank that is determined at runtime and not checked by the compiler.
	AnyRank struct {
		Src *ast.ArrayType
	}

	// AxisLength specification of an array.
	AxisLength interface {
		alen()
		Expr

		// Equal returns true if two axis lengths have been resolved and are equal.
		// Returns an error if one of the axis has not been resolved.
		Equal(Fetcher, AxisLength) (bool, error)

		// AssignableTo returns true if this axis length can be assigned to another.
		AssignableTo(Fetcher, AxisLength) (bool, error)

		// IsGeneric returns true if the axis length is unknown.
		IsGeneric() bool

		// String representation of the axis length.
		String() string
	}

	// AxisEllipsis is an array axis specified as "_".
	AxisEllipsis struct {
		Src *ast.Ident
		X   AxisLength
	}

	// AxisExpr is an array axis specified using an expression.
	AxisExpr struct {
		// Source of the axis expression.
		// May be different from the source of the expression, for example
		// the expression is formed from a function call.
		Src ast.Expr
		// X computes the size of the axis.
		X Expr
	}
)

var (
	_ ResolvedRank = (*Rank)(nil)
	_ ResolvedRank = (*GenericRank)(nil)
	_ ArrayRank    = (*AnyRank)(nil)
	_ AxisLength   = (*AxisExpr)(nil)
	_ AxisLength   = (*AxisEllipsis)(nil)
)

func (*Rank) node()     {}
func (*Rank) nodeRank() {}

// NewRank returns a new rank from a slice of axis lengths.
func NewRank(axlens []int) *Rank {
	axes := make([]AxisLength, len(axlens))
	for i, al := range axlens {
		axes[i] = &AxisExpr{
			X: &AtomicValueT[Int]{
				Val: Int(al),
				Typ: IntLenType(),
			},
		}
	}
	return &Rank{Axes: axes}
}

// Source returns the source node defining the rank.
func (r *Rank) Source() ast.Node { return r.Src }

// NumAxes returns the number of axes.
func (r *Rank) NumAxes() int { return len(r.Axes) }

// Equal returns true if other has the same number of axes and each axis has the same length.
func (r *Rank) Equal(fetcher Fetcher, other ArrayRank) (bool, error) {
	switch otherT := other.(type) {
	case *AnyRank:
		return false, nil
	case *GenericRank:
		if otherT.Rnk == nil {
			return true, nil
		}
		return r.equalRank(fetcher, otherT.Rnk)
	case *Rank:
		return r.equalRank(fetcher, otherT)
	default:
		return false, errors.Errorf("rank type %T not supported", otherT)
	}
}

func (r *Rank) equalRank(fetcher Fetcher, other *Rank) (bool, error) {
	// Check the number of axes.
	if len(r.Axes) != len(other.Axes) {
		return false, nil
	}
	// Check each axis.
	for i, dimX := range r.Axes {
		dimEq, err := dimX.Equal(fetcher, other.Axes[i])
		if err != nil {
			return false, err
		}
		if !dimEq {
			return false, nil
		}
	}
	return true, nil
}

func (r *Rank) assignableTo(fetcher Fetcher, dst *Rank) (bool, error) {
	if len(r.Axes) != len(dst.Axes) {
		return false, nil
	}
	for i, dimThis := range r.Axes {
		ok, err := dimThis.AssignableTo(fetcher, dst.Axes[i])
		if !ok || err != nil {
			return ok, err
		}
	}
	return true, nil
}

// IsGeneric returns true if the length of some axes are unknown.
func (r *Rank) IsGeneric() bool {
	for _, dim := range r.Axes {
		if dim.IsGeneric() {
			return true
		}
	}
	return false
}

// AssignableTo returns true if other can be assigned to the receiver.
func (r *Rank) AssignableTo(fetcher Fetcher, dst ArrayRank) (bool, error) {
	switch dstT := dst.(type) {
	case *AnyRank:
		return true, nil
	case *GenericRank:
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

// ConvertibleTo returns true if other can be converted to the receiver.
func (r *Rank) ConvertibleTo(fetcher Fetcher, dst ArrayRank) (bool, error) {
	var dstR *Rank
	switch dstT := dst.(type) {
	case *AnyRank:
		return true, nil
	case *GenericRank:
		if dstT.Rnk == nil {
			return true, nil
		}
		dstR = dstT.Rnk
	case *Rank:
		dstR = dstT
	default:
		return false, errors.Errorf("rank type %T not supported", dstT)
	}
	thisSize, err := r.Size(fetcher)
	if err != nil {
		return false, err
	}
	otherSize, err := dstR.Size(fetcher)
	if err != nil {
		return false, err
	}
	return areConvertible(fetcher, thisSize, otherSize)
}

// Size is the total number of elements across all axes.
func (r *Rank) Size(fetcher Fetcher) (Expr, error) {
	var expr Expr = &AtomicValueT[Int]{
		Src: r.Src,
		Val: 1,
		Typ: IntLenSliceType(),
	}
	for _, dim := range r.Axes {
		expr = &BinaryExpr{
			Src: &ast.BinaryExpr{
				Op: token.MUL,
				X:  expr.Expr(),
				Y:  expr.Expr(),
			},
			X:   expr,
			Y:   dim,
			Typ: IntLenSliceType(),
		}
	}
	return expr, nil
}

// IsAtomic returns true if the rank is equals to zero
// (that is it has no axes).
func (r *Rank) IsAtomic() bool {
	return r.NumAxes() == 0
}

// AxisLengths of the rank.
func (r *Rank) AxisLengths() []AxisLength {
	return r.Axes
}

// Resolved returns itself.
func (r *Rank) Resolved() *Rank {
	return r
}

// String returns a string representation of the rank.
func (r *Rank) String() string {
	if r == nil {
		return ""
	}
	bld := strings.Builder{}
	for _, dim := range r.Axes {
		bld.WriteString("[" + dim.String() + "]")
	}
	return bld.String()
}

func (*AnyRank) node()     {}
func (*AnyRank) nodeRank() {}

// NumAxes returns the number of axes.
func (r *AnyRank) NumAxes() int { return -1 }

// Equal returns true if other has the same rank and dimensions.
func (r *AnyRank) Equal(fetcher Fetcher, other ArrayRank) (bool, error) {
	switch otherT := other.(type) {
	case *AnyRank:
		return true, nil
	case *GenericRank:
		return false, errors.Errorf("cannot compare an unknown rank to a generic rank")
	case *Rank:
		return false, nil
	default:
		return false, errors.Errorf("rank type %T not supported", otherT)
	}
}

// AssignableTo returns true if other can be assigned to the receiver.
func (r *AnyRank) AssignableTo(fetcher Fetcher, dst ArrayRank) (bool, error) {
	switch dstT := dst.(type) {
	case *AnyRank:
		return true, nil
	case *GenericRank:
		return false, errors.Errorf("cannot assign an unknown rank to a generic rank")
	case *Rank:
		return false, nil
	default:
		return false, errors.Errorf("rank type %T not supported", dstT)
	}
}

// ConvertibleTo returns true if other can be converted to the receiver.
func (r *AnyRank) ConvertibleTo(fetcher Fetcher, dst ArrayRank) (bool, error) {
	switch dstT := dst.(type) {
	case *AnyRank:
		return true, nil
	case *GenericRank:
		return false, errors.Errorf("cannot convert a unknown rank to a generic rank")
	case *Rank:
		return true, nil
	default:
		return false, errors.Errorf("rank type %T not supported", dstT)
	}
}

// IsAtomic returns true if the rank has no axes.
func (r *AnyRank) IsAtomic() bool {
	return false
}

// IsGeneric returns true if the dimension is unknown.
func (r *AnyRank) IsGeneric() bool {
	return false
}

// String returns a string representation of the rank.
func (r *AnyRank) String() string {
	return "[any]"
}

func (*GenericRank) node()     {}
func (*GenericRank) nodeRank() {}

// NumAxes returns the number of axes.
func (r *GenericRank) NumAxes() int {
	if r.Rnk == nil {
		return -1
	}
	return r.Rnk.NumAxes()
}

// Equal returns true if other has the same rank and dimensions.
func (r *GenericRank) Equal(fetcher Fetcher, other ArrayRank) (bool, error) {
	if r.Rnk != nil {
		return r.Rnk.Equal(fetcher, other)
	}
	switch otherT := other.(type) {
	case *AnyRank:
		return false, nil
	case *GenericRank:
		return true, nil
	case *Rank:
		return true, nil
	default:
		return false, errors.Errorf("rank type %T not supported", otherT)
	}
}

// AssignableTo returns true if other can be assigned to the receiver.
func (r *GenericRank) AssignableTo(fetcher Fetcher, dst ArrayRank) (bool, error) {
	if r.Rnk != nil {
		return r.Rnk.AssignableTo(fetcher, dst)
	}
	return true, nil
}

// ConvertibleTo returns true if other can be converted to the receiver.
func (r *GenericRank) ConvertibleTo(fetcher Fetcher, dst ArrayRank) (bool, error) {
	if r.Rnk != nil {
		return r.Rnk.ConvertibleTo(fetcher, dst)
	}
	return true, nil
}

// IsAtomic returns true if the rank has no axes.
func (r *GenericRank) IsAtomic() bool {
	return false
}

// IsGeneric returns true if the dimension is unknown.
func (r *GenericRank) IsGeneric() bool {
	if r.Rnk == nil {
		return true
	}
	return r.Rnk.IsGeneric()
}

// Resolved returns itself.
func (r *GenericRank) Resolved() *Rank {
	return r.Rnk
}

// String returns a string representation of the rank.
func (r *GenericRank) String() string {
	if r.Rnk != nil {
		return fmt.Sprintf("%v", r.Rnk)
	}
	name := "..."
	if r.Name != nil {
		name = r.Name.String()
	}
	return fmt.Sprintf("[%s]", name)
}

func (*AxisEllipsis) alen() {}
func (*AxisEllipsis) node() {}

// Source returns the source expression specifying the dimension.
func (dm *AxisEllipsis) Source() ast.Node { return dm.Src }

// Type of the expression.
func (dm *AxisEllipsis) Type() Type { return IntLenType() }

// Expr returns how to compute the dimension as an expression.
func (dm *AxisEllipsis) Expr() ast.Expr { return dm.Src }

// IsGeneric returns true if the dimension is unknown.
func (dm *AxisEllipsis) IsGeneric() bool {
	if dm.X == nil {
		return true
	}
	return dm.X.IsGeneric()
}

// Equal returns true if other has the axis length.
func (dm *AxisEllipsis) Equal(fetcher Fetcher, other AxisLength) (bool, error) {
	switch otherT := other.(type) {
	case *AxisExpr:
		if dm.X == nil {
			return false, errors.Errorf("unresolved axis length")
		}
		return dm.X.Equal(fetcher, otherT)
	case *AxisEllipsis:
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
func (dm *AxisEllipsis) AssignableTo(fetcher Fetcher, dst AxisLength) (bool, error) {
	if dm.X != nil {
		return dm.X.AssignableTo(fetcher, dst)
	}
	return true, nil
}

// String representation of the dimension.
func (dm *AxisEllipsis) String() string {
	const s = "_"
	if dm.X == nil {
		return s
	}
	return dm.X.String()
}

func (*AxisExpr) alen() {}
func (*AxisExpr) node() {}

// Source returns the source expression specifying the dimension.
func (dm *AxisExpr) Source() ast.Node { return dm.X.Source() }

// IsGeneric returns true if the dimension is unknown.
func (dm *AxisExpr) IsGeneric() bool {
	return false
}

// Eval returns an evaluation of the dimension.
func (dm *AxisExpr) Eval(fetcher Fetcher) (int, []*ValueRef, error) {
	val, refs, err := Eval[Int](fetcher, dm.X)
	return int(val), refs, err
}

// AssignableTo returns true if a dimension can be assigned to another.
func (dm *AxisExpr) AssignableTo(fetcher Fetcher, dst AxisLength) (bool, error) {
	switch dstT := dst.(type) {
	case *AxisExpr:
		return dm.Equal(fetcher, dst)
	case *AxisEllipsis:
		if dstT.X == nil {
			return true, nil
		}
		return dm.Equal(fetcher, dstT.X)
	default:
		return false, errors.Errorf("assigning an axis to an axis type %T not supported", dstT)
	}
}

// Equal returns true if other has the axis length.
func (dm *AxisExpr) Equal(fetcher Fetcher, other AxisLength) (bool, error) {
	switch otherT := other.(type) {
	case *AxisExpr:
		return areEqual(fetcher, dm.X, otherT.X)
	case *AxisEllipsis:
		if otherT.X == nil {
			return false, errors.Errorf("cannot compare with an unresolved axis length")
		}
		return dm.Equal(fetcher, otherT.X)
	default:
		return false, errors.Errorf("cannot compare with axis type %T: not supported", otherT)
	}
}

// Type of the expression.
func (dm *AxisExpr) Type() Type { return IntLenType() }

// Expr returns how to compute the dimension as an expression.
func (dm *AxisExpr) Expr() ast.Expr { return dm.X.Expr() }

// String representation of the dimension.
func (dm *AxisExpr) String() string { return dm.X.String() }

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
		Type   Type
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
	fields := []*Field{}
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

// Len returns the total number of fields in the list.
func (s *FieldList) Len() int {
	r := 0
	for _, grp := range s.List {
		r += grp.NumFields()
	}
	return r
}

// Type returns the fields as a type.
func (s *FieldList) Type() Type {
	fields := s.Fields()
	switch len(fields) {
	case 0:
		return VoidType{}
	case 1:
		return fields[0].Type()
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
func (s *FieldGroup) Source() ast.Node { return s.Src }

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
	return s.Name
}

// Type returns the type of the field.
func (s *Field) Type() Type {
	return s.Group.Type
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
		Expr() ast.Expr
		Type() Type
		String() string
	}

	// StaticValue is a generic expression representing a value defined at compile time.
	StaticValue interface {
		Node
		staticValue()
		Type() Type
	}

	// StaticExpr is a static value represented by an expression
	// (as opposed to a function declaration for example).
	StaticExpr interface {
		Expr
		StaticValue
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

	// AtomicT is an expression computing a scalar value.
	AtomicT[T dtype.GoDataType] interface {
		StaticValue
	}

	// NumberFloat is a float number for which the type has not been inferred yet.
	NumberFloat struct {
		Src *ast.BasicLit
	}

	// NumberInt is an integer number for which the type has not been inferred yet.
	NumberInt struct {
		Src *ast.BasicLit
	}

	// NumberCastExpr casts a number to a given type.
	NumberCastExpr struct {
		X   Expr
		Typ Type
	}

	// StaticAtom is an atom (e.g. a scalar) represented as an expression to be evaluated at compile time.
	StaticAtom struct {
		X Expr
	}

	// AtomicValueT is an array literal.
	AtomicValueT[T dtype.GoDataType] struct {
		Src ast.Expr
		Val T
		Typ Type
	}

	// ArrayLitExpr is an array literal.
	ArrayLitExpr struct {
		Src  *ast.CompositeLit
		Typ  ArrayType
		Vals []Expr
	}

	// SliceExpr is a slice literal.
	SliceExpr struct {
		Src  ast.Expr
		Typ  Type
		Vals []Expr
	}

	// FieldLit assigns a value to a field in a structure literal.
	FieldLit struct {
		Field *Field
		X     Expr
	}

	// StructLitExpr is a structure literal.
	StructLitExpr struct {
		Src  *ast.CompositeLit
		Elts []FieldLit
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

	// CallExpr is an expression calling a function.
	CallExpr struct {
		Src      *ast.CallExpr
		Args     []Expr
		Func     Expr
		FuncType *FuncType
	}

	// CallResultExpr represents the ith result of a function call as an expression.
	CallResultExpr struct {
		Index int
		Call  *CallExpr
	}

	// CastExpr casts a type to another.
	CastExpr struct {
		Src *ast.CallExpr
		Typ Type

		X Expr
	}

	// ValueRef is a reference to a value.
	ValueRef struct {
		Src *ast.Ident
		Typ Type
	}

	// FieldSelectorExpr selects a field on a structure.
	FieldSelectorExpr struct {
		Src    *ast.SelectorExpr
		X      Expr
		Struct *StructType
		Typ    Type
		Field  *Field
	}

	// IndexExpr selects an index on a indexable type.
	IndexExpr struct {
		Src   *ast.IndexExpr
		X     Expr
		Index Expr
		Typ   Type
	}

	// EinsumExpr represents an einsum expression.
	EinsumExpr struct {
		Src        ast.Expr
		X, Y       Expr
		BatchAxes  [2][]int
		ReduceAxes [2][]int
		Typ        Type
	}

	// MethodSelectorExpr selects a method on a type defined within the package.
	MethodSelectorExpr struct {
		Src   *ast.SelectorExpr
		X     Expr
		Named *NamedType
		Typ   Type
		Func  Func
	}

	// PackageFuncSelectorExpr selects a method on a package.
	PackageFuncSelectorExpr struct {
		Src     *ast.SelectorExpr
		Package *PackageRef
		Typ     Type
		Func    Func
	}

	// PackageConstSelectorExpr selects a const on a package.
	PackageConstSelectorExpr struct {
		Src     *ast.SelectorExpr
		Package *PackageRef
		Const   *ConstExpr

		X   Expr
		Typ Type
	}

	// TypeExpr is a reference to a type.
	TypeExpr struct {
		Src ast.Expr
		Typ Type
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
		Value() RuntimeValue
	}

	// RuntimeValueExprT is a runtime expression specialised for a value type.
	// The Src field can be nil if the value was not computed from an expression
	// in the source code (e.g. a value given as a static value programmatically).
	RuntimeValueExprT[T RuntimeValue] struct {
		Src ast.Expr
		Typ Type
		Val T
	}
)

var (
	_ Number           = (*NumberFloat)(nil)
	_ Number           = (*NumberInt)(nil)
	_ StaticValue      = (*NumberCastExpr)(nil)
	_ StaticValue      = (*StaticAtom)(nil)
	_ StaticValue      = (*AtomicValueT[int32])(nil)
	_ StaticValue      = (*StringLiteral)(nil)
	_ Expr             = (*ArrayLitExpr)(nil)
	_ Expr             = (*SliceExpr)(nil)
	_ Expr             = (*StructLitExpr)(nil)
	_ Node             = (*FieldLit)(nil)
	_ Expr             = (*UnaryExpr)(nil)
	_ Expr             = (*ParenExpr)(nil)
	_ Expr             = (*BinaryExpr)(nil)
	_ Expr             = (*CallExpr)(nil)
	_ Expr             = (*CallResultExpr)(nil)
	_ Expr             = (*CastExpr)(nil)
	_ Expr             = (*ValueRef)(nil)
	_ Expr             = (*FieldSelectorExpr)(nil)
	_ Expr             = (*IndexExpr)(nil)
	_ Expr             = (*EinsumExpr)(nil)
	_ Expr             = (*MethodSelectorExpr)(nil)
	_ Expr             = (*PackageFuncSelectorExpr)(nil)
	_ Expr             = (*PackageConstSelectorExpr)(nil)
	_ Expr             = (*TypeExpr)(nil)
	_ RuntimeValueExpr = (*RuntimeValueExprT[RuntimeValue])(nil)
)

func (s *NumberFloat) node()       {}
func (s *NumberFloat) numberExpr() {}

// Zero returns a zero float number.
func (s *NumberFloat) Zero() Expr {
	return &NumberFloat{Src: &ast.BasicLit{Value: "0.0"}}
}

// Expr returns the AST expression.
func (s *NumberFloat) Expr() ast.Expr { return s.Src }

// Source returns the node in the AST tree.
func (s *NumberFloat) Source() ast.Node { return s.Src }

// Type returns the type returned by the function call.
func (s *NumberFloat) Type() Type { return NumberFloatType() }

// DefaultType returns the default type of the number.
func (s NumberFloat) DefaultType() Type { return TypeFromKind(Float64Kind) }

// String representation of the number.
func (s *NumberFloat) String() string { return fmt.Sprintf("%s", s.Src.Value) }

func (s *NumberInt) node()       {}
func (s *NumberInt) numberExpr() {}

// Zero returns a zero float number.
func (s *NumberInt) Zero() Expr {
	return &NumberInt{Src: &ast.BasicLit{Value: "0"}}
}

// Expr returns the AST expression.
func (s *NumberInt) Expr() ast.Expr { return s.Src }

// Source returns the node in the AST tree.
func (s *NumberInt) Source() ast.Node { return s.Src }

// Type returns the type returned by the function call.
func (s *NumberInt) Type() Type { return NumberIntType() }

// DefaultType returns the default type of the number.
func (s NumberInt) DefaultType() Type { return TypeFromKind(Int64Kind) }

// String representation of the number.
func (s *NumberInt) String() string { return fmt.Sprintf("%s", s.Src.Value) }

func (s *NumberCastExpr) node()        {}
func (s *NumberCastExpr) staticValue() {}

// Expr returns the AST expression.
func (s *NumberCastExpr) Expr() ast.Expr { return s.X.Expr() }

// Source returns the node in the AST tree.
func (s *NumberCastExpr) Source() ast.Node { return s.X.Source() }

// Type returns the type returned by the function call.
func (s *NumberCastExpr) Type() Type { return s.Typ }

// String representation of the number.
func (s *NumberCastExpr) String() string { return s.X.String() }

func (s *StringLiteral) node()        {}
func (s *StringLiteral) staticValue() {}

// Expr returns the AST expression.
func (s *StringLiteral) Expr() ast.Expr { return s.Src }

// Source returns the node in the AST tree.
func (s *StringLiteral) Source() ast.Node { return s.Src }

// Type returns the type returned by the function call.
func (s *StringLiteral) Type() Type { return StringType() }

// String representation of the number.
func (s *StringLiteral) String() string { return s.Src.Value }

func (s *AtomicValueT[T]) node()        {}
func (s *AtomicValueT[T]) staticValue() {}

// Expr returns the AST expression.
func (s *AtomicValueT[T]) Expr() ast.Expr { return s.Src }

// Source returns the node in the AST tree.
func (s *AtomicValueT[T]) Source() ast.Node { return s.Src }

// Type returns the type returned by the function call.
func (s *AtomicValueT[T]) Type() Type { return s.Typ }

// String representation.
func (s *AtomicValueT[T]) String() string { return fmt.Sprint(s.Val) }

func (s *StaticAtom) node()        {}
func (s *StaticAtom) staticValue() {}

// Expr returns the AST expression.
func (s *StaticAtom) Expr() ast.Expr { return s.X.Expr() }

// Type returns the type returned by the function call.
func (s *StaticAtom) Type() Type { return s.X.Type() }

// Source returns the node in the AST tree.
func (s *StaticAtom) Source() ast.Node { return s.X.Source() }

// String representation.
func (s *StaticAtom) String() string { return s.X.String() }

func (s *ArrayLitExpr) node()      {}
func (s *ArrayLitExpr) arrayExpr() {}

// Source returns the node in the AST tree.
func (s *ArrayLitExpr) Source() ast.Node {
	return s.Src
}

// Type returns the type returned by the function call.
func (s *ArrayLitExpr) Type() Type { return s.Typ }

// Expr returns the AST expression.
func (s *ArrayLitExpr) Expr() ast.Expr { return s.Src }

// Values returns the expressions defining the values of the array.
func (s *ArrayLitExpr) Values() []Expr { return s.Vals }

// NewFromValues returns a new literal of the same type from a slice of values.
func (s *ArrayLitExpr) NewFromValues(vals []Expr) *ArrayLitExpr {
	return &ArrayLitExpr{Typ: s.Typ, Vals: vals}
}

func (s *StructLitExpr) node() {}

// Source returns the node in the AST tree.
func (s *StructLitExpr) Source() ast.Node { return s.Src }

// Type returns the type returned by the function call.
func (s *StructLitExpr) Type() Type { return s.Typ }

// Expr returns the AST expression.
func (s *StructLitExpr) Expr() ast.Expr { return s.Src }

// String representation.
func (s *StructLitExpr) String() string { return fmt.Sprintf("%T", s) }

func (s *SliceExpr) node() {}

// Source returns the node in the AST tree.
func (s *SliceExpr) Source() ast.Node { return s.Src }

// Type returns the type returned by the function call.
func (s *SliceExpr) Type() Type { return s.Typ }

// Expr returns the AST expression.
func (s *SliceExpr) Expr() ast.Expr { return s.Src }

// String representation.
func (s *SliceExpr) String() string { return fmt.Sprintf("%T", s) }

func (s *UnaryExpr) node() {}

// Source returns the node in the AST tree.
func (s *UnaryExpr) Source() ast.Node { return s.Src }

// Type returns the type returned by the expression.
func (s *UnaryExpr) Type() Type { return s.X.Type() }

// Expr returns the expression in the source code.
func (s *UnaryExpr) Expr() ast.Expr { return s.Src }

// String representation.
func (s *UnaryExpr) String() string { return s.Src.Op.String() + s.X.String() }

func (s *ParenExpr) node() {}

// Source returns the node in the AST tree.
func (s *ParenExpr) Source() ast.Node { return s.Src }

// Type returns the type returned by the expression.
func (s *ParenExpr) Type() Type { return s.X.Type() }

// Expr returns the expression in the source code.
func (s *ParenExpr) Expr() ast.Expr { return s.Src }

// String representation.
func (s *ParenExpr) String() string { return "(" + s.X.String() + ")" }

func (s *BinaryExpr) node() {}

// Source returns the node in the AST tree.
func (s *BinaryExpr) Source() ast.Node { return s.Src }

// Type returns the type returned by the expression.
// Use CallExpr.Func.Type to get the type of the function being called.
func (s *BinaryExpr) Type() Type { return s.Typ }

// Expr returns the expression in the source code.
func (s *BinaryExpr) Expr() ast.Expr { return s.Src }

// String representation.
func (s *BinaryExpr) String() string {
	return s.X.String() + s.Src.Op.String() + s.Y.String()
}

func (s *CallExpr) node() {}

// Source returns the node in the AST tree.
func (s *CallExpr) Source() ast.Node { return s.Src }

// Type returns the type returned by the function call.
// Use CallExpr.Func.Type to get the type of the function being called.
func (s *CallExpr) Type() Type { return s.FuncType.Results.Type() }

// Expr returns the expression in the source code.
func (s *CallExpr) Expr() ast.Expr { return s.Src }

// ExprFromResult returns an expression pointing to the ith result of a function call.
func (s *CallExpr) ExprFromResult(i int) *CallResultExpr {
	return &CallResultExpr{
		Index: i,
		Call:  s,
	}
}

// String representation.
func (s *CallExpr) String() string { return fmt.Sprintf("%T", s) }

func (s *CallResultExpr) node() {}

// Source returns the node in the AST tree.
func (s *CallResultExpr) Source() ast.Node { return s.Call.Source() }

// Type returns the type returned by the function call.
// Use CallExpr.Func.Type to get the type of the function being called.
func (s *CallResultExpr) Type() Type { return s.Call.FuncType.Results.Fields()[s.Index].Type() }

// Expr returns the expression in the source code.
func (s *CallResultExpr) Expr() ast.Expr { return s.Call.Expr() }

// String representation.
func (s *CallResultExpr) String() string { return fmt.Sprintf("%T", s) }

func (s *CastExpr) node() {}

// Source returns the node in the AST tree.
func (s *CastExpr) Source() ast.Node { return s.Src }

// Type returns the target type of the cast.
func (s *CastExpr) Type() Type { return s.Typ }

// Expr returns the expression in the source code.
func (s *CastExpr) Expr() ast.Expr { return s.Src }

// String representation.
func (s *CastExpr) String() string { return fmt.Sprintf("%T", s) }

func (s *ValueRef) node() {}

// Source returns the node in the AST tree.
func (s *ValueRef) Source() ast.Node { return s.Src }

// Type returns the type returned by the function call.
// Use CallExpr.Func.Type to get the type of the function being called.
func (s *ValueRef) Type() Type { return s.Typ }

// Expr returns the expression in the source code.
func (s *ValueRef) Expr() ast.Expr { return s.Src }

// String representation.
func (s *ValueRef) String() string { return s.Src.Name }

func (FieldLit) node() {}

func (s *FieldSelectorExpr) node() {}

// Source returns the node in the AST tree.
func (s *FieldSelectorExpr) Source() ast.Node { return s.Src }

// Type returns the type returned by the function call.
// Use CallExpr.Func.Type to get the type of the function being called.
func (s *FieldSelectorExpr) Type() Type { return s.Typ }

// Expr returns the AST expression.
func (s *FieldSelectorExpr) Expr() ast.Expr { return s.Src }

// String representation.
func (s *FieldSelectorExpr) String() string { return fmt.Sprintf("%T", s) }

func (s *IndexExpr) node() {}

// Source returns the node in the AST tree.
func (s *IndexExpr) Source() ast.Node { return s.Src }

// Type returns the type of the indexed element.
func (s *IndexExpr) Type() Type { return s.Typ }

// Expr returns the AST expression.
func (s *IndexExpr) Expr() ast.Expr { return s.Src }

// String representation.
func (s *IndexExpr) String() string { return fmt.Sprintf("%T", s) }

func (s *EinsumExpr) node() {}

// Source returns the node in the AST tree.
func (s *EinsumExpr) Source() ast.Node { return s.Src }

// Type returns the type returned by the einsum.
func (s *EinsumExpr) Type() Type { return s.Typ }

// Expr returns the expression in the source code.
func (s *EinsumExpr) Expr() ast.Expr { return s.Src }

// String representation.
func (s *EinsumExpr) String() string { return fmt.Sprintf("%T", s) }

func (s *MethodSelectorExpr) node() {}

// Source returns the node in the AST tree.
func (s *MethodSelectorExpr) Source() ast.Node { return s.Src }

// Type returns the type returned by the function call.
// Use CallExpr.Func.Type to get the type of the function being called.
func (s *MethodSelectorExpr) Type() Type { return s.Typ }

// Expr returns the AST expression.
func (s *MethodSelectorExpr) Expr() ast.Expr { return s.Src }

// String representation.
func (s *MethodSelectorExpr) String() string { return fmt.Sprintf("%T", s) }

func (s *PackageFuncSelectorExpr) node() {}

// Source returns the node in the AST tree.
func (s *PackageFuncSelectorExpr) Source() ast.Node { return s.Src }

// Type returns the type returned by the function call.
// Use CallExpr.Func.Type to get the type of the function being called.
func (s *PackageFuncSelectorExpr) Type() Type { return s.Typ }

// Expr returns the AST expression.
func (s *PackageFuncSelectorExpr) Expr() ast.Expr { return s.Src }

// String representation.
func (s *PackageFuncSelectorExpr) String() string { return fmt.Sprintf("%T", s) }

func (s *PackageConstSelectorExpr) node() {}

// Source returns the node in the AST tree.
func (s *PackageConstSelectorExpr) Source() ast.Node { return s.Src }

// Type of the constant.
func (s *PackageConstSelectorExpr) Type() Type { return s.Typ }

// Expr returns the AST expression.
func (s *PackageConstSelectorExpr) Expr() ast.Expr { return s.Src }

// String representation.
func (s *PackageConstSelectorExpr) String() string { return fmt.Sprintf("%T", s) }

func (s *TypeExpr) node() {}

// Source returns the node in the AST tree.
func (s *TypeExpr) Source() ast.Node { return s.Src }

// Expr returns the expression in the source code.
func (s *TypeExpr) Expr() ast.Expr { return s.Src }

// Type returns the type referenced by the expression.
func (s *TypeExpr) Type() Type { return s.Typ }

// String representation of the type.
func (s *TypeExpr) String() string {
	return s.Typ.String()
}

func (s *RuntimeValueExprT[T]) node() {}

// Source returns the node in the AST tree.
func (s *RuntimeValueExprT[T]) Source() ast.Node { return s.Src }

// Expr returns the expression in the source code.
func (s *RuntimeValueExprT[T]) Expr() ast.Expr { return s.Src }

// Type returns the type referenced by the expression.
func (s *RuntimeValueExprT[T]) Type() Type { return s.Typ }

// Value returns the runtime value stored in the expression.
func (s *RuntimeValueExprT[T]) Value() RuntimeValue { return s.Val }

// String representation of the type.
func (s *RuntimeValueExprT[T]) String() string {
	return fmt.Sprintf("runtime(%T)", s.Val)
}

// ----------------------------------------------------------------------------
// Storage.

type (
	// LocalVarAssign is a local variable to which values can be assigned to.
	LocalVarAssign struct {
		Src   *ast.Ident
		TypeF Type
	}
	// StructFieldAssign is a field in a structure.
	StructFieldAssign struct {
		Src       *ast.SelectorExpr
		X         Expr
		TypeF     Type
		FieldName string
	}
)

var (
	_ Assignable = (*LocalVarAssign)(nil)
	_ Assignable = (*StructFieldAssign)(nil)
)

func (*LocalVarAssign) assignableNode() {}
func (*LocalVarAssign) node()           {}

// Source returns the node in the AST tree.
func (s *LocalVarAssign) Source() ast.Node { return s.Src }

// Expr returns the expression in the AST tree.
func (s *LocalVarAssign) Expr() ast.Expr { return s.Src }

// Type of the destination of the assignment.
func (s *LocalVarAssign) Type() Type { return s.TypeF }

// String representation of the assignment.
func (s *LocalVarAssign) String() string { return fmt.Sprint(s.Src) }

func (*StructFieldAssign) assignableNode() {}
func (*StructFieldAssign) node()           {}

// Source returns the node in the AST tree.
func (s *StructFieldAssign) Source() ast.Node { return s.Src }

// Expr returns the expression in the AST tree.
func (s *StructFieldAssign) Expr() ast.Expr { return s.Src }

// Type of the destination of the assignment.
func (s *StructFieldAssign) Type() Type { return s.TypeF }

// String representation of the assignment.
func (s *StructFieldAssign) String() string { return fmt.Sprint(s.Src) }

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

	// Assignable is a variable or a field to which a value can be assigned to.
	Assignable interface {
		Expr
		// assignableNode marks a structure as Assignable.
		assignableNode()
	}

	// AssignCallStmt assigns the results of a function call returning more than one value to variables.
	AssignCallStmt struct {
		Src  *ast.AssignStmt
		Call *CallExpr
		List []Assignable
	}

	// AssignExpr assigns an expression to a Assignable node.
	AssignExpr struct {
		Expr Expr
		Dest Assignable
	}

	// AssignExprStmt assigns the results of expressions (possibly functions returning one value) to variables.
	AssignExprStmt struct {
		Src  *ast.AssignStmt
		List []AssignExpr
	}

	// RangeStmt is a range statement in for loops.
	RangeStmt struct {
		Src        *ast.RangeStmt
		Key, Value Assignable
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
)

var (
	_ Stmt = (*BlockStmt)(nil)
	_ Stmt = (*ReturnStmt)(nil)
	_ Stmt = (*AssignCallStmt)(nil)
	_ Stmt = (*AssignExprStmt)(nil)
	_ Stmt = (*RangeStmt)(nil)
	_ Stmt = (*IfStmt)(nil)
	_ Stmt = (*ExprStmt)(nil)
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
	return TupleType{Types: types}
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
// A slice returns an nil rank.
func Shape(tp Type) (ArrayRank, Type) {
	switch tpT := tp.(type) {
	case ArrayType:
		return tpT.Rank(), tpT.DataType()
	case *NamedType:
		return Shape(tpT.Underlying)
	case *SliceType:
		return nil, tpT.DType
	default:
		return nil, InvalidType{}
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
