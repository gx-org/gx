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

// Package mathtest tests constants in the math package.
package mathtest

import (
	"math"
	"testing"

	"github.com/gx-org/gx/api"
	"github.com/gx-org/gx/tests/bindings/math/math_go_gx"
	gxtesting "github.com/gx-org/gx/tests/testing"
)

var mathGX *math_go_gx.Compiler

func TestMathFloat32(t *testing.T) {
	scalar, err := mathGX.ReturnMaxFloat32.Run()
	if err != nil {
		t.Fatalf("%+v", err)
	}
	got := gxtesting.FetchAtom(t, scalar)
	want := float32(math.MaxFloat32)
	if got != want {
		t.Errorf("got %f but want %f", got, want)
	}
}

func TestMathFloat64(t *testing.T) {
	scalar, err := mathGX.ReturnMaxFloat64.Run()
	if err != nil {
		t.Fatalf("%+v", err)
	}
	got := gxtesting.FetchAtom(t, scalar)
	want := float64(math.MaxFloat64)
	if got != want {
		t.Errorf("got %f but want %f", got, want)
	}
}

func setupTest(rtm *api.Runtime) error {
	gxPackage, err := math_go_gx.Load(rtm)
	if err != nil {
		return err
	}
	dev, err := gxPackage.Runtime.Platform().Device(0)
	if err != nil {
		return err
	}
	mathGX = gxPackage.CompilerFor(dev)
	return nil
}
