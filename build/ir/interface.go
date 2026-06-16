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

	"github.com/gx-org/gx/build/fmterr"
	"github.com/gx-org/gx/build/ir/annotations"
	"github.com/gx-org/gx/build/ir/irkind"
)

// IMethod is a method defined in an interface.
type IMethod struct {
	// Anns are annotations attached to the method.
	Anns annotations.Annotations
	// IFace is the interface owning the method.
	IFace *Interface

	// Src is the source defining the method.
	Src *ast.Field
	// FType is the function signature of the method.
	FType *FuncType
}

var _ PkgFunc = (*IMethod)(nil)

func (*IMethod) node()         {}
func (*IMethod) staticValue()  {}
func (*IMethod) storage()      {}
func (*IMethod) storageValue() {}
func (*IMethod) pkgFunc()      {}

// Node returns the node in the AST tree.
func (m *IMethod) Node() ast.Node { return m.Src }

// New returns a new function given a source, a file, and a type.
func (m *IMethod) New() PkgFunc {
	n := *m
	return &n
}

// Same returns true if the other storage is this storage.
func (m *IMethod) Same(o Storage) bool {
	return Storage(m) == o
}

// SourceSignature returns a string representing the method.
func (m *IMethod) SourceSignature(from *File) string {
	return m.FType.SourceSignature(from, m.NameDef())
}

// Annotations returns the annotations attached to the function.
func (m *IMethod) Annotations() *annotations.Annotations {
	return &m.Anns
}

// Doc returns associated documentation or nil.
func (m *IMethod) Doc() *ast.CommentGroup {
	return m.Src.Doc
}

// FuncType returns the concrete type of the function.
func (m *IMethod) FuncType() *FuncType {
	return m.FType
}

// Type returns the type of the function.
func (m *IMethod) Type() Type {
	return m.FType
}

// Name of the function.
func (m *IMethod) Name() string {
	return m.NameDef().Name
}

// NameDef returns the name identifier of the method.
func (m *IMethod) NameDef() *ast.Ident {
	return m.Src.Names[0]
}

// File owning the function.
func (m *IMethod) File() *File {
	return m.IFace.File()
}

// DefineString returns the GX source code of the node.
func (m *IMethod) DefineString(from *File) string {
	return m.FType.SourceSignature(from, m.NameDef())
}

// ShortString returns a short string for error messages related to annotations.
func (m *IMethod) ShortString() string {
	return m.ReferString(nil)
}

// ReferString returns the string representation of the node in an error message.
func (m *IMethod) ReferString(from *File) string {
	name := m.Name()
	if name != "" {
		return name
	}
	return m.FuncType().ReferString(from)
}

// Value returns a reference to the function.
func (m *IMethod) Value(x Expr) Expr {
	return NewFuncValExpr(x, m)
}

// Interface represents a set of types.
type Interface struct {
	BaseType[*ast.InterfaceType]
	FFile *File

	types   []Type
	methods []*IMethod
}

var (
	_ Type        = (*Interface)(nil)
	_ ArrayType   = (*Interface)(nil)
	_ assignsFrom = (*Interface)(nil)
	_ TypeMethods = (*Interface)(nil)
)

// NewInterface returns a new type set given a set of types.
func NewInterface(src *ast.InterfaceType, types []Type, meths []*IMethod) *Interface {
	if src == nil {
		src = &ast.InterfaceType{}
	}
	iface := &Interface{
		BaseType: BaseType[*ast.InterfaceType]{Src: src},
		types:    types,
		methods:  meths,
	}
	for _, meth := range meths {
		meth.IFace = iface
	}
	return iface
}

func (*Interface) node() {}

// Rank of the array.
func (*Interface) Rank() ArrayRank { return scalarRank }

// DataType returns the type of the element.
func (s *Interface) DataType() Type {
	return s
}

// ArrayType returns the source code defining the type.
// Always returns nil.
func (s *Interface) ArrayType() ast.Expr {
	return s.BaseType.Src
}

// Kind returns the scalar kind.
func (s *Interface) Kind() irkind.Kind { return irkind.Interface }

// Equal returns true if other is the exact same type set.
func (s *Interface) Equal(tpcmp TypeCmp, target Type) (bool, error) {
	targetSet, ok := target.(*Interface)
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
		ok, err := typ.Equal(tpcmp, targetSet.types[i])
		if err != nil {
			err = fmt.Errorf("cannot compare type set %s to %s: %w", s.ReferString(tpcmp.File()), targetSet.ReferString(tpcmp.File()), err)
			return false, err
		}
		if !ok {
			return false, nil
		}
	}
	return true, nil
}

