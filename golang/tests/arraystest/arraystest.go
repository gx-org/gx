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

// Package arraystest tests the Go bindings of the arrays GX package.
package arraystest

import (
	"testing"

	"github.com/gx-org/gx/api"
	"github.com/gx-org/gx/golang/binder/gobindings/types"
	"github.com/gx-org/gx/tests/bindings/arrays/arrays_go_gx"
	gxtesting "github.com/gx-org/gx/tests/testing"
)

var arrays *arrays_go_gx.Package

func TestArgAsArrayIndex(t *testing.T) {
	scalar, err := arrays.ArgAsArrayIndex(types.Int(2))
	if err != nil {
		t.Fatalf("%+v", err)
	}
	got := gxtesting.FetchAtom(t, scalar)
	want := float32(5)
	if got != want {
		t.Errorf("got %f but want %f", got, want)
	}
}

func setupTest(dev *api.Device) error {
	var err error
	arrays, err = arrays_go_gx.BuildFor(dev)
	return err
}
