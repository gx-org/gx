// Copyright 2026 Google LLC
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
	"path"
	"strings"

	"github.com/gx-org/gx/src/gx/ast"
)

// ImportDecl imports a package.
type ImportDecl struct {
	Src     *ast.ImportSpec
	Path    string
	Package *Package
}

var _ StorageWithValue = (*ImportDecl)(nil)

func (*ImportDecl) node()         {}
func (*ImportDecl) storage()      {}
func (*ImportDecl) storageValue() {}

// Unroll returns an error because import declarations do not support unrolling.
func (s *ImportDecl) Unroll(ev Fetcher, urlr Unroller) (ast.Expr, bool) {
	return nil, ev.Err().AppendInternalf(s.Src, "%T does not support unrolling", s)
}

// Node returns the node in the AST tree.
func (s *ImportDecl) Node() ast.Node {
	return s.Src
}

// PackageTypeSeparator is the separator in a package to define the type of a package.
const PackageTypeSeparator = ":"

// Name used to reference the package.
func (s *ImportDecl) Name() string {
	if s.Src.Name != nil {
		return s.Src.Name.Name
	}
	pkgpath := s.Path
	sepPos := strings.Index(pkgpath, PackageTypeSeparator)
	if sepPos != -1 {
		pkgpath = pkgpath[sepPos+len(PackageTypeSeparator):]
	}
	return path.Base(pkgpath)
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
		X:    x.(*Ident),
		Decl: s,
	}
}

// Same returns true if the other storage is this storage.
func (s *ImportDecl) Same(o Storage) bool {
	return Storage(s) == o
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

// ImportEmbedded imports a package.
type ImportEmbedded struct {
	Src     ast.Expr
	Package *Package
}

var _ Expr = (*ImportEmbedded)(nil)

func (*ImportEmbedded) node() {}

// Node returns the source of the import.
func (ie *ImportEmbedded) Node() ast.Node {
	return ie.Src
}

// SourceString returns a string for the embedded import.
func (ie *ImportEmbedded) SourceString(from *File) string {
	return fmt.Sprintf("import(%s)", ie.Package.Path())
}

// Unroll the import expression.
func (ie *ImportEmbedded) Unroll(Fetcher, Unroller) (ast.Expr, bool) {
	return ie.Src, true
}

// Expr return the source expression.
func (ie *ImportEmbedded) Expr() ast.Expr {
	return ie.Src
}

// Type of the import expression.
func (ie *ImportEmbedded) Type() Type {
	return PackageType()
}
