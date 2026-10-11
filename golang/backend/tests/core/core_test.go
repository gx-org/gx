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

package core_test

import (
	"testing"

	"github.com/gomlx/compute/gobackend"
	"github.com/gx-org/gx/api"
	"github.com/gx-org/gx/build/builder/testbuild"
	core "github.com/gx-org/gx/build/builder/testdata"
	"github.com/gx-org/gx/internal/testing/testrtm"
	"github.com/gx-org/gx/tests"
)

var skip = map[string]bool{
	"set/set01.gx":       true,
	"einsum/einsum.gx":   true,
	"einsum/einsum02.gx": true,
}

func TestCoreUnitGoBackend(t *testing.T) {
	// Skip some tests.
	core.Sources.Filter = func(name string) bool {
		return !skip[name]
	}
	bck, err := gobackend.New("")
	if err != nil {
		t.Fatal(err)
	}
	rtm := api.NewRuntime(bck, tests.NewBuilder())
	testbuild.RunFactory(t,
		rtm.Builder().Loader().Importers(),
		testrtm.Factory(
			rtm,
			tests.CoreUnits...,
		),
	)
}
