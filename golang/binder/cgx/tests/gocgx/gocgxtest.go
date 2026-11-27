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
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/gomlx/compute"
	"github.com/gomlx/compute/shapes"
	"github.com/gx-org/gx/api/hostio"
	"github.com/gx-org/gx/cgx/handle"
	cgxtesting "github.com/gx-org/gx/golang/binder/cgx/testing"

	// Required for the C headers.
	_ "github.com/gx-org/gx/golang/binder/cgx"
)

const genericsPackage = "github.com/gx-org/gx/tests/bindings/generics"

// TestGenericScalar tests calling a generic GX function.
// The test is not in a _test.go file because Go does not support CGo in test files.
func TestGenericScalar(t *testing.T) {
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

// TestGenericSlice tests calling a generic GX function.
// The test is not in a _test.go file because Go does not support CGo in test files.
func TestGenericSlice(t *testing.T) {
	const want = `DeviceArray{DeviceID: 0}: [2][3]float32{
	{0, 0, 0},
	{0, 0, 0},
}`

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
	arraySize, err := Send(dev, C.CGX_INT64, []int64{2, 3}, 2)
	if err != nil {
		t.Fatal(err)
	}
	defer C.cgx_release_reference(C.cgx_handle(arraySize))

	fn, err := FindFunction(genpkg, "ArrayF32")
	if err != nil {
		t.Fatal(err)
	}
	defer C.cgx_release_reference(C.cgx_handle(fn))

	// Run the function.
	vals, err := RunFunc(fn, arraySize)
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

// TestValueHostBufferToHostError tests that cgx_value_host_buffer returns a contextualized error
// when DeviceArray.ToHostArray fails.
// The test is not in a _test.go file because Go does not support CGo in test files.
func TestValueHostBufferToHostError(t *testing.T) {
	defer cgxtesting.CheckHandleCount(t, handle.Count())
	_, dev, finalize, err := Setup(nil)
	if err != nil {
		t.Fatal(err)
	}
	defer finalize()

	val := C.cgx_value_new_int64(C.cgx_device(dev), 42)
	if val.error != Null {
		t.Fatal(GoError(Error(val.error)))
	}
	defer C.cgx_release_reference(val.value)

	valGo, err := handle.Unwrap[hostio.Value](handle.Handle(val.value))
	if err != nil {
		t.Fatalf("Unwrap() unexpected error: %v", err)
	}
	if err := valGo.Finalize(); err != nil {
		t.Fatalf("Finalize() unexpected error: %v", err)
	}

	res := C.cgx_value_host_buffer(val.value)
	if res.buffer != Null {
		C.cgx_release_reference(C.cgx_handle(res.buffer))
		t.Fatalf("cgx_value_host_buffer(%v) buffer = %v, want %v", val.value, res.buffer, Null)
	}
	if res.error == Null {
		t.Fatalf("cgx_value_host_buffer(%v) error = %v, want non-zero handle", val.value, Null)
	}
	defer C.cgx_release_reference(C.cgx_handle(res.error))

	got := GoError(Error(res.error)).Error()
	wantSubstrings := []string{
		fmt.Sprintf("cgx_value_host_buffer: value of handle %d (type int64):", val.value),
		"finalized",
	}
	for _, want := range wantSubstrings {
		if !strings.Contains(got, want) {
			t.Errorf("cgx_value_host_buffer(%v) error = %q, want substring %q", val.value, got, want)
		}
	}
}

type failingTransferBuffer struct {
	compute.Buffer
	shapeErr    error
	transferErr error
}

func (b *failingTransferBuffer) Shape() (shapes.Shape, error) {
	if b.shapeErr != nil {
		return shapes.Shape{}, b.shapeErr
	}
	return b.Buffer.Shape()
}

func (b *failingTransferBuffer) ToFlatData(any) error {
	return b.transferErr
}

// TestValueGetFetchError tests that calling a scalar getter returns a contextualized error when
// querying the shape or fetching the value from the device fails.
// The test is not in a _test.go file because Go does not support CGo in test files.
func TestValueGetFetchError(t *testing.T) {
	defer cgxtesting.CheckHandleCount(t, handle.Count())
	_, dev, finalize, err := Setup(nil)
	if err != nil {
		t.Fatal(err)
	}
	defer finalize()

	val := C.cgx_value_new_int64(C.cgx_device(dev), 42)
	if val.error != Null {
		t.Fatal(GoError(Error(val.error)))
	}
	defer C.cgx_release_reference(val.value)

	valGo, err := handle.Unwrap[hostio.Value](handle.Handle(val.value))
	if err != nil {
		t.Fatalf("Unwrap() unexpected error: %v", err)
	}
	orig := valGo.(*hostio.DeviceArray)
	failingBuf := &failingTransferBuffer{Buffer: orig.Buffer()}
	failingArray, err := hostio.NewDeviceArray(orig.Type(), failingBuf)
	if err != nil {
		t.Fatalf("NewDeviceArray() unexpected error: %v", err)
	}
	failingHandle := C.cgx_value(handle.Wrap[hostio.Value](failingArray))
	defer C.cgx_release_reference(failingHandle)

	wantShapeErr := errors.New("device shape failed")
	failingBuf.shapeErr = wantShapeErr
	shapeRes := C.cgx_value_get_int64(failingHandle)
	if shapeRes.value != 0 {
		t.Errorf("cgx_value_get_int64(%v) value = %v, want 0", failingHandle, shapeRes.value)
	}
	if shapeRes.error == Null {
		t.Fatalf("cgx_value_get_int64(%v) error = %v, want non-zero handle", failingHandle, Null)
	}
	defer C.cgx_release_reference(C.cgx_handle(shapeRes.error))
	if got, want := GoError(Error(shapeRes.error)).Error(), fmt.Sprintf("cgx_value_get_int64: value of handle %d (type int64): %s", failingHandle, wantShapeErr); !strings.Contains(got, want) {
		t.Errorf("cgx_value_get_int64(%v) error = %q, want substring %q", failingHandle, got, want)
	}

	failingBuf.shapeErr = nil
	wantTransferErr := errors.New("device transfer failed")
	failingBuf.transferErr = wantTransferErr
	res := C.cgx_value_get_int64(failingHandle)
	if res.value != 0 {
		t.Errorf("cgx_value_get_int64(%v) value = %v, want 0", failingHandle, res.value)
	}
	if res.error == Null {
		t.Fatalf("cgx_value_get_int64(%v) error = %v, want non-zero handle", failingHandle, Null)
	}
	defer C.cgx_release_reference(C.cgx_handle(res.error))

	got := GoError(Error(res.error)).Error()
	want := fmt.Sprintf("cgx_value_get_int64: value of handle %d (type int64): %s", failingHandle, wantTransferErr)
	if !strings.Contains(got, want) {
		t.Errorf("cgx_value_get_int64(%v) error = %q, want substring %q", failingHandle, got, want)
	}
}

const (
	invalidHandlePanicCaseEnv = "GOCGX_INVALID_HANDLE_PANIC_CASE"
	invalidHandlePrefix       = "GOCGX_TEST_HANDLE="
)

func runInvalidHandlePanicCase(t *testing.T, panicCase string) {
	switch panicCase {
	case "zero_handle":
		C.cgx_value_kind_of(0)
	case "released_handle":
		_, dev, _, err := Setup(nil)
		if err != nil {
			t.Fatal(err)
		}
		val := C.cgx_value_new_int64(C.cgx_device(dev), 42)
		if val.error != Null {
			t.Fatal(GoError(Error(val.error)))
		}
		released := val.value
		C.cgx_release_reference(released)
		fmt.Printf("\n%s%d\n", invalidHandlePrefix, released)
		C.cgx_package_ir_build_for(0, C.cgx_device(released))
	case "wrong_type_handle":
		_, dev, _, err := Setup(nil)
		if err != nil {
			t.Fatal(err)
		}
		fmt.Printf("\n%s%d\n", invalidHandlePrefix, dev)
		C.cgx_value_kind_of(C.cgx_value(dev))
	default:
		t.Fatalf("unknown panic case %q", panicCase)
	}
	t.Fatalf("expected panic for %q, got none", panicCase)
}

func invalidHandleSubprocessEnv(panicCase string) []string {
	stripPrefixes := []string{
		"TEST_PREMATURE_EXIT_FILE=",
		"XML_OUTPUT_FILE=",
		"XML_EXTRA_PROPERTIES_FILE=",
		"GUNIT_OUTPUT=",
		"GO_TEST_CHATTY_OUTPUT=",
		"TESTBRIDGE_TEST_ONLY=",
		"TEST_SHARD_INDEX=",
		"TEST_TOTAL_SHARDS=",
		"TEST_SHARD_STATUS_FILE=",
		invalidHandlePanicCaseEnv + "=",
	}
	var env []string
	for _, entry := range os.Environ() {
		strip := false
		for _, prefix := range stripPrefixes {
			if strings.HasPrefix(entry, prefix) {
				strip = true
				break
			}
		}
		if !strip {
			env = append(env, entry)
		}
	}
	return append(env, invalidHandlePanicCaseEnv+"="+panicCase)
}

func extractTestHandleID(output string) (string, bool) {
	for _, line := range strings.Split(output, "\n") {
		if id, ok := strings.CutPrefix(line, invalidHandlePrefix); ok {
			return strings.TrimSpace(id), true
		}
	}
	return "", false
}

// TestInvalidHandlePanic tests that exports without an error result panic with a clear message when
// given a zero, released, or wrong-type handle.
// Each case runs in a subprocess so the unrecovered Go panic aborts the child process as in
// production instead of unwinding across a CGo frame into recover(), which skips CGo traceback
// context release.
// The test is not in a _test.go file because Go does not support CGo in test files.
func TestInvalidHandlePanic(t *testing.T) {
	if panicCase := os.Getenv(invalidHandlePanicCaseEnv); panicCase != "" {
		runInvalidHandlePanicCase(t, panicCase)
		return
	}

	exe, err := os.Executable()
	if err != nil {
		t.Fatalf("os.Executable() failed: %v", err)
	}
	testName := t.Name()

	for _, tc := range []struct {
		desc        string
		needHandle  bool
		wantSubstrs func(handleID string) []string
	}{
		{
			desc: "zero_handle",
			wantSubstrs: func(string) []string {
				return []string{
					"cgx_value_kind_of: invalid handle 0 (zero handle), want a handle to a hostio.Value",
				}
			},
		},
		{
			desc:       "released_handle",
			needHandle: true,
			wantSubstrs: func(handleID string) []string {
				return []string{
					fmt.Sprintf("cgx_package_ir_build_for: device: invalid handle %s (released or never created), want a handle to a *core.DeviceSetup", handleID),
				}
			},
		},
		{
			desc:       "wrong_type_handle",
			needHandle: true,
			wantSubstrs: func(handleID string) []string {
				return []string{
					fmt.Sprintf("cgx_value_kind_of: invalid handle %s:", handleID),
					"cannot convert *core.DeviceSetup to hostio.Value",
				}
			},
		},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			cmd := exec.Command(exe, fmt.Sprintf("-test.run=^%s$", testName))
			cmd.Env = invalidHandleSubprocessEnv(tc.desc)
			out, err := cmd.CombinedOutput()
			if err == nil {
				t.Fatalf("subprocess exited with status 0, want panic exit status 2\noutput:\n%s", out)
			}
			var exitErr *exec.ExitError
			if !errors.As(err, &exitErr) {
				t.Fatalf("subprocess failed with unexpected error %v\noutput:\n%s", err, out)
			}
			if got := exitErr.ExitCode(); got != 2 {
				t.Fatalf("subprocess exit code = %d, want 2 (Go unrecovered panic)\noutput:\n%s", got, out)
			}

			got := string(out)
			var handleID string
			if tc.needHandle {
				var ok bool
				handleID, ok = extractTestHandleID(got)
				if !ok || handleID == "" {
					t.Fatalf("missing %q line in subprocess output:\n%s", invalidHandlePrefix, got)
				}
			}
			for _, want := range tc.wantSubstrs(handleID) {
				if !strings.Contains(got, want) {
					t.Errorf("subprocess output missing substring %q\noutput:\n%s", want, got)
				}
			}
		})
	}
}
