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

// Package localfs builds GX source code from the local filesystem.
package localfs

import (
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"github.com/gx-org/gx/build/builder"
	"github.com/gx-org/gx/build/importers"
	"github.com/gx-org/gx/build/module"
	"github.com/gx-org/gx/stdlib"
)

// Importer imports GX packages from the local file system.
type Importer struct {
	mod *module.Module
}

var _ importers.Importer = (*Importer)(nil)

// New returns a new GX using the local filesystem and Go module.
// If no Go module can be found, returns a nil importer and a nil error.
func New() (*Importer, error) {
	mod, err := module.Current()
	if mod == nil || err != nil {
		return nil, err
	}
	return NewWithModule(mod), nil
}

// NewWithModule returns an importer given a module.
func NewWithModule(mod *module.Module) *Importer {
	return &Importer{mod: mod}
}

// Module used by the importer.
func (imp *Importer) Module() *module.Module {
	return imp.mod
}

// NewBuilder returns a builder using the local filesystem to find package.
// This function should only be used to generate bindings.
func NewBuilder() (*builder.Builder, error) {
	importer, err := New()
	if err != nil {
		return nil, err
	}
	return builder.New(importers.NewCacheLoader(
		stdlib.Importer(nil),
		importer,
	)), nil
}

// Support returns if the package belongs to the module.
func (imp *Importer) Support(path string) bool {
	return imp.mod.Belongs(path)
}

// Import a package given its path.
func (imp *Importer) Import(bld *builder.Builder, importPath string) (builder.Package, error) {
	packagePath, packageName, err := imp.mod.Split(importPath)
	if err != nil {
		return nil, errors.Errorf("cannot import path %s: %v", importPath, err)
	}
	pkg, err := ImportAt(bld, imp.mod.FS(), importPath, packagePath)
	if err != nil {
		return nil, err
	}
	if packageName != pkg.IR().Name.Name {
		return nil, errors.Errorf("package %s has files with package name %s", importPath, packageName)
	}
	return pkg, nil
}

// ImportAt imports a package given a path on the virtual file system.
// The last element of the import path needs to match the package names in all the GX source files
// present in the folder on the file system.
func ImportAt(bld *builder.Builder, vfs fs.ReadDirFS, importPath, fsPath string) (builder.Package, error) {
	entries, err := fs.ReadDir(vfs, fsPath)
	if err != nil {
		return nil, err
	}
	var inputFiles []string
	for _, entry := range entries {
		if !IsGXFile(entry) {
			continue
		}
		entryPath := filepath.Join(fsPath, entry.Name())
		inputFiles = append(inputFiles, entryPath)
	}
	if len(inputFiles) == 0 {
		return nil, errors.Errorf("cannot import %s: no source file found in directory %s", importPath, fsPath)
	}
	packagePathS := strings.Split(importPath, "/")
	return bld.BuildFiles(
		strings.Join(packagePathS[:len(packagePathS)-1], "/"),
		packagePathS[len(packagePathS)-1],
		vfs, inputFiles)
}

// IsGXFile returns true if a directory entry is a GX source file.
func IsGXFile(entry fs.DirEntry) bool {
	return !entry.IsDir() && strings.HasSuffix(entry.Name(), ".gx")
}
