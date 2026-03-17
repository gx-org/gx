// Copyright 2025 Google LLC
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

// Package interperr_test tests the Go backend for when a function is not implemented.
package interperr_test

import (
	"strings"
	"testing"

	"github.com/gx-org/gx/build/builder"
	"github.com/gx-org/gx/build/importers/embedpkg"
	"github.com/gx-org/gx/golang/backend"
	"github.com/gx-org/gx/golang/backend/tests/interperr/interperr_go_gx"
	"github.com/gx-org/gx/stdlib"
)

func buildInterperr() (*interperr_go_gx.Package, error) {
	bck := backend.New(builder.New(
		stdlib.Importer(nil),
		embedpkg.New(),
	))
	dev, err := bck.Device(0)
	if err != nil {
		return nil, err
	}
	return interperr_go_gx.BuildFor(dev)
}

// TestNotImplemented tests calling a function in the standard library which does not have an implementation.
func TestNotImplemented(t *testing.T) {
	interperr, err := buildInterperr()
	if err != nil {
		t.Fatal(t)
	}
	_, err = interperr.CallSum()
	got := err.Error()
	want := "function num.Sum has no implementation"
	if !strings.Contains(got, want) {
		t.Errorf("got error %q but want an error that contains %q", got, want)
	}
}

// TestUndef tests calling a function that use a static variable that has not been defined by the host.
func TestUndef(t *testing.T) {
	interperr, err := buildInterperr()
	if err != nil {
		t.Fatal(t)
	}
	_, err = interperr.NewUndef()
	got := err.Error()
	want := "undefined: Undef"
	if !strings.Contains(got, want) {
		t.Errorf("got error %q but want an error that contains %q", got, want)
	}
}
