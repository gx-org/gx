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
	"sync"
	"sync/atomic"

	"github.com/pkg/errors"
	"github.com/gx-org/gx/build/builder"
)

type (
	// ImporterAdder is implemented by loaders to which importers can be added
	// after the loader has been created.
	ImporterAdder interface {
		AddImporter(Importer)
	}

	// PathReseter resets a path in the loader to force a rebuild.
	PathReseter interface {
		ResetPath(string)
	}
)

func findImporter(importers []Importer, path string) Importer {
	for _, imp := range importers {
		if imp.Support(path) {
			return imp
		}
	}
	return nil
}

// Load a package given its path.
func Load(bld *builder.Builder, importers []Importer, path string) (builder.Package, error) {
	imp := findImporter(importers, path)
	if imp == nil {
		return nil, errors.Errorf("cannot find an importer for %s", path)
	}
	return imp.Import(bld, path)
}

type packageBuilder struct {
	cl *CacheLoader

	mut sync.Mutex
	// Use an atomic so that the builder can be reset without locking
	// (prevent a deadlock when the builder is building a package while
	// the loader is being reset)
	done atomic.Bool
	pkg  builder.Package
	err  error
}

func (pb *packageBuilder) reset() {
	pb.done.Store(false)
}

func (pb *packageBuilder) build(bld *builder.Builder, path string) (builder.Package, error) {
	pb.mut.Lock()
	defer pb.mut.Unlock()

	if pb.done.Load() {
		return pb.pkg, pb.err
	}

	pb.done.Store(true)
	pb.pkg, pb.err = Load(bld, pb.cl.importers, path)
	if pb.pkg != nil {
		// Force building the IR to avoid concurrent construction.
		pb.pkg.IR()
	}
	return pb.pkg, pb.err
}

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
func (cl *CacheLoader) Load(bld *builder.Builder, path string) (builder.Package, error) {
	return cl.loadPackageBuilder(path).build(bld, path)
}
