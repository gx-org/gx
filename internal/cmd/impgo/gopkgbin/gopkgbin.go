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

// Package gopkgbin loads Go library to inspect them.
package gopkgbin

import (
	"os"
	"path"
	"path/filepath"
	"strings"

	"golang.org/x/tools/go/packages"
	"github.com/gx-org/gx/build/module"
	"github.com/gx-org/gx/internal/cmd/impgo/generator"
)

// Loader loads go binary package.
type Loader struct {
	mod *module.Module
}

var _ generator.Loader = (*Loader)(nil)

// New loader.
func New() (*Loader, error) {
	mod, err := module.Current()
	if err != nil {
		return nil, err
	}
	return &Loader{mod: mod}, nil
}

// Load a package given the Go import path.
func (*Loader) Load(pkgpath string) (pkg generator.Pkg, err error) {
	pkg.Dir, pkg.Name = path.Split(pkgpath)
	cfg := &packages.Config{Mode: packages.LoadTypes | packages.LoadSyntax}
	var pkgs []*packages.Package
	pkgs, err = packages.Load(cfg, pkgpath)
	if err != nil {
		return
	}
	pkg.Pkg = pkgs[0].Types
	return
}

// BuildPath build the output path given a package path and a file name.
func (l *Loader) BuildPath(pkgpath, filename string) (string, error) {
	name := path.Base(pkgpath) + "_importgo"
	innerPath := strings.TrimPrefix(pkgpath, l.mod.Name())
	folder := filepath.Join(l.mod.OSPath(innerPath), name)
	if err := os.MkdirAll(folder, 0750); err != nil {
		return "", err
	}
	return filepath.Join(folder, filename), nil
}

// GoImportPath modifies, if necessary, an import path used in the generated Go file.
func (l *Loader) GoImportPath(pkgpath string) string {
	return pkgpath
}
