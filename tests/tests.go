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
	"github.com/gx-org/gx/build/importers/fsimporter"
	"github.com/gx-org/gx/build/importers"
	"github.com/gx-org/gx/stdlib/impl"
	"github.com/gx-org/gx/stdlib"
)

// FS is the filesystem containing all GX test files.
//
//go:embed testfiles errors
var FS embed.FS

// Errors is the set of paths testing for compiler errors.
var Errors = []string{
	"errors/einsum",
	"errors/ellipsis",
	"errors/process",
	"errors/resolve",
	"errors/slices",
	"errors/stdlib",
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
	"testfiles/stdlib",
	"testfiles/nn",
	"testfiles/ellipsis",
}

func appendAll(paths ...[]string) []string {
	r := []string{}
	for _, ps := range paths {
		r = append(r, ps...)
	}
	return r
}

// All includes all paths.
var All = appendAll(Language, Stdlib)

// CoreBuilder returns the builder to run core tests.
func CoreBuilder() *builder.Builder {
	return builder.New(importers.NewCacheLoader(fsimporter.New(FS)))
}

// StdlibBuilder returns the builder to run tests with the standard library.
func StdlibBuilder(stdlibImpl *impl.Stdlib) *builder.Builder {
	return builder.New(importers.NewCacheLoader(
		stdlib.Importer(stdlibImpl),
		fsimporter.New(FS),
	))
}
