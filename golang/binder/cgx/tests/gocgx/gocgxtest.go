// Copyright 2026 Google LLC
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

package gocgx

// #include <golang/binder/cgx/cgx.h>
// #include <golang/binder/cgx/cgx.cgo.h>
import "C"

import (
	"testing"

	"github.com/gx-org/gx/cgx/handle"
	cgxtesting "github.com/gx-org/gx/golang/binder/cgx/testing"

	// Required for the C headers.
	_ "github.com/gx-org/gx/golang/binder/cgx"
)

const genericsPackage = "github.com/gx-org/gx/tests/bindings/generics"

// TestGeneric tests calling a generic GX function.
// The test is not in a _test.go file because Go does not support CGo in test files.
func TestGeneric(t *testing.T) {
	const size = 3
	const want = "DeviceArray{DeviceID: 0}: [3]float32{0, 0, 0}"

	defer cgxtesting.CheckHandleCount(t, handle.Count())
	rtm, dev, finalize, err := Setup(nil)
	if err != nil {
		t.Fatal(err)
	}
	defer finalize()

	// Build the package.
	genpkg, err := BuildPackage(rtm, dev, genericsPackage)
	if err != nil {
		t.Fatal(err)
	}
	defer C.cgx_release_reference(C.cgx_handle(genpkg))

	// Create the argument value.
	vecSize := C.cgx_value_new_int64(C.cgx_device(dev), size)
	if vecSize.error != Null {
		t.Fatal(GoError(Error(vecSize.error)))
		return
	}
	defer C.cgx_release_reference(vecSize.value)

	fn, err := FindFunction(genpkg, "VecF32")
	if err != nil {
		t.Fatal(err)
	}
	defer C.cgx_release_reference(C.cgx_handle(fn))

	// Run the function.
	vals, err := RunFunc(fn, Value(vecSize.value))
	if err != nil {
		t.Fatal(err)
	}
	defer ReleaseValues(vals)

	// Compare the result.
	got := String(vals[0])
	if got != want {
		t.Errorf("got %s but want %s", got, want)
	}
}
