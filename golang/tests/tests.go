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

// Package tests includes all the Go bindings tests.
package tests

import (
	"testing"

	"github.com/gx-org/gx/api"
	"github.com/gx-org/gx/golang/tests/basictest"
	"github.com/gx-org/gx/golang/tests/cartpoletest"
	"github.com/gx-org/gx/golang/tests/dtypestest"
	"github.com/gx-org/gx/golang/tests/importtest"
	"github.com/gx-org/gx/golang/tests/mathtest"
	"github.com/gx-org/gx/golang/tests/parameterstest"
	"github.com/gx-org/gx/golang/tests/pkgvarstest"
	"github.com/gx-org/gx/golang/tests/randtest"
	"github.com/gx-org/gx/golang/tests/unexportedtest"
)

var all = []func(t *testing.T, dev *api.Device){
	basictest.Run,
	importtest.Run,
	mathtest.Run,
	parameterstest.Run,
	pkgvarstest.Run,
	randtest.Run,
	dtypestest.Run,
	cartpoletest.Run,
	unexportedtest.Run,
}

// RunAll runs all the bindings tests.
func RunAll(t *testing.T, rtm *api.Runtime) {
	dev, err := rtm.Device(0)
	if err != nil {
		t.Fatal(err)
	}
	Run(t, dev, all)
}

// Run all the Go binding tests.
func Run(t *testing.T, dev *api.Device, tests []func(t *testing.T, dev *api.Device)) {
	t.Helper()
	for _, funTest := range tests {
		funTest(t, dev)
	}
}
