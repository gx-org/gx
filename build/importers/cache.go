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

package importers

import "sync"

// CacheLoader loads a package and then cache it
// for future access. Consequently, a package is only
// compiled once.
type CacheLoader struct {
	mut       sync.Mutex
	importers []Importer
	packages  map[string]*packageBuilder
}

var (
	_ ImporterAdder = (*CacheLoader)(nil)
	_ PathReseter   = (*CacheLoader)(nil)
)

// NewCacheLoader returns a new loader given a set of importers.
func NewCacheLoader(importers ...Importer) *CacheLoader {
	return &CacheLoader{
		packages:  make(map[string]*packageBuilder),
		importers: importers,
	}
}

// ResetPath forces a build next time package at path will be loaded.
func (cl *CacheLoader) ResetPath(path string) {
	cl.mut.Lock()
	defer cl.mut.Unlock()

	pb := cl.packages[path]
	if pb == nil {
		return
	}
	pb.reset()
}

// Clear the package cache.
func (cl *CacheLoader) Clear() {
	cl.mut.Lock()
	defer cl.mut.Unlock()

	for _, pb := range cl.packages {
		pb.reset()
	}
}

// AddImporter adds an importer.
func (cl *CacheLoader) AddImporter(imp Importer) {
	cl.mut.Lock()
	defer cl.mut.Unlock()

	// Insert the importer in first position so that the latest
	// added always has the priority.
	cl.importers = append([]Importer{imp}, cl.importers...)
}

func (cl *CacheLoader) loadPackageBuilder(path string) *packageBuilder {
	cl.mut.Lock()
	defer cl.mut.Unlock()

	pkgBuilder := cl.packages[path]
	if pkgBuilder != nil {
		return pkgBuilder
	}
	pkgBuilder = &packageBuilder{cl: cl}
	cl.packages[path] = pkgBuilder
	return pkgBuilder
}

// Load a package given its path.
func (cl *CacheLoader) Load(bld Builder, path string) (Package, error) {
	return cl.loadPackageBuilder(path).build(bld, path)
}
