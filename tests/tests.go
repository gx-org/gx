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

// Package tests embed all GX test files.
package tests

import (
	"embed"

	"github.com/gx-org/gx/build/builder"
	"github.com/gx-org/gx/build/builder/testbuild"
	"github.com/gx-org/gx/build/importers"
	"github.com/gx-org/gx/build/importers/localfs"
	"github.com/gx-org/gx/stdlib/impl"
	"github.com/gx-org/gx/stdlib"
	"github.com/gx-org/gx/tests/testmacros"
)

// FS is the filesystem containing all GX test files.
//
//go:embed testfiles errors
var FS embed.FS

// Errors is the set of paths testing for compiler errors.
var Errors = []string{
	"errors/einsum",
	"errors/ellipsis",
	"errors/meta",
	"errors/process",
	"errors/beforefunc",
	// "errors/resolve", // TODO(degris): FIX ASAP
	"errors/redefined",
	"errors/slices",
	// "errors/stdlib", // TODO(degris): FIX ASAP
}

// AlgebraicCore is a set of paths only testing core algebraic operations.
var AlgebraicCore = []string{
	"testfiles/core",
}

// LanguageCore is a set of paths testing the GX core language
// without built-in like einsum.
var LanguageCore = appendAll(AlgebraicCore, []string{
	"testfiles/forloops",
	"testfiles/generics",
	"testfiles/ifstmts",
	"testfiles/imports",
	"testfiles/slices",
})

// Language is a set of paths testing the GX language.
var Language = appendAll(LanguageCore, []string{
	"testfiles/einsum",
	"testfiles/trace",
})

// Stdlib is a set of path testing the standard library or where a standard library implementation is required.
var Stdlib = []string{
	"testfiles/bfloat16",
	"testfiles/grad",
	"testfiles/stdlib",
	"testfiles/nn",
	"testfiles/ellipsis",
	"testfiles/compeval",
}

// Units tests each file in the folder as its own package.
// It is used to facilitate debugging by isolating GX code for each file/package.
// Tests by default should be added to folders defined above.
// Only copy a test here to facilitate debugging.
var Units = []string{
	"testfiles/units",
}

func appendAll(paths ...[]string) []string {
	var r []string
	for _, ps := range paths {
		r = append(r, ps...)
	}
	return r
}

type importer struct {
	embedded map[string]testbuild.DeclarePackage
}

// Support checks if the importer supports the import path given its prefix.
func (imp importer) Support(path string) bool { return true }

// Import a path.
func (imp importer) Import(bld *builder.Builder, path string) (builder.Package, error) {
	embedded, isEmbedded := imp.embedded[path]
	if !isEmbedded {
		return localfs.ImportAt(bld, FS, path, path)
	}
	return embedded.Build(bld)
}

// Importer returns the importer for GX tests.
func Importer() importers.Importer {
	return importer{embedded: map[string]testbuild.DeclarePackage{
		"testmacros": testmacros.DeclarePackage,
	}}
}

// All includes all paths.
var All = appendAll(Language, Stdlib)

// CoreBuilder returns the builder to run core tests.
func CoreBuilder(embedded ...testbuild.DeclarePackage) *builder.Builder {
	return builder.New(importers.NewCacheLoader(Importer()))
}

// StdlibBuilder returns the builder to run tests with the standard library.
func StdlibBuilder(stdlibImpl *impl.Stdlib, embedded ...testbuild.DeclarePackage) *builder.Builder {
	return builder.New(importers.NewCacheLoader(
		stdlib.Importer(stdlibImpl),
		Importer(),
	))
}
