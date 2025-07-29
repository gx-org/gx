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

// Package module provides helper functions to handle GX modules.
package module

import (
	"fmt"
	"io/fs"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/pkg/errors"
	"golang.org/x/mod/modfile"
	"golang.org/x/mod/module"
)

const moduleFileName = "go.mod"

func findModuleRoot(dir string) (roots string) {
	dir = filepath.Clean(dir)
	if dir == "" {
		return ""
	}
	for {
		if fi, err := os.Stat(filepath.Join(dir, moduleFileName)); err == nil && !fi.IsDir() {
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

var (
	current     *Module
	currentErr  error
	currentOnce sync.Once
)

// Current returns the module of the current working directory.
func Current() (*Module, error) {
	currentOnce.Do(func() {
		current, currentErr = New("")
	})
	return current, currentErr
}

// New returns a new module given a directory.
// If no directory is given (i.e. empty string), then the current working directory is used.
// If no go.mod can be find, returns a nil module with a nil error.
func New(osPath string) (*Module, error) {
	if osPath == "" {
		var err error
		osPath, err = os.Getwd()
		if err != nil {
			return nil, errors.Errorf("cannot get current directory: %v", err)
		}
	}
	modRoot := findModuleRoot(osPath)
	if modRoot == "" {
		return nil, errors.Errorf("module file %s not found", moduleFileName)
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

// File returns the module go.mod file.
func (mod *Module) File() *modfile.File {
	return mod.mod
}

// Belongs returns true if a package, given its path, belongs to the module.
func (mod *Module) Belongs(importPath string) bool {
	return strings.HasPrefix(importPath, mod.name)
}

func (mod *Module) checkBelong(importPath string) error {
	if !mod.Belongs(importPath) {
		return fmt.Errorf("package %q does not belong to %s", importPath, mod.name)
	}
	return nil
}

// Split a full package into a module path, a package path, and a package name.
func (mod *Module) Split(importPath string) (packagePath, packageName string, err error) {
	if err = mod.checkBelong(importPath); err != nil {
		return
	}
	pkgWithName := strings.TrimPrefix(importPath, mod.Name())
	if pkgWithName == "" {
		return "", mod.Name(), nil
	}
	pkgWithNameElmts := strings.Split(pkgWithName, "/")
	packagePath = strings.Join(pkgWithNameElmts, "/")
	packagePath = strings.TrimPrefix(packagePath, "/")
	packageName = pkgWithNameElmts[len(pkgWithNameElmts)-1]
	return
}

// ImportToOSPath converts an import path to a path on the operating system.
func (mod *Module) ImportToOSPath(importPath string) (string, error) {
	if err := mod.checkBelong(importPath); err != nil {
		return "", err
	}
	return mod.OSPath(importPath[len(mod.name):]), nil
}

// OSPath converts a path within the module to a path on the operating system.
func (mod *Module) OSPath(path string) string {
	if path == "." || path == "" {
		return mod.root
	}
	return strings.Join([]string{
		mod.root,
		path,
	}, "/")
}

// FS returns the filesystem for the current module.
func (mod *Module) FS() fs.ReadDirFS {
	return mod.fs
}

// Name of the module as specified in the go.mod file.
func (mod *Module) Name() string {
	return mod.name
}

// GXPathFromOS returns the GX package from an OS path.
// Returns an empty string if the path does not belong to the package.
func (mod *Module) GXPathFromOS(osPath string) (string, error) {
	absPath, err := filepath.Abs(osPath)
	if err != nil {
		return "", fmt.Errorf("cannot get absolute path of %s: %v", osPath, err)
	}
	if !strings.HasPrefix(absPath, mod.root) {
		return "", fmt.Errorf("%s does not have path prefix %s", absPath, mod.root)
	}
	info, err := os.Stat(absPath)
	if err != nil {
		return "", fmt.Errorf("cannot read %s: %v", absPath, err)
	}
	if !info.IsDir() {
		absPath = filepath.Dir(absPath)
	}
	pkgPath := strings.TrimPrefix(absPath, mod.root)
	return filepath.Join(mod.name, pkgPath), nil
}

// Deps returns a list of all the direct dependencies.
func (mod *Module) Deps() []*module.Version {
	var deps []*module.Version
	for _, dep := range mod.mod.Require {
		if dep.Indirect {
			continue
		}
		deps = append(deps, &dep.Mod)
	}
	return deps
}

// VersionOf returns the version of a given module path
// or an empty string if not found.
func (mod *Module) VersionOf(path string) string {
	for _, dep := range mod.Deps() {
		if dep.Path == path {
			return dep.Version
		}
	}
	return ""
}

// Root returns the parent directory of go.mod.
func (mod *Module) Root() string {
	return mod.root
}
