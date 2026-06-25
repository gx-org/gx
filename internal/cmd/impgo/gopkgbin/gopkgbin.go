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
	"go/types"

	"golang.org/x/tools/go/packages"
)

// Loader loads go binary package.
type Loader struct{}

// New loader.
func New() (*Loader, error) {
	return &Loader{}, nil
}

// Load a package given the Go import path.
func (*Loader) Load(path string) (pkgDir, pkgName string, pkg *types.Package, err error) {
	cfg := &packages.Config{Mode: packages.NeedFiles | packages.NeedSyntax}
	var pkgs []*packages.Package
	pkgs, err = packages.Load(cfg, path)
	if err != nil {
		return
	}
	pkg = pkgs[0].Types
	return
}
