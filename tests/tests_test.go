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

package tests_test

import (
	"path/filepath"
	"testing"

	"github.com/gx-org/gx/build/builder"
	"github.com/gx-org/gx/build/importers"
	"github.com/gx-org/gx/stdlib"
	gxtesting "github.com/gx-org/gx/tests/testing"
	"github.com/gx-org/gx/tests"
)

func TestCompilerErrors(t *testing.T) {
	bld := builder.New(importers.NewCacheLoader(
		stdlib.Importer(nil),
		tests.Importer(),
	))
	for _, path := range tests.Errors {
		_, testName := filepath.Split(path)
		t.Run(testName, func(t *testing.T) {
			pkg, err := bld.Build(path)
			gxtesting.RunAll(t, nil, pkg.IR(), err)
		})
	}
}
