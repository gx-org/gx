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

// Package cartpoletest tests the Go bindings for cartpole.
package cartpoletest

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/gx-org/gx/api"
	"github.com/gx-org/gx/golang/binder/gobindings/types"
	"github.com/gx-org/gx/tests/bindings/cartpole/cartpole_go_gx"
)

var cartpole *cartpole_go_gx.Package

func wantState(t *testing.T, fn func() (types.Array[float32], error), want []float32) {
	state, err := fn()
	if err != nil {
		t.Fatalf("%+v", err)
	}
	hostState, err := state.Fetch()
	if err != nil {
		t.Fatalf("%+v", err)
	}
	got := hostState.CopyFlat()
	if !cmp.Equal(got, want) {
		t.Errorf("incorrect state: got %v but want %v", got, want)
	}
}

// TestStep runs a few cartpole steps.
func TestStep(t *testing.T) {
	wantInitialFullState := []float32{9.8, 1, 0.1, 1.1, 0.5, 0.05, 10, 0.02, 0.20943952, 2.4, 0, 0, 0, 0}
	wantStateAfterStep := []float32{0, 0.19512196, 0, -0.29268292}

	cp, err := cartpole.New.Run()
	if err != nil {
		t.Fatal(err)
	}
	wantState(t, cp.FullState().Run, wantInitialFullState)

	// Run a few step function.
	cp, err = cp.Step().Run(types.Float32(1))
	if err != nil {
		t.Fatal(err)
	}
	wantState(t, cp.State().Run, wantStateAfterStep)

	// Reset cartpole.
	cp, err = cp.Reset().Run()
	if err != nil {
		t.Fatal(err)
	}
	wantState(t, cp.FullState().Run, wantInitialFullState)

	// Run a few step function.
	cp, err = cp.Step().Run(types.Float32(1))
	if err != nil {
		t.Fatal(err)
	}
	wantState(t, cp.State().Run, wantStateAfterStep)
}

func setupTest(dev *api.Device) error {
	gxPackage, err := cartpole_go_gx.Load(dev.Runtime())
	if err != nil {
		return err
	}
	cartpole = gxPackage.BuildFor(dev)
	return nil
}
