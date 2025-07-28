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

package fmtpath_test

import (
	"testing"

	"github.com/gx-org/gx/golang/binder/ccbindings/fmtpath"
)

const pkgPath = "github.com/gx-org/some/package/somewhere"

func init() {
	fmtpath.SetModuleName("github.com/gx-org/some")
}

func TestHeaderPath(t *testing.T) {
	got, err := fmtpath.HeaderPath(pkgPath)
	if err != nil {
		t.Fatal(err)
	}
	want := "somewhere.h"
	if got != want {
		t.Errorf("incorrect header path: got %s but want %s", got, want)
	}
}

func TestHeaderGuard(t *testing.T) {
	got, err := fmtpath.HeaderGuard(pkgPath)
	if err != nil {
		t.Fatal(err)
	}
	want := "PACKAGE_SOMEWHERE_H"
	if got != want {
		t.Errorf("incorrect header path: got %s but want %s", got, want)
	}
}

func TestNamespace(t *testing.T) {
	got, err := fmtpath.Namespace(pkgPath)
	if err != nil {
		t.Fatal(err)
	}
	want := "package::somewhere"
	if got != want {
		t.Errorf("incorrect header path: got %s but want %s", got, want)
	}
}
