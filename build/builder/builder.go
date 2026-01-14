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
	"github.com/gx-org/gx/build/importers"
	"github.com/gx-org/gx/build/ir"
)

type (
	// bPackage is a private package implemented by both *FilePackage and *IncrementalPackage.
	bPackage interface {
		importers.Package
		base() *basePackage
	}

	// Builder represents a build session from text to
	// the intermediate representation.
	Builder struct {
		loader importers.Loader
	}
)

var _ ir.Importer = (*Builder)(nil)

// New returns a new build session.
func New(loader importers.Loader) *Builder {
	return &Builder{loader: loader}
}

// Build a package given its path.
func (b *Builder) Build(path string) (importers.Package, error) {
	return b.loader.Load(b, path)
}

// Loader returns the package loader of the builder.
func (b *Builder) Loader() importers.Loader {
	return b.loader
}

// NewPackage returns a new builder package given the path of the package and its name.
func (b *Builder) NewPackage(path, name string) (importers.FilePackage, error) {
	if name == "" {
		return nil, errors.Errorf("cannot create package at path %q with an empty name", path)
	}
	return b.newFilePackage(path, name), nil
}

func (b *Builder) importPath(path string) (bPackage, error) {
	pkg, err := b.loader.Load(b, path)
	if err != nil {
		return nil, err
	}
	return pkg.(bPackage), nil
}

// Import a package from its path using the builder.
func (b *Builder) Import(path string) (*ir.Package, error) {
	pkg, err := b.loader.Load(b, path)
	if err != nil {
		return nil, err
	}
	return pkg.IR(), nil
}

// BuildFiles builds a package from a list of files.
// Note that the package is not registered by the builder.
func (b *Builder) BuildFiles(packagePath, packageName string, fs fs.FS, filenames []string) (importers.Package, error) {
	pkg := b.newFilePackage(packagePath, packageName)
	return pkg, pkg.BuildFiles(fs, filenames)
}
