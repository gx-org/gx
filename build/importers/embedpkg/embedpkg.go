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

// Package embedpkg loads imports and files using the embed Go package.
//
// More specifically, the GX source files of a GX package are embedded into
// a matching Go package. This process is done by
// [google3/third_party/gxlang/gx/golang/packager/packager] using the
// [embed] package from the Go standard library. These GX-Go packages are
// then statically linked into the binary.
//
// At runtime, all GX packages are registered at startup using this package.
// When a GX package is imported, the embedded source code is compiled into
// the GX intermediate representation and returned to the caller.
//
// This intermediate representation can be compiled for a specific device
// by the GX interpreter or can be used to generate bindings for a given language.
package embedpkg

import (
	"github.com/pkg/errors"
	"github.com/gx-org/gx/build/builder"
	"github.com/gx-org/gx/build/importers"
	"github.com/gx-org/gx/stdlib/impl"
	"github.com/gx-org/gx/stdlib"
)

type (
	// BuildFunc defines the function that a Go library packaging GX files needs to implement.
	BuildFunc func(importers.Builder) (importers.Package, error)

	// Importer maps GX package path to a Build function defined
	// by the Go library packaging the GX files.
	Importer struct{}
)

var _ importers.Importer = (*Importer)(nil)

var registered = make(map[string]BuildFunc)

// RegisterPackage registers the build function of a package given its path.
func RegisterPackage(path string, buildFunc BuildFunc) {
	registered[path] = buildFunc
}

// New returns a new importer.
func New() importers.Importer {
	return &Importer{}
}

// Support returns true if path has been registered.
func (imp *Importer) Support(path string) bool {
	_, ok := registered[path]
	return ok
}

// Import a package given its path.
func (imp *Importer) Import(bld importers.Builder, path string) (importers.Package, error) {
	buildFunc, ok := registered[path]
	if !ok {
		return nil, errors.Errorf("cannot find package %s", path)
	}
	return buildFunc(bld)
}

// NewBuilder returns a builder loading GX files from the files embedded in the binary.
func NewBuilder(stdlibImpl *impl.Stdlib) *builder.Builder {
	return builder.New(
		stdlib.Importer(nil),
		New(),
	)
}
