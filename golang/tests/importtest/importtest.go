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

// Package importtest tests Go bindings of the import GX package.
package importtest

import (
	"testing"

	"github.com/gx-org/gx/api"
	"github.com/gx-org/gx/tests/bindings/imports/imports_go_gx"
	gxtesting "github.com/gx-org/gx/tests/testing"
)

var imports *imports_go_gx.Package

func TestImportsReturnScalar(t *testing.T) {
	scalar, err := imports.ReturnFromBasic.Run()
	if err != nil {
		t.Fatalf("%+v", err)
	}
	got := gxtesting.FetchAtom(t, scalar)
	want := float32(4.2)
	if got != want {
		t.Errorf("got %f but want %f", got, want)
	}
}

func TestAddPrivate(t *testing.T) {
	imp, err := imports.NewImporter.Run()
	if err != nil {
		t.Fatalf("%+v", err)
	}
	gotDevice, err := imp.Add().Run()
	if err != nil {
		t.Fatalf("%+v", err)
	}
	got := gxtesting.FetchAtom(t, gotDevice)
	want := int32(6)
	if got != want {
		t.Errorf("got %d but want %d", got, want)
	}
}

func setupTest(dev *api.Device) error {
	gxPackage, err := imports_go_gx.Load(dev.Runtime())
	if err != nil {
		return err
	}
	imports = gxPackage.BuildFor(dev)
	return nil
}
