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
	"github.com/gx-org/gx/build/importers/embedpkg"
	"github.com/gx-org/gx/build/importers"
	"github.com/gx-org/gx/stdlib"
	"github.com/gx-org/gx/tests/rtpackages/base"
	"github.com/gx-org/gx/tests/testpkgs"

	// Testdata
	core "github.com/gx-org/gx/build/builder/testdata"

	grad "github.com/gx-org/gx/stdlib/math/grad/testdata"
	math "github.com/gx-org/gx/stdlib/math/testdata"
	num "github.com/gx-org/gx/stdlib/num/testdata"
	reflect "github.com/gx-org/gx/stdlib/reflect/testdata"
	shape "github.com/gx-org/gx/stdlib/shape/testdata"

	// Embedded GX packages.
	_ "github.com/gx-org/gx/tests/testfiles/imports/pkg1/pkg1_go_gx"
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
	"errors/functype",
	"errors/beforefunc",
	"errors/resolve",
	"errors/redefined",
	"errors/slices",
	"errors/stdlib",
}

// Language is a set of paths testing the GX core language.
var Language = []string{
	"testfiles/forloops",
	"testfiles/ifstmts",
	"testfiles/imports",
	"testfiles/arrayslices",
	"testfiles/slices",
}

// Stdlib is a set of path testing the standard library or where a standard library implementation is required.
var Stdlib = []string{
	"testfiles/bfloat16",
	"testfiles/stdlib",
	"testfiles/nn",
	"testfiles/ellipsis",
	"testfiles/compeval",
}

// CoreUnits tests core language features.
var CoreUnits = []testbuild.TestFactory{
	core.Sources,
}

// StdlibUnits groups all the test in the standard library.
var StdlibUnits = []testbuild.TestFactory{
	shape.Sources,
	math.Sources,
	grad.Sources,
	num.Sources,
	reflect.Sources,
}

func appendAll(paths ...[]string) []string {
	var r []string
	for _, ps := range paths {
		r = append(r, ps...)
	}
	return r
}

// All includes all paths.
var All = appendAll(Language, Stdlib)

// Importers used for testing.
var Importers = []importers.Importer{
	testpkgs.TestMacros,
	testpkgs.TestMacroError,
	testpkgs.TestCPUndef,
	testpkgs.TestGenerics,
	testpkgs.TestUnknown,
	testpkgs.TestFieldPaths,
	base.New(),
	stdlib.Importer(),
	embedpkg.New(),
}

// NewBuilder returns a new builder given some importers.
// An additional importer to load test files is automatically added.
func NewBuilder() *builder.Builder {
	return builder.New(Importers...)
}
