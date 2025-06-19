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

// Package testing provide a cgx runtime for GX testing.
// The runtime includes GX test files statically linked in the binary.
package testing

import (
	"testing"

	"github.com/gx-org/gx/api"
	"github.com/gx-org/gx/build/builder"
	"github.com/gx-org/gx/golang/backend"
	"github.com/gx-org/gx/golang/binder/cgx"
	gxtesting "github.com/gx-org/gx/tests/testing"
)

// #include "third_party/gxlang/gx/golang/binder/cgx/cgx.h"
import "C"

//export cgx_testing_builder
func cgx_testing_builder() C.cgx_builder {
	return C.cgx_builder(cgx.Wrap[*builder.Builder](gxtesting.NewBuilderStaticSource(nil)))
}

//export cgx_testing_runtime
func cgx_testing_runtime() C.struct_cgx_runtime_new_result {
	bld := gxtesting.NewBuilderStaticSource(nil)
	rtm := backend.New(bld)
	return C.struct_cgx_runtime_new_result{
		runtime: C.cgx_runtime(cgx.Wrap[*api.Runtime](rtm)),
	}
}

// CheckHandleCount compares the current handle count to a reference.
// Signal a testing error if the two counts do not match.
func CheckHandleCount(t *testing.T, startCount int) {
	endCount := cgx.HandleCount()
	if endCount != startCount {
		t.Errorf("handles are leaking: started with %d and ended with %d\nActive handles:\n%s", startCount, endCount, cgx.HandleDump())
	}
}
