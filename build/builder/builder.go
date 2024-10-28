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

// Package builder builds a GX intermediate representation (IR).
// First, the GX source code is parsed to build a
// [go/ast] tree.
//
// This package then builds an internal tree from the [go/ast]
// tree to resolve types and references. The root of the internal
// tree is a package. Each node in the tree refers to an
// external IR node that will be the output of the compilation.
//
// The output IR tree is build from multiple passes of the
// internal tree:
//  1. internal nodes and their matching external GIR node
//     are created
//  2. all types are resolved.
//  3. the external IR tree is filled.
//
// Errors are accumulated during these phases to report
// to the user. The output tree is available even if some
// errors occurred.
package builder

import (
	"io/fs"

	"github.com/pkg/errors"
	"github.com/gx-org/gx/build/ir"
)

type (
	// Package used by the builder to build a corresponding IR package.
	Package interface {
		// ImportID adds definition to the package given some intermediate representation.
		ImportIR(repr *ir.Package) error
		// BuildFiles adds definition to the package from a list of files at a path.
		BuildFiles(fs fs.FS, filenames []string) error
		// IR returns the current package intermediate representation.
		IR() *ir.Package
	}

	// Importer loads a package given its path.
	Importer interface {
		// Support checks if the importer supports the import path given its prefix.
		Support(path string) bool
		// Import a path.
		Import(bld *Builder, path string) (Package, error)
	}

	// Builder represents a build session from text to
	// the intermediate representation.
	Builder struct {
		importers []Importer
		packages  map[string]*bPackage
	}
)

// New returns a new build session.
func New(importers []Importer) *Builder {
	b := &Builder{
		packages:  make(map[string]*bPackage),
		importers: importers,
	}
	return b
}

// Build a package given its path.
func (b *Builder) Build(path string) (*ir.Package, error) {
	pkg, err := b.importPath(path)
	if err != nil {
		return nil, err
	}
	return pkg.IR(), nil
}

func (b *Builder) findImporter(path string) Importer {
	for _, imp := range b.importers {
		if imp.Support(path) {
			return imp
		}
	}
	return nil
}

func (b *Builder) importPath(path string) (*bPackage, error) {
	built, ok := b.packages[path]
	if ok {
		return built, nil
	}
	imp := b.findImporter(path)
	if imp == nil {
		return nil, errors.Errorf("cannot find an importer for %s", path)
	}
	pkg, err := imp.Import(b, path)
	if err != nil {
		return nil, err
	}
	bPkg := pkg.(*bPackage)
	b.packages[path] = bPkg
	return bPkg, nil
}

// NewPackage returns a new builder package given the path of the package and its name.
func (b *Builder) NewPackage(path, name string) (Package, error) {
	if name == "" {
		return nil, errors.Errorf("cannot create package at path %q with an empty name", path)
	}
	pkg := b.newPackage(path, name)
	b.packages[pkg.IR().FullName()] = pkg
	return pkg, nil
}

// Importers returns the importers used by the builder.
func (b *Builder) Importers() []Importer {
	return b.importers
}

// BuildFiles builds a package from a list of files.
// Note that the package is not registered by the builder.
func (b *Builder) BuildFiles(path string, fs fs.FS, filenames []string) (Package, error) {
	pkg := b.newPackage(path, "")
	return pkg, pkg.BuildFiles(fs, filenames)
}
