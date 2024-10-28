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
	"sort"
	"strings"

	"github.com/pkg/errors"
	"golang.org/x/exp/maps"
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

	// TypeRank is a type with a rank.
	TypeRank interface {
		Type
		Rank() ArrayRank
	}

	// AtomicType defines a scalar of a given kind.
	AtomicType struct {
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

		Receiver *NamedType
		Params   *FieldList
		Results  *FieldList
	}

	// SliceType defines the type for a slice.
	SliceType struct {
		Src   *ast.ArrayType
		DType Type
		Rank  int
	}

	// ArrayType defines the type of an array from code.
	ArrayType struct {
		Src   *ast.ArrayType
		DType Type
		RankF ArrayRank
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
	_ Type     = (*NamedType)(nil)
	_ Type     = (*StructType)(nil)
	_ Type     = (*InterfaceType)(nil)
	_ Type     = (*FuncType)(nil)
	_ Type     = (*SliceType)(nil)
	_ Type     = (*PackageRef)(nil)
	_ Type     = (*PackageTypeSelector)(nil)
	_ Type     = (*TupleType)(nil)
	_ Expr     = (*PackageTypeSelector)(nil)
	_ Type     = (*InvalidType)(nil)
	_ TypeRank = (*AtomicType)(nil)
	_ TypeRank = (*ArrayType)(nil)
)

// DefaultFloatType is the default type used for a scalar.
var DefaultFloatType = &AtomicType{Knd: Float32Kind}

