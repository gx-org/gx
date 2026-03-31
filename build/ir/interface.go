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

	"github.com/pkg/errors"
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
func (s *Interface) Equal(fetcher Fetcher, target Type) (bool, CompEvalError, error) {
	targetSet, ok := target.(*Interface)
	if !ok {
		return false, nil, nil
	}
	if s == targetSet {
		return true, nil, nil
	}
	if len(s.types) != len(targetSet.types) {
		return false, nil, nil
	}
	for i, typ := range s.types {
		ok, cpErr, err := typ.Equal(fetcher, targetSet.types[i])
		if err != nil {
			err = fmt.Errorf("cannot compare type set %s to %s: %w", s.ReferString(fetcher.File()), targetSet.ReferString(fetcher.File()), err)
			return false, nil, err
		}
		if cpErr != nil || !ok {
			return false, cpErr, nil
		}
	}
	return true, nil, nil
}

// AssignableTo reports whether a value of the type can be assigned to another.
func (s *Interface) AssignableTo(fetcher Fetcher, target Type) (bool, CompEvalError, error) {
	if targetSet, ok := target.(*Interface); ok {
		return targetSet.assignableFrom(fetcher, s)
	}
	for _, typ := range s.types {
		ok, cpErr, err := typ.AssignableTo(fetcher, target)
		if !ok || cpErr != nil || err != nil {
			return false, cpErr, err
		}
	}
	return len(s.types) > 0, nil, nil
}

func (s *Interface) checkTypesAssignableFrom(fetcher Fetcher, source Type) (bool, CompEvalError, error) {
	if len(s.types) == 0 {
		return true, nil, nil
	}

	if sourceSet, ok := source.(*Interface); ok {
		return s.containsTypes(fetcher, sourceSet)
	}
	for _, typ := range s.types {
		ok, cpErr, err := source.AssignableTo(fetcher, typ)
		if cpErr != nil || err != nil {
			return false, cpErr, err
		}
		if ok {
			return true, nil, nil
		}
	}
	return false, nil, nil
}

type toString func(file *File) string

func checkHasMethod(fetcher Fetcher, ifaceName toString, source *NamedType, imeth *IMethod) (bool, CompEvalError, error) {
	methName := imeth.Name()
	meth := MethodByName(source, methName)
	if meth == nil {
		from := fetcher.File()
		return false, CompEvalError(errors.Errorf("%s does not implement %s (missing method %s)", ifaceName(from), source.ReferString(from), methName)), nil
	}
	eq, cpErr, err := meth.FuncType().equalParamsResults(fetcher, imeth.FType)
	if cpErr != nil || err != nil {
		return false, cpErr, err
	}
	if !eq {
		from := fetcher.File()
		return false, CompEvalError(errors.Errorf("%s does not implement %s (wrong type for method %s)\n\thave %s\n\twant %s", ifaceName(from), source.ReferString(from), methName, meth.FuncType().sourceSignature(from, imeth.NameDef(), true), imeth.SourceSignature(from))), nil
	}
	return true, nil, nil
}

func (s *Interface) checkImplement(fetcher Fetcher, source Type, ifaceName toString) (bool, CompEvalError, error) {
	named, ok := source.(*NamedType)
	if !ok {
		return len(s.methods) == 0, nil, nil
	}
	for _, method := range s.methods {
		ok, cpErr, err := checkHasMethod(fetcher, ifaceName, named, method)
		if !ok || cpErr != nil || err != nil {
			return ok, cpErr, err
		}
	}
	return true, nil, nil
}

func (s *Interface) assignableFromWithName(fetcher Fetcher, source Type, name toString) (bool, CompEvalError, error) {
	typeOk, cpErr, err := s.checkTypesAssignableFrom(fetcher, source)
	if cpErr != nil || err != nil {
		return false, cpErr, err
	}
	methodOk, cpErr, err := s.checkImplement(fetcher, source, name)
	if cpErr != nil || err != nil {
		return false, cpErr, err
	}
	return typeOk && methodOk, nil, nil
}

func (s *Interface) assignableFrom(fetcher Fetcher, source Type) (bool, CompEvalError, error) {
	return s.assignableFromWithName(fetcher, source, s.ReferString)
}

// ConvertibleTo reports whether a value of the type can be converted to another
// (using static type casting).
func (s *Interface) ConvertibleTo(fetcher Fetcher, target Type) (bool, CompEvalError, error) {
	if _, ok := target.(*Interface); ok {
		return s.Equal(fetcher, target)
	}
	for _, typ := range s.types {
		ok, cpErr, err := typ.ConvertibleTo(fetcher, target)
		if cpErr != nil || err != nil {
			return false, cpErr, err
		}
		if ok {
			return true, nil, nil
		}
	}
	return false, nil, nil
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
func (s *Interface) Specialise(Specialiser) (Type, CompEvalError, error) {
	return s, nil, nil
}

// UnifyWith recursively unifies a type parameters with types.
func (*Interface) UnifyWith(unifier Unifier, typ Type) bool {
	return true
}

var anyType = &Interface{}

// AnyType returns the type for the keyword any.
func AnyType() Type {
	return anyType
}
