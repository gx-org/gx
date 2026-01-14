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

// Package notimp_test tests the Go backend for when a function is not implemented.
package notimp_test

import (
	"strings"
	"testing"

	"github.com/gx-org/gx/build/builder"
	"github.com/gx-org/gx/build/importers/embedpkg"
	"github.com/gx-org/gx/golang/backend"
	"github.com/gx-org/gx/golang/backend/tests/notimp/notimp_go_gx"
	"github.com/gx-org/gx/stdlib"
)

// ExampleRun runs GX linear regression codelab.
func TestNotImplemented(t *testing.T) {
	bck := backend.New(builder.New(
		stdlib.Importer(nil),
		embedpkg.New(),
	))
	dev, err := bck.Device(0)
	if err != nil {
		t.Fatal(err)
	}
	notimp, err := notimp_go_gx.BuildFor(dev)
	if err != nil {
		t.Fatal(err)
	}
	_, err = notimp.CallSum()
	got := err.Error()
	want := "function num.Sum has no implementation"
	if !strings.Contains(got, want) {
		t.Errorf("got error %q but want an error that contains %q", got, want)
	}
}
