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

// Package fsimporter loads imports and files using a virtual Go filesystem.
//
// More specifically, given a path, the loader compiles all the GX files at that
// path in single package. Consequently, the same rule than open-source Go applies:
// one GX package per folder.
package fsimporter

import (
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"github.com/gx-org/gx/build/builder"
	"github.com/gx-org/gx/build/importers"
)

type (
	// Importer maps GX package path to a Build function defined
	// by the Go library packaging the GX files.
	Importer struct {
		fs fs.ReadDirFS
	}
)

var _ importers.Importer = (*Importer)(nil)

// New returns a new importer.
func New(fs fs.ReadDirFS) *Importer {
	return &Importer{fs: fs}
}

// FS returns the filesystem used by the importer.
func (imp *Importer) FS() fs.ReadDirFS {
	return imp.fs
}

// Support always returns true.
// This importer is only used to store GX test files.
func (imp *Importer) Support(path string) bool {
	return true
}

// Import a package given its path.
func (imp *Importer) Import(bld *builder.Builder, path string) (builder.Package, error) {
	return CompileDir(bld, imp.fs, path)
}

// ImportAs imports a package given its package path and a directory.
func (imp *Importer) ImportAs(bld *builder.Builder, packagePath, dirName string) (builder.Package, error) {
	return CompileDirAs(bld, imp.fs, packagePath, dirName)
}

// CompileDir returns a package compile from a directory.
func CompileDir(bld *builder.Builder, fs fs.ReadDirFS, dirName string) (builder.Package, error) {
	packagePath := filepath.Dir(dirName)
	return CompileDirAs(bld, fs, packagePath, dirName)
}

// CompileDirAs compiles a directory.
func CompileDirAs(bld *builder.Builder, fs fs.ReadDirFS, packagePath, dirName string) (builder.Package, error) {
	entries, err := fs.ReadDir(dirName)
	if err != nil {
		return nil, err
	}
	var inputFiles []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if !strings.HasSuffix(entry.Name(), ".gx") {
			continue
		}
		entryPath := filepath.Join(dirName, entry.Name())
		inputFiles = append(inputFiles, entryPath)
	}
	if len(inputFiles) == 0 {
		return nil, errors.Errorf("no source file found for package %s in directory %s", packagePath, dirName)
	}
	return bld.BuildFiles(packagePath, fs, inputFiles)
}
