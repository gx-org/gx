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
	"fmt"
	"math/rand"
	"runtime"
	"sync"
	"testing"
	"unsafe"

	"github.com/gx-org/gx/api"
	"github.com/gx-org/gx/build/builder"
	"github.com/gx-org/gx/build/importers/embedpkg"
	"github.com/gx-org/gx/build/importers"
	"github.com/gx-org/gx/cgx/handle"
	cgxtesting "github.com/gx-org/gx/golang/binder/cgx/testing"
	"github.com/gx-org/gx/stdlib"

	_ "github.com/gx-org/gx/golang/binder/cgx" // C dependency
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

func cError(err C.cgx_error) error {
	cmsg := C.cgx_error_debug_message(err)
	msg := C.GoString(cmsg)
	C.free(unsafe.Pointer(cmsg))
	return fmt.Errorf("%s", msg)
}

const (
	basicPackage     = "github.com/gx-org/gx/tests/bindings/basic"
	parameterPackage = "github.com/gx-org/gx/tests/bindings/parameters"
)

func setup(rtm *api.Runtime) (C.cgx_runtime, C.cgx_device, error) {
	if C.cgx_handle_count() != 0 {
		return empty, empty, fmt.Errorf("memory handle count is not 0")
	}
	crtm := C.cgx_runtime(handle.Wrap[*api.Runtime](rtm))
	deviceResult := C.cgx_device_get(crtm, 0)
	if deviceResult.error != empty {
		return empty, empty, cError(deviceResult.error)
	}
	return crtm, deviceResult.device, nil
}

// NewBuilder returns a builder that can used for the tests.
func NewBuilder() *builder.Builder {
	return builder.New(importers.NewCacheLoader(
		stdlib.Importer(nil),
		embedpkg.New(),
	))
}

func clearLoaderCache(cdev C.cgx_device) {
	dev := handle.Unwrap[*api.Device](handle.Handle(cdev))
	loader := dev.Runtime().Builder().Loader()
	cache, ok := loader.(*importers.CacheLoader)
	if !ok {
		return
	}
	cache.Clear()
}

func buildPackage(t *testing.T, rtm C.cgx_runtime, dev C.cgx_device, rnd *rand.Rand, path string) C.cgx_package {
	if rnd.Uint32()%10 > 0 {
		clearLoaderCache(dev)
	}

	cpath := C.CString(path)
	pr := C.cgx_package_ir_load(rtm, cpath)
	defer C.cgx_release_reference(pr._package)
	if pr.error != empty {
		t.Error(cError(pr.error))
		return empty
	}
	return C.cgx_package_ir_build_for(pr._package, dev)
}

func findFunction(t *testing.T, pkg C.cgx_package, name string) C.cgx_function {
	cname := C.CString(name)
	defer C.free(unsafe.Pointer(cname))
	fr := C.cgx_function_find(pkg, cname)
	if fr.error != empty {
		t.Error(cError(fr.error))
		return empty
	}
	return fr.function
}

func functionRun(t *testing.T, fn C.cgx_function, args ...C.cgx_value) []C.cgx_value {
	var pargs *C.cgx_value
	if len(args) > 0 {
		cargs := unsafe.SliceData(args)
		pinner := runtime.Pinner{}
		pinner.Pin(cargs)
		defer pinner.Unpin()
		pargs = (*C.cgx_value)(cargs)
	}
	r := C.cgx_function_run(fn, empty, C.int(len(args)), pargs)
	if r.error != empty {
		t.Error(cError(r.error))
		return nil
	}
	vals := unsafe.Slice(r.values, r.value_size)
	C.cgx_free_function_run_result(&r)
	return vals
}

func releaseValues(vals []C.cgx_value) {
	defer C.cgx_release_references((*C.cgx_handle)(unsafe.Pointer(&vals[0])), C.uint32_t(len(vals)))
}

func runFunctionReturnFloat32(t *testing.T, dev C.cgx_device, pkg C.cgx_package) {
	fn := findFunction(t, pkg, "ReturnFloat32")
	if fn == empty {
		return
	}
	defer C.cgx_release_reference(fn)

	vals := functionRun(t, fn)
	if vals == nil {
		return
	}
	releaseValues(vals)
}

func runAddToStruct(t *testing.T, dev C.cgx_device, pkg C.cgx_package) {
	newStruct := findFunction(t, pkg, "NewStruct")
	if newStruct == empty {
		return
	}
	defer C.cgx_release_reference(newStruct)

	offsetR := C.cgx_value_new_float32(dev, 1.0)
	if offsetR.error != empty {
		t.Error(cError(offsetR.error))
		return
	}
	defer C.cgx_release_reference(offsetR.value)

	addToStruct := findFunction(t, pkg, "AddToStruct")
	if addToStruct == empty {
		return
	}
	defer C.cgx_release_reference(addToStruct)

	vals := functionRun(t, newStruct, offsetR.value)
	if vals == nil {
		return
	}
	defer releaseValues(vals)

	vals = functionRun(t, addToStruct, vals...)
	if vals == nil {
		return
	}
	defer releaseValues(vals)
}

// TestCall is a test to run in one Go routine.
type TestCall struct {
	pkg  string
	test func(*testing.T, C.cgx_device, C.cgx_package)
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

func runCGXTest(t *testing.T, rtm C.cgx_runtime, dev C.cgx_device, wg *sync.WaitGroup, seed int, tests []TestCall) {
	defer wg.Done()
	rnd := rand.New(rand.NewSource(int64(seed)))
	for range numTestCall {
		test := tests[rnd.Intn(len(tests))]
		pkg := buildPackage(t, rtm, dev, rnd, test.pkg)
		if pkg == empty {
			continue
		}
		test.test(t, dev, pkg)
		C.cgx_release_reference(pkg)
	}
}

// RunTestAsyncCGX tests concurrent access to CGX.
func RunTestAsyncCGX(t *testing.T, rtm *api.Runtime, testCalls ...TestCall) {
	if len(testCalls) == 0 {
		testCalls = allTests
	}
	defer cgxtesting.CheckHandleCount(t, handle.Count())
	crtm, dev, err := setup(rtm)
	if err != nil {
		t.Fatal(err)
	}
	defer C.cgx_release_reference(dev)
	defer C.cgx_release_reference(crtm)
	wg := sync.WaitGroup{}
	for i := range numGoRoutines {
		wg.Add(1)
		go runCGXTest(t, crtm, dev, &wg, i, testCalls)
	}
	wg.Wait()
}
