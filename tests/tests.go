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
	"github.com/gx-org/gx/build/builder"
	"github.com/gx-org/gx/build/builder/testbuild"
	"github.com/gx-org/gx/build/importers/embedpkg"
	"github.com/gx-org/gx/build/importers"
	"github.com/gx-org/gx/stdlib"
	"github.com/gx-org/gx/tests/rtpackages/base"
	"github.com/gx-org/gx/tests/testpkgs"

	// Testdata
	core "github.com/gx-org/gx/build/builder/testdata"

	control "github.com/gx-org/gx/stdlib/control/testdata"
	dtype "github.com/gx-org/gx/stdlib/dtype/testdata"
	grad "github.com/gx-org/gx/stdlib/math/grad/testdata"
	math "github.com/gx-org/gx/stdlib/math/testdata"
	num "github.com/gx-org/gx/stdlib/num/testdata"
	rand "github.com/gx-org/gx/stdlib/rand/testdata"
	reflect "github.com/gx-org/gx/stdlib/reflect/testdata"
	shape "github.com/gx-org/gx/stdlib/shape/testdata"
)

// CoreUnits tests core language features.
var CoreUnits = []testbuild.TestFactory{
	core.Sources,
}

// StdlibUnits groups all the test in the standard library.
var StdlibUnits = []testbuild.TestFactory{
	control.Sources,
	dtype.Sources,
	shape.Sources,
	math.Sources,
	grad.Sources,
	num.Sources,
	rand.Sources,
	reflect.Sources,
}

// Importers used for testing.
var Importers = []importers.Importer{
	testpkgs.TestMacros,
	testpkgs.TestMacroError,
	testpkgs.TestCPUndef,
	testpkgs.TestGenerics,
	testpkgs.TestUnknown,
	testpkgs.TestFieldPaths,
	testpkgs.TestPkg1,
	base.New(),
	stdlib.Importer(),
	embedpkg.New(),
}

// NewBuilder returns a new builder given some importers.
// An additional importer to load test files is automatically added.
func NewBuilder() *builder.Builder {
	return builder.New(Importers...)
}
