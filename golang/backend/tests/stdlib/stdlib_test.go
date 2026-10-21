// Copyright 2024 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package stdlib_test

import (
	"testing"

	"github.com/gx-org/gx/build/builder/testbuild"
	"github.com/gx-org/gx/golang/backend"
	"github.com/gx-org/gx/internal/testing/testrtm"
	rand "github.com/gx-org/gx/stdlib/rand/testdata"
	"github.com/gx-org/gx/tests"
)

func TestStdlib(t *testing.T) {
	// Skip rand tests because gobackend's RNGBitGenerator uses Go's PCG generator
	// instead of XLA's Philox algorithm, producing different random values from
	// the expected test outputs.
	rand.Sources.Filter = func(string) bool {
		return false
	}
	rtm, err := backend.New(tests.NewBuilder())
	if err != nil {
		t.Fatal(err)
	}
	testbuild.RunFactory(t,
		rtm.Builder().Loader().Importers(),
		testrtm.Factory(
			rtm,
			tests.StdlibUnits...,
		),
	)
}
