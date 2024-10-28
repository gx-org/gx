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

// Package basictest tests the Go bindings of the basic GX package.
package basictest

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/gx-org/gx/api"
	"github.com/gx-org/gx/golang/binder/gobindings/types"
	"github.com/gx-org/gx/tests/bindings/basic/basic_go_gx"
	gxtesting "github.com/gx-org/gx/tests/testing"
)

var basic *basic_go_gx.Compiler

func TestReturnFloat32(t *testing.T) {
	scalar, err := basic.ReturnFloat32.Run()
	if err != nil {
		t.Fatalf("%+v", err)
	}
	got := gxtesting.FetchAtom(t, scalar)
	want := float32(4.2)
	if got != want {
		t.Errorf("got %f but want %f", got, want)
	}
}

func TestReturnTensorFloat32(t *testing.T) {
	array, err := basic.ReturnArrayFloat32.Run()
	if err != nil {
		t.Fatalf("%+v", err)
	}
	got := gxtesting.FetchArray(t, array)
	want := []float32{4.2, 42}
	if !cmp.Equal(got, want) {
		t.Errorf("got %v but want %v", got, want)
	}
}

func TestNew(t *testing.T) {
	bsc, err := basic.New.Run()
	if err != nil {
		t.Fatalf("%+v", err)
	}

	intGot := gxtesting.FetchAtom(t, bsc.Int)
	intWant := int32(42)
	if intGot != intWant {
		t.Errorf("got %d but want %d", intGot, intWant)
	}

	floatGot := gxtesting.FetchAtom(t, bsc.Float)
	floatWant := float32(4.2)
	if floatGot != floatWant {
		t.Errorf("got %f but want %f", floatGot, floatWant)
	}

	arrayGot := gxtesting.FetchArray(t, bsc.Array)
	arrayWant := []float32{4.2, 42}
	if !cmp.Equal(arrayGot, arrayWant) {
		t.Errorf("got %v but want %v", arrayGot, arrayWant)
	}
}

func TestAddPrivatePackageLevel(t *testing.T) {
	bsc, err := basic.New.Run()
	if err != nil {
		t.Fatalf("%+v", err)
	}
	intDevice, err := basic.AddPrivate.Run(bsc)
	if err != nil {
		t.Fatalf("%+v", err)
	}
	intGot := gxtesting.FetchAtom(t, intDevice)
	intWant := int32(6)
	if intGot != intWant {
		t.Errorf("got %d but want %d", intGot, intWant)
	}
}

func TestAddPrivateStructLevel(t *testing.T) {
	bsc, err := basic.New.Run()
	if err != nil {
		t.Fatalf("%+v", err)
	}
	intDevice, err := bsc.AddPrivate().Run()
	if err != nil {
		t.Fatalf("%+v", err)
	}
	intGot := gxtesting.FetchAtom(t, intDevice)
	intWant := int32(6)
	if intGot != intWant {
		t.Errorf("got %d but want %d", intGot, intWant)
	}
}

func TestSetFloat(t *testing.T) {
	bsc, err := basic.New.Run()
	if err != nil {
		t.Fatalf("%+v", err)
	}
	var want float32 = 4.5
	bsc, err = bsc.SetFloat().Run(types.Float32(want))
	if err != nil {
		t.Fatalf("%+v", err)
	}
	got := gxtesting.FetchAtom(t, bsc.Float)
	if got != want {
		t.Errorf("got %f but want %f", got, want)
	}
}

func setupTest(rtm *api.Runtime) error {
	gxPackage, err := basic_go_gx.Load(rtm)
	if err != nil {
		return err
	}
	dev, err := gxPackage.Runtime.Platform().Device(0)
	if err != nil {
		return err
	}
	basic = gxPackage.CompilerFor(dev)
	return nil
}
