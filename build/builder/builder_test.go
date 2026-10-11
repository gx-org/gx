// Copyright 2025 Google LLC
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

package builder_test

import (
	"testing"

	"github.com/gx-org/gx/build/builder/testbuild"
	"github.com/gx-org/gx/build/builder/testdata"
	"github.com/gx-org/gx/build/importers"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/build/ir/irhelper"
	"github.com/gx-org/gx/stdlib"
	"github.com/gx-org/gx/tests/rtpackages/base"
	"github.com/gx-org/gx/tests/testpkgs"
)

var (
	wantPackage = &ir.Package{Name: irhelper.IdentAST("test")}
	wantFile    = &ir.File{Package: wantPackage}
)

func TestBuilder(t *testing.T) {
	testbuild.RunFactory(t,
		// Use a custom list of importers to make sure we do not need to compile GX code to run this test.
		[]importers.Importer{
			testpkgs.TestMacros,
			testpkgs.TestMacroError,
			testpkgs.TestCPUndef,
			testpkgs.TestGenerics,
			testpkgs.TestUnknow,
			base.New(),
			stdlib.Importer(),
		},
		testbuild.CompEval(
			testdata.Sources,
		))
}
