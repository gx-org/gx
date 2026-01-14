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

// Package stdlib provides the GX standard library that is independent of a backend.
// This includes generic GX code and function signatures computation.
package stdlib

import (
	"maps"
	"slices"

	"github.com/pkg/errors"
	"github.com/gx-org/gx/build/importers"
	"github.com/gx-org/gx/stdlib/builtin"
	"github.com/gx-org/gx/stdlib/control"
	"github.com/gx-org/gx/stdlib/dtype"
	"github.com/gx-org/gx/stdlib/impl"
	"github.com/gx-org/gx/stdlib/math/grad"
	"github.com/gx-org/gx/stdlib/math"
	"github.com/gx-org/gx/stdlib/num"
	"github.com/gx-org/gx/stdlib/rand"
	"github.com/gx-org/gx/stdlib/shapes"
)

// Stdlib builds the standard library given import paths.
type Stdlib struct {
	impl *impl.Stdlib
	libs map[string]builtin.PackageBuilder
}

var _ importers.Importer = (*Stdlib)(nil)

var packages = []builtin.PackageBuilder{
	control.Package,
	dtype.Package,
	grad.Package,
	math.Package,
	num.Package,
	rand.Package,
	shapes.Package,
}

// Importer returns the standard library importer.
func Importer(stdlibImpl *impl.Stdlib) *Stdlib {
	if stdlibImpl == nil {
		stdlibImpl = &impl.Stdlib{}
	}
	lib := &Stdlib{
		impl: stdlibImpl,
		libs: make(map[string]builtin.PackageBuilder),
	}
	for _, pkgBuilder := range packages {
		lib.libs[pkgBuilder.FullPath] = pkgBuilder
	}
	return lib
}

// Support returns true if the path is a GX standard library.
func (l *Stdlib) Support(path string) bool {
	_, ok := l.libs[path]
	return ok
}

// Import a package given its path.
func (l *Stdlib) Import(bld importers.Builder, path string) (importers.Package, error) {
	pkgBuilder, ok := l.libs[path]
	if !ok {
		return nil, errors.Errorf("package %s is not in std", path)
	}
	return builtin.Build(bld, l.impl, pkgBuilder)
}

// Paths returns all the paths in the standard library
// (alphabetically ordered).
func (l *Stdlib) Paths() []string {
	return slices.Sorted(maps.Keys(l.libs))
}
