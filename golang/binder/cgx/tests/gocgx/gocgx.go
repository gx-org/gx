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

// Package gocgx runs tests calling cgx from Go.
package gocgx

// #cgo CFLAGS: -I ../../../../..
// #include <stdlib.h>
// #include <golang/binder/cgx/cgx.h>
// #include <golang/binder/cgx/cgx.cgo.h>
import "C"

import (
	"fmt"
	"runtime"
	"unsafe"

	"github.com/gx-org/gx/api"
	"github.com/gx-org/gx/build/builder"
	"github.com/gx-org/gx/build/importers/embedpkg"
	"github.com/gx-org/gx/cgx/handle"
	"github.com/gx-org/gx/golang/backend"
	"github.com/gx-org/gx/stdlib"

	// Required for the C headers.
	_ "github.com/gx-org/gx/golang/binder/cgx"
)

type (
	// Device is a C pointer to a device.
	Device uintptr
	// Error is a C pointer to an error.
	Error uintptr
	// Function is a C pointer to a function.
	Function uintptr
	// Package is a C pointer to a package.
	Package uintptr
	// Runtime is a C pointer to a runtime.
	Runtime uintptr
	// Shape is a C pointer to a shape.
	Shape uintptr
	// Value is a C pointer to a value.
	Value uintptr
)

// Null is the zero/empty value of cgx_handle, for convenience.
const Null = 0

// GoError converts a C error into a Go error.
// The C error is free.
func GoError(err Error) error {
	cmsg := C.cgx_error_message(C.cgx_error(err))
	msg := C.GoString(cmsg)
	C.free(unsafe.Pointer(cmsg))
	return fmt.Errorf("%s", msg)
}

// NewBuilder returns a default GX builder.
func NewBuilder() *builder.Builder {
	return builder.New(
		stdlib.Importer(),
		embedpkg.New(),
	)

}

// Setup the runtime.
func Setup(rtm *api.Runtime) (Runtime, Device, func(), error) {
	C.cgx_error_verbose(true)
	ownRuntime := false
	if rtm == nil {
		var err error
		rtm, err = backend.New(NewBuilder())
		if err != nil {
			return Null, Null, nil, err
		}
		ownRuntime = true
	}
	if C.cgx_handle_count() != 0 {
		return Null, Null, nil, fmt.Errorf("memory handle count is not 0")
	}
	crtm := C.cgx_runtime(handle.Wrap[*api.Runtime](rtm))
	deviceResult := C.cgx_device_get(crtm, 0)
	if deviceResult.error != Null {
		return Null, Null, nil, GoError(Error(deviceResult.error))
	}
	finalize := func() {
		C.cgx_release_reference(C.cgx_handle(deviceResult.device))
		C.cgx_release_reference(C.cgx_handle(crtm))
		if ownRuntime {
			rtm.Backend().Finalize()
		}
		runtime.GC()
	}
	return Runtime(crtm), Device(deviceResult.device), finalize, nil
}

// BuildPackage builds a package via the CGX API.
func BuildPackage(rtm Runtime, dev Device, path string) (Package, error) {
	cpath := C.CString(path)
	defer C.free(unsafe.Pointer(cpath))
	pr := C.cgx_package_ir_load(C.cgx_runtime(rtm), cpath)
	defer C.cgx_release_reference(pr._package)
	if pr.error != Null {
		return Null, GoError(Error(pr.error))
	}
	return Package(C.cgx_package_ir_build_for(C.cgx_package(pr._package), C.cgx_device(dev))), nil
}

// FindFunction finds a function via the CGX API.
func FindFunction(pkg Package, name string) (Function, error) {
	cname := C.CString(name)
	defer C.free(unsafe.Pointer(cname))
	fr := C.cgx_function_find(C.cgx_package(pkg), cname)
	if fr.error != Null {
		return Null, GoError(Error(fr.error))
	}
	return Function(fr.function), nil
}

// RunFunc runs a function via the CGX API.
func RunFunc(fn Function, args ...Value) ([]Value, error) {
	var pargs *C.cgx_value
	if len(args) > 0 {
		cgxargs := make([]C.cgx_value, len(args))
		for i, arg := range args {
			cgxargs[i] = C.cgx_value(arg)
		}
		cargs := unsafe.SliceData(cgxargs)
		pinner := runtime.Pinner{}
		pinner.Pin(cargs)
		defer pinner.Unpin()
		pargs = (*C.cgx_value)(cargs)
	}
	r := C.cgx_function_run(C.cgx_function(fn), Null, C.int(len(args)), pargs)
	if r.error != Null {
		return nil, GoError(Error(r.error))
	}
	cvals := unsafe.Slice(r.values, r.value_size)
	vals := make([]Value, len(cvals))
	for i, val := range cvals {
		vals[i] = Value(val)
	}
	C.cgx_free_function_run_result(&r)
	return vals, nil
}

// NewShape creates a new shape via the CGX API.
func NewShape(dtype uint32, axes ...int64) Shape {
	var paxes *C.cint64_t
	if len(axes) > 0 {
		caxes := unsafe.SliceData(axes)
		pinner := runtime.Pinner{}
		pinner.Pin(caxes)
		defer pinner.Unpin()
		paxes = (*C.cint64_t)(caxes)
	}
	return Shape(C.cgx_shape_new(C.enum_cgx_value_kind(dtype), paxes, C.int(len(axes))))
}

// Send a slice of values to the device via the CGX API.
func Send[T any](dev Device, dtype uint32, data []T, axes ...int64) (Value, error) {
	shp := NewShape(dtype, axes...)
	defer C.cgx_release_reference(C.cgx_handle(shp))
	var pdata unsafe.Pointer
	var byteSize int
	if len(data) > 0 {
		cdata := unsafe.SliceData(data)
		pinner := runtime.Pinner{}
		pinner.Pin(cdata)
		defer pinner.Unpin()
		pdata = unsafe.Pointer(cdata)
		byteSize = len(data) * int(unsafe.Sizeof(data[0]))
	}
	vr := C.cgx_value_send(C.cgx_device(dev), C.cgx_shape(shp), pdata, C.uint64_t(byteSize))
	if vr.error != Null {
		return Null, GoError(Error(vr.error))
	}
	return Value(vr.value), nil
}

// ReleaseValues releases all the value handles.
func ReleaseValues(vals []Value) {
	defer C.cgx_release_references((*C.cgx_handle)(unsafe.Pointer(&vals[0])), C.uint32_t(len(vals)))
}

// String representation of a value.
func String(val Value) string {
	cstr := C.cgx_value_string(C.cgx_value(val))
	s := C.GoString(cstr)
	C.free(unsafe.Pointer(cstr))
	return s
}
