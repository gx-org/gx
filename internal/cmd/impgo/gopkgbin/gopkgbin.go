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

// Importer satisfies the types.Importer interface, loading export data
// files in gc, gccgo and appengine formats from a google3 build tree.
type Importer struct{}

// NewImporter given options.
func NewImporter(importDir, gccgo, installsuffix, goroot string) (*Importer, error) {
	return &Importer{}, nil
}

// Import returns the imported package for the given import path.
func (imp *Importer) Import(path string) (pkg *types.Package, err error) {
	cfg := &packages.Config{Mode: packages.NeedFiles | packages.NeedSyntax}
	pkgs, err := packages.Load(cfg, path)
	if err != nil {
		return nil, err
	}
	return pkgs[0].Types, nil
}
