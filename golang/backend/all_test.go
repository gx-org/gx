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

package backend_test

import (
	"testing"

	"github.com/gx-org/gx/build/builder/testbuild"
	core "github.com/gx-org/gx/build/builder/testdata"
	"github.com/gx-org/gx/golang/backend"
	"github.com/gx-org/gx/internal/testing/testrtm"
	rand "github.com/gx-org/gx/stdlib/rand/testdata"
	"github.com/gx-org/gx/tests"
)

var skip = map[string]bool{
	"set/set01.gx":       true,
	"einsum/einsum.gx":   true,
	"einsum/einsum02.gx": true,
	// control.While requires function literals to implement grapheval.SubGrapher,
	// which is not yet supported at runtime.
	"control/control.gx": true,
	// gobackend's RNGBitGenerator uses Go's PCG generator instead of XLA's Philox
	// algorithm, producing different random values from the expected test outputs.
	"misc.gx":      true,
	"rand_test.gx": true,
	"structs.gx":   true,
}

func TestGoBackend(t *testing.T) {
	// Skip some tests.
	filter := func(name string) bool {
		return !skip[name]
	}
	core.Sources.Filter = filter
	rand.Sources.Filter = filter
	rtm, err := backend.New(tests.NewBuilder())
	if err != nil {
		t.Fatal(err)
	}
	testbuild.RunFactory(t,
		rtm.Builder().Loader().Importers(),
		testrtm.Factory(
			rtm,
			tests.All...,
		),
	)
}
