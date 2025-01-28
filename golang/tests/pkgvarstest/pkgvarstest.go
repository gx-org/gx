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

// Package pkgvarstest tests the Go bindings of the basic GX package.
package pkgvarstest

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/gx-org/gx/api"
	"github.com/gx-org/gx/tests/bindings/pkgvars/pkgvars_go_gx"
	gxtesting "github.com/gx-org/gx/tests/testing"
)

var pkgvars *pkgvars_go_gx.Package

var (
	var1 int32 = 5
	size int64 = 2
)

func TestReturnVar1(t *testing.T) {
	res, err := pkgvars.ReturnVar1.Run()
	if err != nil {
		t.Fatalf("%+v", err)
	}
	got := gxtesting.FetchAtom(t, res)
	if !cmp.Equal(got, var1) {
		t.Errorf("incorrect computation: got %T(%v) but want %T(%v)", got, got, var1, var1)
	}
}

func TestNewTwiceSize(t *testing.T) {
	res, err := pkgvars.NewTwiceSize.Run()
	if err != nil {
		t.Fatalf("%+v", err)
	}
	got := gxtesting.FetchArray(t, res)
	want := []float32{3, 3, 3, 3}
	if !cmp.Equal(got, want) {
		t.Errorf("incorrect computation: got %T(%v) but want %T(%v)", got, got, want, want)
	}
}

func setupTest(dev *api.Device) error {
	gxPackage, err := pkgvars_go_gx.Load(dev.Runtime())
	if err != nil {
		return err
	}
	pkgvars = gxPackage.BuildFor(
		dev,
		pkgvars_go_gx.Var1.Set(var1),
		pkgvars_go_gx.Size.Set(size),
	)
	return nil
}