var (
	// DefaultIntKind is the default kind for integer.
	DefaultIntKind = Int64Kind

	// DefaultIntType is the default type used for an integer.
	DefaultIntType = &AtomicType{Knd: DefaultIntKind}
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

// Equal returns true if other is the same type.
func (s TupleType) Equal(Fetcher, Type) (bool, error) { return false, nil }

// AssignableTo reports whether a value of the type can be assigned to another.
// Always returns false.
func (s TupleType) AssignableTo(Fetcher, Type) (bool, error) { return false, nil }

// ConvertibleTo reports whether a value of the type can be converted to another
// (using static type casting). Always returns false.
func (s TupleType) ConvertibleTo(Fetcher, Type) (bool, error) { return false, nil }

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

func (*AtomicType) node() {}

// Kind returns the scalar kind.
func (s *AtomicType) Kind() Kind { return s.Knd }

func (s *AtomicType) equalAtomic(other *AtomicType) (bool, error) {
	return s.Knd == other.Knd, nil
}

func (s *AtomicType) equalArray(fetcher Fetcher, other *ArrayType) (bool, error) {
	dtypeEq, err := s.Equal(fetcher, other.DType)
	if !dtypeEq || err != nil {
		return false, err
	}
	return s.Rank().Equal(fetcher, other.Rank())

}

// Equal returns true if other is the same type.
func (s *AtomicType) Equal(fetcher Fetcher, other Type) (bool, error) {
	switch otherT := other.(type) {
	case *AtomicType:
		return s.equalAtomic(otherT)
	case *ArrayType:
		return s.equalArray(fetcher, otherT)
	default:
		return false, nil
	}
}

var scalarRank = &Rank{}

// Rank of the array.
func (s *AtomicType) Rank() ArrayRank { return scalarRank }

// AssignableTo reports if the type can be assigned to other.
func (s *AtomicType) AssignableTo(fetcher Fetcher, target Type) (bool, error) {
	switch targetT := target.(type) {
	case *AtomicType:
		if s.Knd == NumberKind {
			return true, nil
		}
		return s.equalAtomic(targetT)
	case *ArrayType:
		rankOk, err := s.equalArray(fetcher, targetT)
		if !rankOk || err != nil {
			return rankOk, err
		}
		return s.Knd == NumberKind, nil
	default:
		return false, nil
	}
}

// ConvertibleTo reports whether a value of the type can be converted to another
// (using static type casting).
func (s *AtomicType) ConvertibleTo(fetcher Fetcher, target Type) (bool, error) {
	switch targetT := target.(type) {
	case *AtomicType:
		return IsAtomic(target.Kind()), nil
	case *ArrayType:
		return s.Rank().Equal(fetcher, targetT.Rank())
	default:
		return false, nil
	}
}

// String representation of the type.
func (s *AtomicType) String() string {
	return s.Kind().String()
}

func (*NamedType) node() {}

// Kind of the underlying type.
func (s *NamedType) Kind() Kind { return s.Underlying.Kind() }

// Equal returns true if other is the same type.
func (s *NamedType) Equal(fetcher Fetcher, other Type) (bool, error) {
	otherT, ok := other.(*NamedType)
	if !ok {
		return false, nil
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

// Field returns a field given its ID.
func (s *StructType) Field(id int) *Field {
	field := s.Fields.Fields()[id]
	if field.ID != id {
		panic(fmt.Sprintf("field ID (=%d) does not match its position (=%d) in the list", field.ID, id))
	}
	return field
}

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
func (s *FuncType) Equal(_ Fetcher, other Type) (bool, error) {
	otherT, ok := other.(*FuncType)
	if !ok {
		return false, nil
	}
	return s == otherT, nil
}

// AssignableTo reports if the type can be assigned to other.
func (s *FuncType) AssignableTo(fetcher Fetcher, other Type) (bool, error) {
	return s.Equal(fetcher, other)
}

// ConvertibleTo reports whether a value of the type can be converted to another
// (using static type casting).
func (s *FuncType) ConvertibleTo(fetcher Fetcher, target Type) (bool, error) {
	return false, nil
}

// Source returns the node in the AST tree.
func (s *FuncType) Source() ast.Node { return s.Src }

// String representation of the type.
func (s *FuncType) String() string {
	return s.Kind().String()
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

func (*ArrayType) node() {}

// Kind returns the tensor kind.
func (s *ArrayType) Kind() Kind {
	if s.RankF == nil {
		return InvalidKind
	}
	if s.RankF.NumAxes() == 0 {
		return s.DType.Kind()
	}
	return TensorKind
}

// DataType returns the type of the data stored in the array.
func (s *ArrayType) DataType() Type { return s.DType }

// Rank of the array.
func (s *ArrayType) Rank() ArrayRank { return s.RankF }

func (s *ArrayType) assignableToArray(fetcher Fetcher, other *ArrayType) (bool, error) {
	dtypeEq, err := s.DataType().AssignableTo(fetcher, other.DataType())
	if !dtypeEq || err != nil {
		return dtypeEq, err
	}
	return s.Rank().AssignableTo(fetcher, other.Rank())
}

func (s *ArrayType) equalArray(fetcher Fetcher, other *ArrayType) (bool, error) {
	dtypeEq, err := s.DataType().Equal(fetcher, other.DataType())
	if !dtypeEq || err != nil {
		return dtypeEq, err
	}
	return s.Rank().Equal(fetcher, other.Rank())
}

// Equal returns true if other is the same type.
func (s *ArrayType) Equal(fetcher Fetcher, other Type) (bool, error) {
	switch otherT := other.(type) {
	case *AtomicType:
		return otherT.equalArray(fetcher, s)
	case *ArrayType:
		return s.equalArray(fetcher, otherT)
	}
	return false, nil
}

// AssignableTo reports if the type can be assigned to other.
func (s *ArrayType) AssignableTo(fetcher Fetcher, target Type) (bool, error) {
	switch targetT := target.(type) {
	case *AtomicType:
		return targetT.equalArray(fetcher, s)
	case *ArrayType:
		return s.assignableToArray(fetcher, targetT)
	}
	return false, nil
}

// ConvertibleTo reports whether a value of the type can be converted to another
// (using static type casting).
func (s *ArrayType) ConvertibleTo(fetcher Fetcher, target Type) (bool, error) {
	switch targetT := target.(type) {
	case *AtomicType:
		return s.RankF.ConvertibleTo(fetcher, scalarRank)
	case *ArrayType:
		dtypeOk, err := s.DType.ConvertibleTo(fetcher, targetT.DType)
		if !dtypeOk || err != nil {
			return dtypeOk, err
		}
		return s.RankF.ConvertibleTo(fetcher, targetT.RankF)
	}
	return false, nil
}

// Source returns the node in the AST tree.
func (s *ArrayType) Source() ast.Node { return s.Src }

// String representation of the tensor type.
func (s *ArrayType) String() string {
	dtype := "dtype"
	if s.DType != nil {
		dtype = s.DType.String()
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
	case AxisIndexKind, AxisLengthKind:
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
		Node

		// Name of the function (without the package name).
		Name() string

		// Doc returns associated documentation or nil.
		Doc() *ast.CommentGroup

		// File owning the function.
		File() *File

		// Type of the function.
		// If the function is generic, then the type will have
		// been inferred by the compiler from the types passed as args.
		Type() *FuncType
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

	// ImportDecl imports a package.
	ImportDecl struct {
		Src     *ast.ImportSpec
		Package *Package
	}

	// VarDecl declares a static variable and, optionally, a default value.
	VarDecl struct {
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
		Src  *ast.ValueSpec
		Type Type

		Exprs []*ConstExpr
	}
)

var (
	_ Node = (*Package)(nil)
	_ Node = (*ImportDecl)(nil)
	_ Node = (*ConstDecl)(nil)
	_ Func = (*FuncDecl)(nil)
	_ Func = (*FuncBuiltin)(nil)
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
		names := maps.Keys(pkg.Files)
		sort.Strings(names)
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
func (pkg *Package) ExportedFuncs() []Func {
	funcs := []Func{}
	for _, fn := range pkg.Funcs {
		if !IsExported(fn.Name()) {
			continue
		}
		funcs = append(funcs, fn)
	}
	return funcs
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

func (*FuncDecl) node() {}

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
			Type: s.Type().Receiver,
		},
		Name: groupSrc.Names[0],
	}
	field.Group.Fields = []*Field{field}
	return field
}

// Type returns the type of the function.
func (s *FuncDecl) Type() *FuncType {
	return s.FType
}

// File declaring the function.
func (s *FuncDecl) File() *File {
	return s.FFile
}

func (*FuncBuiltin) node() {}

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
func (s *FuncBuiltin) Type() *FuncType {
	return s.FType
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

func (*ConstExpr) node() {}

// Source returns the node in the AST tree.
func (cst *ConstExpr) Source() ast.Node {
	return cst.VName
}

// String representation of the constant.
func (cst *ConstExpr) String() string {
	return fmt.Sprintf("const(%s)", cst.VName)
}

// ----------------------------------------------------------------------------
// Array axes specification

var (
	axisIndexType   = &AtomicType{Knd: AxisIndexKind}
	axisLengthType  = &AtomicType{Knd: AxisLengthKind}
	axisIndicesType = &SliceType{DType: axisIndexType, Rank: 1}
	axisLengthsType = &SliceType{DType: axisLengthType, Rank: 1}
)

// AxisIndexType returns the type of an array axis index.
func AxisIndexType() *AtomicType {
	return axisIndexType
}

// AxisLengthType returns the type of an array axis length.
func AxisLengthType() *AtomicType {
	return axisLengthType
}

// AxisLengthsType returns the type of a slice of axis length.
func AxisLengthsType() *SliceType {
	return axisLengthsType
}

// AxisIndicesType returns the type of a slice of axis indices.
func AxisIndicesType() *SliceType {
	return axisIndicesType
}

type (
	// ArrayRank of an array.
	ArrayRank interface {
		Node
		nodeRank()

		// IsGeneric returns true if some axes are unknown.
		IsGeneric() bool

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
	}

	// AnyRank is a rank that is determined at runtime and not checked by the compiler.
	AnyRank struct {
		Src *ast.ArrayType
	}

	// AxisLength specification of an array.
	AxisLength interface {
		alen()
		SourceNode

		// Expr returns how to compute the axis length as an expression.
		Expr() Expr

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
		X   *AxisExpr
	}

	// AxisExpr is an array axis specified using an expression.
	AxisExpr struct {
		Src ast.Expr
		X   Expr // Expression computing the size.
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
		Typ: AxisLengthsType(),
	}
	for _, dim := range r.Axes {
		dimExpr := dim.Expr()
		if dimExpr == nil {
			return nil, errors.Errorf("missing axis length")
		}
		expr = &BinaryExpr{
			Src: &ast.BinaryExpr{
				Op: token.MUL,
				X:  expr.Expr(),
				Y:  expr.Expr(),
			},
			X:   expr,
			Y:   dimExpr,
			Typ: AxisLengthsType(),
		}
	}
	return expr, nil
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
	resolved := ""
	if r.Rnk != nil {
		resolved = ":" + r.Rnk.String()
	}
	return fmt.Sprintf("[...%s]", resolved)
}

func (*AxisEllipsis) alen() {}
func (*AxisEllipsis) node() {}

// Source returns the source expression specifying the dimension.
func (dm *AxisEllipsis) Source() ast.Node { return dm.Src }

// Expr returns how to compute the dimension as an expression.
func (dm *AxisEllipsis) Expr() Expr {
	if dm.X == nil {
		return nil
	}
	return dm.X.Expr()
}

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
	x := dm.X.String()
	if !strings.HasPrefix(x, s) {
		x = s + "=" + dm.X.String()
	}
	return x
}

func (*AxisExpr) alen() {}
func (*AxisExpr) node() {}

// Source returns the source expression specifying the dimension.
func (dm *AxisExpr) Source() ast.Node { return dm.Src }

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

// Expr returns how to compute the dimension as an expression.
func (dm *AxisExpr) Expr() Expr { return dm.X }

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
		ID   int // ID of the field in the structure.
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

	// Atomic is a generic expression representing an atomic value.
	Atomic interface {
		scalarExpr()
		Expr
	}

	// AtomicT is an expression computing a scalar value.
	AtomicT[T dtype.GoDataType] interface {
		Atomic
		scalarExprT(T)
	}

	// Number is a number for which the type has not been inferred yet.
	Number struct {
		Src *ast.BasicLit
	}

	// AtomicExprT is a scalar represented as an expression to be evaluated.
	AtomicExprT[T dtype.GoDataType] struct {
		X Expr
	}

	// AtomicValueT is an array literal.
	AtomicValueT[T dtype.GoDataType] struct {
		Src ast.Expr
		Val T
		Typ Type
	}

	// ArrayLitExpr represents an array value.
	ArrayLitExpr interface {
		arrayExpr()
		Expr
		Values() []Expr
	}

	// ArrayLitExprT is an array literal.
	ArrayLitExprT[T dtype.GoDataType] struct {
		Src  *ast.CompositeLit
		Typ  *ArrayType
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

	// ParenExpr is an operator with a single argument.
	ParenExpr struct {
		Src *ast.ParenExpr
		X   Expr
		Typ Type
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
	// The Src field can be nil if the value has not computed from an expression
	// in the source code (e.g. a value given as a static value programmatically).
	RuntimeValueExprT[T RuntimeValue] struct {
		Src ast.Expr
		Typ Type
		Val T
	}
)

var (
	_ Atomic           = (*Number)(nil)
	_ AtomicT[int32]   = (*AtomicExprT[int32])(nil)
	_ AtomicT[int32]   = (*AtomicValueT[int32])(nil)
	_ ArrayLitExpr     = (*ArrayLitExprT[int32])(nil)
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

func (s *Number) node()       {}
func (s *Number) scalarExpr() {}

// Expr returns the AST expression.
func (s *Number) Expr() ast.Expr { return s.Src }

// Source returns the node in the AST tree.
func (s *Number) Source() ast.Node { return s.Src }

var numberType = &AtomicType{Knd: NumberKind}

// Type returns the type returned by the function call.
func (s *Number) Type() Type { return numberType }

// String representation of the number.
func (s *Number) String() string { return fmt.Sprintf("number(%s)", s.Src.Value) }

func (s *AtomicValueT[T]) node()         {}
func (s *AtomicValueT[T]) scalarExpr()   {}
func (s *AtomicValueT[T]) scalarExprT(T) {}

// Expr returns the AST expression.
func (s *AtomicValueT[T]) Expr() ast.Expr { return s.Src }

// Source returns the node in the AST tree.
func (s *AtomicValueT[T]) Source() ast.Node { return s.Src }

// Type returns the type returned by the function call.
func (s *AtomicValueT[T]) Type() Type { return s.Typ }

// String representation.
func (s *AtomicValueT[T]) String() string { return fmt.Sprint(s.Val) }

func (s *AtomicExprT[T]) node()         {}
func (s *AtomicExprT[T]) scalarExpr()   {}
func (s *AtomicExprT[T]) scalarExprT(T) {}

// Expr returns the AST expression.
func (s *AtomicExprT[T]) Expr() ast.Expr { return s.X.Expr() }

// Type returns the type returned by the function call.
func (s *AtomicExprT[T]) Type() Type { return s.X.Type() }

// Source returns the node in the AST tree.
func (s *AtomicExprT[T]) Source() ast.Node { return s.X.Source() }

// String representation.
func (s *AtomicExprT[T]) String() string { return s.X.String() }

func (s *ArrayLitExprT[T]) node()      {}
func (s *ArrayLitExprT[T]) arrayExpr() {}

// Source returns the node in the AST tree.
func (s *ArrayLitExprT[T]) Source() ast.Node {
	return s.Src
}

// Type returns the type returned by the function call.
func (s *ArrayLitExprT[T]) Type() Type { return s.Typ }

// Expr returns the AST expression.
func (s *ArrayLitExprT[T]) Expr() ast.Expr { return s.Src }

// Values returns the expressions defining the values of the array.
func (s *ArrayLitExprT[T]) Values() []Expr { return s.Vals }

// String representation.
func (s *ArrayLitExprT[T]) String() string {
	return fmt.Sprintf("%s%v", s.Type().String(), s.Vals)
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
func (s *ParenExpr) Type() Type { return s.Typ }

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
		Src     *ast.SelectorExpr
		X       Expr
		TypeF   Type
		FieldID int
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

// Type of the destination of the assignment.
func (s *LocalVarAssign) Type() Type { return s.TypeF }

func (*StructFieldAssign) assignableNode() {}
func (*StructFieldAssign) node()           {}

// Source returns the node in the AST tree.
func (s *StructFieldAssign) Source() ast.Node { return s.Src }

// Type of the destination of the assignment.
func (s *StructFieldAssign) Type() Type { return s.TypeF }

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
	}

	// ReturnStmt is a return statement in a function.
	ReturnStmt struct {
		Src     *ast.ReturnStmt
		Results []Expr
	}

	// Assignable is a variable or a field to which a value can be assigned to.
	Assignable interface {
		SourceNode
		// assignableNode marks a structure as Assignable.
		assignableNode()

		// Type of the destination of the assignment.
		Type() Type
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
	case *ArrayType:
		return tpT.RankF, tpT.DataType()
	case *AtomicType:
		return tpT.Rank(), tpT
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
