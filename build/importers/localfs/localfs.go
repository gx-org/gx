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
	"fmt"
	"io/fs"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"golang.org/x/mod/modfile"
	"github.com/gx-org/gx/build/builder"
	"github.com/gx-org/gx/build/importers/fsimporter"
	"github.com/gx-org/gx/stdlib/impl"
	"github.com/gx-org/gx/stdlib"
)

func findModuleRoot(dir string) (roots string) {
	dir = filepath.Clean(dir)
	if dir == "" {
		return ""
	}
	for {
		if fi, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil && !fi.IsDir() {
			return dir
		}
		d := filepath.Dir(dir)
		if d == dir {
			break
		}
		dir = d
	}
	return ""
}

// Module is the GX module.
type Module struct {
	err error

	root string
	mod  *modfile.File
	name string
	fs   fs.ReadDirFS
}

func initModule() (*Module, error) {
	wd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	modRoot := findModuleRoot(wd)
	if modRoot == "" {
		return nil, errors.Errorf("directory %q is not a Go module: cannot find go.mod", wd)
	}
	absModRoot, err := filepath.Abs(modRoot)
	if err != nil {
		return nil, errors.Errorf("invalid path %q: %v", modRoot, modRoot)
	}
	info := Module{
		root: absModRoot,
		fs:   os.DirFS(absModRoot).(fs.ReadDirFS),
	}
	modPath := filepath.Join(modRoot, "go.mod")
	modData, err := ioutil.ReadFile(modPath)
	if err != nil {
		return nil, errors.Errorf("cannot read %s: %v", absModRoot, err)
	}
	info.mod, err = modfile.Parse(modPath, modData, nil)
	if err != nil {
		return nil, errors.Errorf("cannot parse %s: %v", absModRoot, err)
	}
	for _, stmt := range info.mod.Syntax.Stmt {
		line, ok := stmt.(*modfile.Line)
		if !ok {
			continue
		}
		if len(line.Token) < 2 {
			continue
		}
		if line.Token[0] != "module" {
			continue
		}
		info.name = line.Token[1]
		break
	}
	return &info, nil
}

// FolderOf returns the folder of a GX package in the module.
func (mod *Module) FolderOf(pkgPath string) (string, error) {
	if !strings.HasPrefix(pkgPath, mod.name) {
		return "", fmt.Errorf("package %q does not belong to %s", pkgPath, mod.name)
	}
	return filepath.Join(mod.root, pkgPath[len(mod.name):]), nil
}

// Importer imports GX packages from the local file system.
type Importer struct {
	mod *Module
	imp *fsimporter.Importer
}

var _ builder.Importer = (*Importer)(nil)

// New returns a new GX using the local filesystem and Go module.
func New() (*Importer, error) {
	mod, err := initModule()
	if err != nil {
		return nil, err
	}
	imp := fsimporter.New(mod.fs)
	return &Importer{
		mod: mod,
		imp: imp,
	}, nil
}

// Module used by the importer.
func (imp *Importer) Module() *Module {
	return imp.mod
}

// NewBuilder returns a builder using the local filesystem to find package.
// This function should only be used to generate bindings.
func NewBuilder() (*builder.Builder, error) {
	importer, err := New()
	if err != nil {
		return nil, err
	}
	return builder.New([]builder.Importer{
		stdlib.Importer(nil),
		importer,
	}), nil
}

// Support returns if the package belongs to the module.
func (imp *Importer) Support(path string) bool {
	return strings.HasPrefix(path, imp.mod.name)
}

// Import a package given its path.
func (imp *Importer) Import(bld *builder.Builder, path string) (builder.Package, error) {
	if !imp.Support(path) {
		return nil, errors.Errorf("cannot import path %s: does not have prefix %q", path, imp.mod.name)
	}
	pathInModule := path[len(imp.mod.name):]
	pathInModule = strings.TrimPrefix(pathInModule, string(os.PathSeparator))

	packagePaths := strings.Split(path, "/")
	dirInPaths := packagePaths[:len(packagePaths)-1]
	nameInPaths := packagePaths[len(packagePaths)-1]
	packageDir := strings.Join(dirInPaths, "/")
	pkg, err := imp.imp.ImportAs(bld, packageDir, pathInModule)
	if err != nil {
		return nil, err
	}
	if nameInPaths != pkg.IR().Name.Name {
		return nil, errors.Errorf("package %s has files with package name %s", dirInPaths, nameInPaths)
	}
	return pkg, nil
}

// StdlibBuilder returns the GX builder with the bindings package and the given standard library
// registered in the importer.
func StdlibBuilder(stdlibImpl *impl.Stdlib) (*builder.Builder, error) {
	importer, err := New()
	if err != nil {
		return nil, err
	}
	return builder.New([]builder.Importer{
		stdlib.Importer(stdlibImpl),
		importer,
	}), nil
}
