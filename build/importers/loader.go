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

package importers

import (
	"github.com/pkg/errors"
	"github.com/gx-org/gx/build/builder"
)

// CacheLoader loads a package and then cache it
// for future access. Consequently, a package is only
// compiled once.
type CacheLoader struct {
	importers []Importer
	packages  map[string]builder.Package
}

// NewCacheLoader returns a new loader given a set of importers.
func NewCacheLoader(importers ...Importer) *CacheLoader {
	return &CacheLoader{
		packages:  make(map[string]builder.Package),
		importers: importers,
	}
}

func (cl *CacheLoader) findImporter(path string) Importer {
	for _, imp := range cl.importers {
		if imp.Support(path) {
			return imp
		}
	}
	return nil
}

// Clear the package cache.
func (cl *CacheLoader) Clear() {
	for k := range cl.packages {
		delete(cl.packages, k)
	}
}

// Load a package given its path.
func (cl *CacheLoader) Load(bld *builder.Builder, path string) (builder.Package, error) {
	built, ok := cl.packages[path]
	if ok {
		return built, nil
	}
	imp := cl.findImporter(path)
	if imp == nil {
		return nil, errors.Errorf("cannot find an importer for %s", path)
	}
	pkg, err := imp.Import(bld, path)
	if err == nil {
		// Only save the package in the cache when no compilation error.
		cl.packages[path] = pkg
	}
	return pkg, err
}
