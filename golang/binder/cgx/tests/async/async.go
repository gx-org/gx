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

// Package async tests cgx with concurrent calls.
package async

import (
	"math/rand"
	"sync"
	"testing"

	"github.com/gx-org/gx/api"
	"github.com/gx-org/gx/build/importers"
	"github.com/gx-org/gx/cgx/handle"
	cgxtesting "github.com/gx-org/gx/golang/binder/cgx/testing"
	"github.com/gx-org/gx/golang/binder/cgx/tests/gocgx"
	"github.com/gx-org/gx/golang/binder/gobindings/core"

	// Required for the C headers.
	_ "github.com/gx-org/gx/golang/binder/cgx"
)

// #cgo CFLAGS: -I ../../../../..
// #include <stdlib.h>
// #include <golang/binder/cgx/cgx.h>
// #include <golang/binder/cgx/cgx.cgo.h>
// #include <golang/binder/cgx/testing/testing.cgo.h>
import "C"

const (
	// Zero/empty value of cgx_handle, for convenience.
	empty = 0

	numGoRoutines = 10
	numTestCall   = 10
)

const (
	basicPackage     = "github.com/gx-org/gx/tests/bindings/basic"
	parameterPackage = "github.com/gx-org/gx/tests/bindings/parameters"
)

func clearLoaderCache(cdev gocgx.Device) {
	dev := handle.Unwrap[*core.DeviceSetup](handle.Handle(cdev))
	loader := dev.Runtime().Builder().Loader()
	cache, ok := loader.(*importers.CacheLoader)
	if !ok {
		return
	}
	cache.Clear()
}

func buildPackage(t *testing.T, rtm gocgx.Runtime, dev gocgx.Device, rnd *rand.Rand, path string) gocgx.Package {
	if rnd.Uint32()%10 > 0 {
		clearLoaderCache(dev)
	}
	pkg, err := gocgx.BuildPackage(gocgx.Runtime(rtm), gocgx.Device(dev), path)
	if err != nil {
		t.Fatal(err)
	}
	return pkg
}

func runFunctionReturnFloat32(t *testing.T, dev gocgx.Device, pkg gocgx.Package) {
	fn, err := gocgx.FindFunction(pkg, "ReturnFloat32")
	if err != nil {
		t.Error(err)
		return
	}
	defer C.cgx_release_reference(C.cgx_handle(fn))

	vals, err := gocgx.RunFunc(fn)
	if err != nil {
		t.Error(err)
		return
	}
	gocgx.ReleaseValues(vals)
}

func runAddToStruct(t *testing.T, dev gocgx.Device, pkg gocgx.Package) {
	newStruct, err := gocgx.FindFunction(pkg, "NewStruct")
	if err != nil {
		t.Error(err)
		return
	}
	defer C.cgx_release_reference(C.cgx_handle(newStruct))

	offsetR := C.cgx_value_new_float32(C.cgx_device(dev), 1.0)
	if offsetR.error != empty {
		t.Error(gocgx.GoError(gocgx.Error(offsetR.error)))
		return
	}
	defer C.cgx_release_reference(offsetR.value)

	addToStruct, err := gocgx.FindFunction(pkg, "AddToStruct")
	if err != nil {
		t.Error(err)
		return
	}
	defer C.cgx_release_reference(C.cgx_handle(addToStruct))

	vals, err := gocgx.RunFunc(newStruct, gocgx.Value(offsetR.value))
	if err != nil {
		t.Error(err)
		return
	}
	defer gocgx.ReleaseValues(vals)

	vals, err = gocgx.RunFunc(addToStruct, vals...)
	if err != nil {
		t.Error(err)
		return
	}
	defer gocgx.ReleaseValues(vals)
}

// TestCall is a test to run in one Go routine.
type TestCall struct {
	pkg  string
	test func(*testing.T, gocgx.Device, gocgx.Package)
}

var (
	// RunFunctionReturnFloat32 runs ReturnFloat32 in the basic package.
	RunFunctionReturnFloat32 = TestCall{
		pkg:  basicPackage,
		test: runFunctionReturnFloat32,
	}

	// RuntAddToStruct runs AddToStruct in the parameter package.
	RuntAddToStruct = TestCall{
		pkg:  parameterPackage,
		test: runAddToStruct,
	}

	allTests = []TestCall{
		RunFunctionReturnFloat32,
		RuntAddToStruct,
	}
)

func runCGXTest(t *testing.T, rtm gocgx.Runtime, dev gocgx.Device, wg *sync.WaitGroup, seed int, tests []TestCall) {
	defer wg.Done()
	rnd := rand.New(rand.NewSource(int64(seed)))
	for range numTestCall {
		test := tests[rnd.Intn(len(tests))]
		pkg := buildPackage(t, rtm, dev, rnd, test.pkg)
		if pkg == empty {
			continue
		}
		test.test(t, dev, pkg)
		C.cgx_release_reference(C.cgx_handle(pkg))
	}
}

// RunTestAsyncCGX tests concurrent access to CGX.
func RunTestAsyncCGX(t *testing.T, rtm *api.Runtime, testCalls ...TestCall) {
	if len(testCalls) == 0 {
		testCalls = allTests
	}
	defer cgxtesting.CheckHandleCount(t, handle.Count())
	crtm, dev, finalize, err := gocgx.Setup(rtm)
	if err != nil {
		t.Fatal(err)
	}
	defer finalize()
	wg := sync.WaitGroup{}
	for i := range numGoRoutines {
		wg.Add(1)
		go runCGXTest(t, crtm, dev, &wg, i, testCalls)
	}
	wg.Wait()
}
