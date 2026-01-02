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

	"github.com/gx-org/gx/build/ir/annotations"
)

type (
	// Annotator of a function: add meta-data to a declared function.
	Annotator struct {
		MetaCore

		Annotate AnnotatorImpl
	}

	// MetaCore is the core shared amongst meta functions:
	// macros and annotations.
	MetaCore struct {
		FFile *File
		Src   *ast.FuncDecl
		FType *FuncType
	}
)

func (*MetaCore) node()         {}
func (*MetaCore) staticValue()  {}
func (*MetaCore) storage()      {}
func (*MetaCore) storageValue() {}
func (*MetaCore) pkgFunc()      {}

// FullName returns the fully qualified name of the macro.
func (s *MetaCore) FullName() string {
	return fullName(s)
}

// Name of the function.
func (s *MetaCore) Name() string {
	return s.Src.Name.Name
}

// NameDef is the name definition of the function.
func (s *MetaCore) NameDef() *ast.Ident {
	return s.Src.Name
}

// Value returns a reference to the function.
func (s *MetaCore) Value(x Expr) AssignableExpr {
	return &FuncValExpr{
		X: x,
		F: s,
		T: s.FuncType(),
	}
}

// Doc returns associated documentation or nil.
func (s *MetaCore) Doc() *ast.CommentGroup {
	return nil
}

// File declaring the function literal.
func (s *MetaCore) File() *File {
	return s.FFile
}

// FuncType returns the concrete type of the function.
func (s *MetaCore) FuncType() *FuncType {
	return s.FType
}

// Type returns the type of the function.
func (s *MetaCore) Type() Type {
	return s.FuncType()
}

// Node returns the node in the AST tree.
func (s *MetaCore) Node() ast.Node {
	return s.Src
}

// Same returns true if the other storage is this storage.
func (s *MetaCore) Same(o Storage) bool {
	return Storage(s) == o
}

// Annotations returns the annotations attached to the function.
func (s *MetaCore) Annotations() *annotations.Annotations {
	return &annotations.Annotations{}
}

// String representation of the literal.
func (s *MetaCore) String() string {
	return fmt.Sprintf("metafunc %s", s.Name())
}

// New returns a new function given a source, a file, and a type.
func (s *MetaCore) New() PkgFunc {
	n := *s
	return &n
}

// ShortString returns the name of the function.
func (s *MetaCore) ShortString() string {
	return s.File().Package.Name.Name + "." + s.Name()
}
