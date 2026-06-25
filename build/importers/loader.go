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

	// Loader loads packages given their import path.
	Loader interface {
		Load(bld Builder, path string) (Package, error)
		Importers() []Importer
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

type packageBuilder struct {
	cl *CacheLoader

	mut sync.Mutex
	// Use an atomic so that the builder can be reset without locking
	// (prevent a deadlock when the builder is building a package while
	// the loader is being reset)
	done atomic.Bool
	pkg  Package
	err  error
}

func (pb *packageBuilder) reset() {
	pb.done.Store(false)
}

func load(bld Builder, importers []Importer, path string) (Package, error) {
	imp := findImporter(importers, path)
	if imp == nil {
		return nil, errors.Errorf("cannot find an importer for %s", path)
	}
	return imp.Import(bld, path)
}

func (pb *packageBuilder) build(bld Builder, path string) (Package, error) {
	pb.mut.Lock()
	defer pb.mut.Unlock()

	if pb.done.Load() {
		return pb.pkg, pb.err
	}

	pb.done.Store(true)
	pb.pkg, pb.err = load(bld, pb.cl.importers, path)
	return pb.pkg, pb.err
}
