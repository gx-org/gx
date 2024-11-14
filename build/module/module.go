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

var (
	current     *Module
	currentErr  error
	currentOnce sync.Once
)

// Current returns the module of the current working directory.
func Current() (*Module, error) {
	currentOnce.Do(func() {
		var wd string
		wd, currentErr = os.Getwd()
		if currentErr != nil {
			return
		}
		current, currentErr = New(wd)
	})
	return current, currentErr
}

// New returns a new module.
func New(osPath string) (*Module, error) {
	modRoot := findModuleRoot(osPath)
	if modRoot == "" {
		return nil, errors.Errorf("directory %q is not a Go module: cannot find go.mod", osPath)
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
		err = fmt.Errorf("empty package path after module name %s", mod.Name())
		return
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