// AssignableTo reports whether a value of the type can be assigned to another.
func (s *Interface) AssignableTo(tpcmp TypeCmp, target Type) (bool, error) {
	if targetSet, ok := target.(*Interface); ok {
		return targetSet.assignableFrom(tpcmp, s)
	}
	for _, typ := range s.types {
		ok, err := typ.AssignableTo(tpcmp, target)
		if !ok || err != nil {
			return false, err
		}
	}
	return len(s.types) > 0, nil
}

func (s *Interface) checkTypesAssignableFrom(tpcmp TypeCmp, source Type) (bool, error) {
	if len(s.types) == 0 {
		return true, nil
	}

	if sourceSet, ok := source.(*Interface); ok {
		return s.containsTypes(tpcmp, sourceSet)
	}
	for _, typ := range s.types {
		ok, err := source.AssignableTo(tpcmp, typ)
		if err != nil {
			return false, err
		}
		if ok {
			return true, nil
		}
	}
	return false, nil
}

// Instantiate the interface.
func (s *Interface) Instantiate(Fetcher, Specialiser) (Type, bool) {
	return s, true
}

type toString func(file *File) string

func checkHasMethod(tpcmp TypeCmp, ifaceName toString, source *NamedType, imeth *IMethod) (bool, error) {
	methName := imeth.Name()
	meth := MethodByName(source, methName)
	if meth == nil {
		from := tpcmp.File()
		return false, CompileErrorF("%s does not implement %s (missing method %s)", ifaceName(from), source.ReferString(from), methName)
	}
	eq, err := meth.FuncType().equalParamsResults(tpcmp, imeth.FType)
	if err != nil {
		return false, err
	}
	if !eq {
		from := tpcmp.File()
		return false, CompileErrorF("%s does not implement %s (wrong type for method %s)\n\thave %s\n\twant %s", ifaceName(from), source.ReferString(from), methName, meth.FuncType().sourceSignature(from, imeth.NameDef().Name, true), imeth.SourceSignature(from))
	}
	return true, nil
}

func (s *Interface) checkImplement(tpcmp TypeCmp, source Type, ifaceName toString) (bool, error) {
	named, ok := source.(*NamedType)
	if !ok {
		return len(s.methods) == 0, nil
	}
	for _, method := range s.methods {
		ok, err := checkHasMethod(tpcmp, ifaceName, named, method)
		if !ok || err != nil {
			return ok, err
		}
	}
	return true, nil
}

func (s *Interface) assignableFromWithName(tpcmp TypeCmp, source Type, name toString) (bool, error) {
	typeOk, err := s.checkTypesAssignableFrom(tpcmp, source)
	if err != nil {
		return false, err
	}
	methodOk, err := s.checkImplement(tpcmp, source, name)
	if err != nil {
		return false, err
	}
	return typeOk && methodOk, nil
}

func (s *Interface) assignableFrom(tpcmp TypeCmp, source Type) (bool, error) {
	return s.assignableFromWithName(tpcmp, source, s.ReferString)
}

// ConvertibleTo reports whether a value of the type can be converted to another
// (using static type casting).
func (s *Interface) ConvertibleTo(tpcmp TypeCmp, target Type) (bool, error) {
	if _, ok := target.(*Interface); ok {
		return s.Equal(tpcmp, target)
	}
	for _, typ := range s.types {
		ok, err := typ.ConvertibleTo(tpcmp, target)
		if err != nil {
			return false, err
		}
		if ok {
			return true, nil
		}
	}
	return false, nil
}

// Methods returns the list of methods available for the interface.
func (s *Interface) Methods() []PkgFunc {
	methods := make([]PkgFunc, len(s.methods))
	for i, meth := range s.methods {
		methods[i] = meth
	}
	return methods
}

// File owning the function.
func (s *Interface) File() *File {
	return s.FFile
}

// Specialise a type to a given target.
func (s *Interface) Specialise(Specialiser) (Type, bool) {
	return s, true
}

// UnifyWith recursively unifies a type parameters with types.
func (*Interface) UnifyWith(unifier Unifier, typ Type) bool {
	return true
}

// IndexForVarArgs returns a type specific to a given index in varargs.
func (s *Interface) IndexForVarArgs(fmterr.ErrAppender, int) (Type, bool) {
	return s, true
}

var anyType = &Interface{}

// AnyType returns the type for the keyword any.
func AnyType() Type {
	return anyType
}
