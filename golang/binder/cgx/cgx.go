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

// Package cgx provides an interface for calling into GX from C.
package cgx

import (
	"fmt"
	"unsafe"

	"github.com/pkg/errors"
	"github.com/gx-org/backend/dtype"
	"github.com/gx-org/backend/platform"
	"github.com/gx-org/backend/shape"
	"github.com/gx-org/gx/api"
	"github.com/gx-org/gx/api/options"
	"github.com/gx-org/gx/api/values"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/cgx/handle"
	"github.com/gx-org/gx/golang/backend/kernels"
	"github.com/gx-org/gx/golang/binder/gobindings/core"
	"github.com/gx-org/gx/golang/binder/gobindings/types"
)

/*
#cgo CFLAGS: -I ../../..

#include <stdbool.h>
#include <stdint.h>
#include <stdlib.h>

#include <golang/binder/cgx/cgx.h>

// cgx_device_get_result is the return value for cgx_device_get().
struct cgx_device_get_result {
	cgx_device device;
	cgx_error error;
};

// cgx_list_statics_result is the return value when listing static variables of a package.
struct cgx_list_statics_result {
	cgx_static* statics;
	int num_statics;
	cgx_error error;
};

// cgx_list_functions_result is the return value when listing functions of a GX element.
struct cgx_list_functions_result {
	cgx_function* funcs;
	int num_functions;
	cgx_error error;
};

// cgx_function_signature_element describes a function parameter or return value.
struct cgx_function_signature_element {
	const char* name;
	enum cgx_value_kind kind;
};

// cgx_function_signature_result is the return value for cgx_function_signature().
struct cgx_function_signature_result {
	struct cgx_function_signature_element* parameter;
	uint32_t parameter_size;
	struct cgx_function_signature_element* result;
	uint32_t result_size;

	cgx_error error;
};

// cgx_static_find_result is the return value for cgx_static_find().
struct cgx_static_find_result {
	cgx_static static_var;
	cgx_error error;
};

// cgx_interface_find_result is the return value for cgx_interface_find().
struct cgx_interface_find_result {
	cgx_interface iface;
	cgx_error error;
};

// cgx_function_find_result is the return value for cgx_function_find().
struct cgx_function_find_result {
	cgx_function function;
	cgx_error error;
};

// cgx_function_run_result is the return value for cgx_function_run().
struct cgx_function_run_result {
	cgx_value* values;
	uint32_t value_size;
	cgx_error error;
};

// cgx_value_new_result is the return value for cgx_value_new_*().
struct cgx_value_new_result {
	cgx_value value;
	cgx_error error;
};

// cgx_value_host_buffer_result is the return value for cgx_value_host_buffer().
struct cgx_value_host_buffer_result {
	cgx_host_buffer buffer;
	cgx_error error;
};

// cgx_value_get_struct_result is the return value for cgx_value_get_struct.
struct cgx_value_get_struct_result {
	cgx_struct strct;
	cgx_error error;
};

// cgx_struct_field_element describes a structure field.
struct cgx_struct_field_element {
	const char* name;
	enum cgx_value_kind kind;
};

// cgx_struct_field_list_result is the return value for cgx_struct_field_list().
struct cgx_struct_field_list_result {
	struct cgx_struct_field_element* field;
	uint32_t field_size;

	cgx_error error;
};

// cgx_shape_axes_result is the return value for cgx_shape_axes().
struct cgx_shape_axes_result {
	const int64_t* axis_lengths;
	uint32_t num_axes;
	cgx_error error;
};
*/
import "C"

// Since uintptr cannot be negative, each of these statements will trigger a compile-time error iff
// the first type is smaller than the second type.
const (
	// Ensure C.int64_t and Go int are the same size.
	_ = unsafe.Sizeof(C.int64_t(0)) - unsafe.Sizeof(int(0))
	_ = unsafe.Sizeof(int(0)) - unsafe.Sizeof(C.int64_t(0))

	// Ensure C.int64_t and Go uintptr are the same size.
	_ = unsafe.Sizeof(C.int64_t(0)) - unsafe.Sizeof(uintptr(0))
	_ = unsafe.Sizeof(uintptr(0)) - unsafe.Sizeof(C.int64_t(0))
)

//export cgx_release_reference
func cgx_release_reference(h C.cgx_handle) uintptr {
	handle.Release(handle.Handle(h))
	return 0
}

//export cgx_release_references
func cgx_release_references(ptr *C.cgx_handle, size C.uint32_t) uintptr {
	refs := unsafe.Slice(ptr, size)
	for i, ref := range refs {
		cgx_release_reference(ref)
		refs[i] = 0
	}
	return 0
}

/* Helper local functions */

func wrap[T comparable](v T) C.cgx_handle {
	return C.cgx_handle(handle.Wrap[T](v))
}

func unwrap[T any](h C.cgx_handle) T {
	return handle.Unwrap[T](handle.Handle(h))
}

// copyPrimitiveSlice returns a copy of the slice described by `data` and `size`.
func copyPrimitiveSlice[T any](data unsafe.Pointer, size int) []T {
	return append([]T{}, unsafe.Slice((*T)(data), size)...)
}

/* cgx_handle */

// cgx_handle_count returns the number of outstanding handles.
//
// For testing only.
//
//export cgx_handle_count
func cgx_handle_count() C.int64_t {
	return C.int64_t(handle.Count())
}

// cgx_handle_dump returns a full list of all outstanding handles.
//
// For testing only.
//
//export cgx_handle_dump
func cgx_handle_dump() *C.cchar_t {
	return C.CString(handle.Dump())
}

/* cgx_error */

// Errorf formats according to a format specifier and returns the new error via cgx_error handle.
func Errorf(fmt string, args ...any) C.cgx_error {
	return (C.cgx_error)(wrap[error](errors.Errorf(fmt, args...)))
}

//export cgx_error_message
func cgx_error_message(cgxError C.cgx_error) *C.cchar_t {
	err := unwrap[error](cgxError)
	return C.CString(err.Error())
}

//export cgx_error_debug_message
func cgx_error_debug_message(cgxError C.cgx_error) *C.cchar_t {
	err := unwrap[error](cgxError)
	return C.CString(fmt.Sprintf("%+v\n", err))
}

/* cgx_device */

//export cgx_device_get
func cgx_device_get(cgxRuntime C.cgx_runtime, deviceIdx int) C.struct_cgx_device_get_result {
	rtm := unwrap[*api.Runtime](cgxRuntime)
	dev, err := rtm.Device(deviceIdx)
	if err != nil {
		return C.struct_cgx_device_get_result{
			error: (C.cgx_error)(wrap[error](err)),
		}
	}
	return C.struct_cgx_device_get_result{
		device: (C.cgx_device)(wrap[*api.Device](dev)),
	}
}

//export cgx_device_get_runtime
func cgx_device_get_runtime(cgxDevice C.cgx_device) C.cgx_runtime {
	dev := unwrap[*api.Device](cgxDevice)
	return (C.cgx_runtime)(wrap[*api.Runtime](dev.Runtime()))
}

/* cgx_package_ir */

//export cgx_package_ir_load
func cgx_package_ir_load(cgxRuntime C.cgx_runtime, pathPtr *C.cchar_t) C.struct_cgx_package_ir_load_result {
	rtm := unwrap[*api.Runtime](cgxRuntime)
	pkg, err := core.BuildPackage(rtm.Builder(), C.GoString(pathPtr))
	return C.struct_cgx_package_ir_load_result{
		error:    (C.cgx_error)(wrap[error](err)),
		_package: (C.cgx_package_ir)(wrap[*core.Package](pkg)),
	}
}

/* cgx_package */

// NewPackageHandle returns a new C handle to compile a package for a runtime and a device.
// The returned value always has type C.cgx_package.
func NewPackageHandle(dev *api.Device, pkg *core.Package) C.cgx_handle {
	return wrap[*core.PackageCompileSetup](pkg.Setup(dev, nil))
}

//export cgx_package_ir_build_for
func cgx_package_ir_build_for(cgxPackageIR C.cgx_package_ir, cgxDevice C.cgx_device) C.cgx_package {
	dev := unwrap[*api.Device](cgxDevice)
	pkg := unwrap[*core.Package](cgxPackageIR)
	return C.cgx_package(NewPackageHandle(dev, pkg))
}

//export cgx_package_ir_name
func cgx_package_ir_name(cgxPackageIR C.cgx_package_ir) *C.cchar_t {
	pkg := unwrap[*core.Package](cgxPackageIR)
	return C.CString(pkg.Package().Name.Name)
}

//export cgx_package_ir_fullname
func cgx_package_ir_fullname(cgxPackageIR C.cgx_package_ir) *C.cchar_t {
	pkg := unwrap[*core.Package](cgxPackageIR)
	return C.CString(pkg.Package().FullName())
}

type packageOption struct {
	opt options.PackageOptionFactory
}

//export cgx_package_append_option
func cgx_package_append_option(cgxPackage C.cgx_package, cgxOption C.cgx_package_option) {
	cpkg := unwrap[*core.PackageCompileSetup](cgxPackage)
	copt := unwrap[*packageOption](cgxOption)
	cpkg.AppendOptions(copt.opt)
}

//export cgx_package_set_static
func cgx_package_set_static(cgxPackage C.cgx_package, staticNamePtr *C.cchar_t, cgxValue C.cgx_value) {
	cpkg := unwrap[*core.PackageCompileSetup](cgxPackage)
	cpkg.AppendOptions(func(plat platform.Platform) options.PackageOption {
		return options.PackageVarSetValue{
			Pkg:   cpkg.IR().FullName(),
			Var:   C.GoString(staticNamePtr),
			Value: unwrap[values.Value](cgxValue),
		}
	})
}

//export cgx_package_list_statics
func cgx_package_list_statics(cgxPackage C.cgx_package) C.struct_cgx_list_statics_result {
	cpkg := unwrap[*core.PackageCompileSetup](cgxPackage)
	statics := cpkg.IR().ExportedStatics()
	return C.struct_cgx_list_statics_result{
		statics:     (*C.cgx_function)(handle.PinSliceData(handle.WrapSlice(statics))),
		num_statics: C.int(len(statics)),
	}
}

func findStatic(cgxPackage C.cgx_package, staticNamePtr *C.cchar_t) (*ir.Package, string, *ir.VarExpr) {
	cpkg := unwrap[*core.PackageCompileSetup](cgxPackage)
	name := C.GoString(staticNamePtr)
	irPkg := cpkg.IR()
	for _, vr := range irPkg.ExportedStatics() {
		if vr.VName.Name == name {
			return irPkg, name, vr
		}
	}
	return irPkg, name, nil
}

//export cgx_static_has
func cgx_static_has(cgxPackage C.cgx_package, staticNamePtr *C.cchar_t) bool {
	_, _, vr := findStatic(cgxPackage, staticNamePtr)
	return vr != nil
}

//export cgx_static_find
func cgx_static_find(cgxPackage C.cgx_package, staticNamePtr *C.cchar_t) (res C.struct_cgx_static_find_result) {
	pkg, name, vr := findStatic(cgxPackage, staticNamePtr)
	if vr == nil {
		res.error = Errorf("static variable %s not found in package %s", name, pkg.Name.String())
		return
	}
	if !ir.IsExported(name) {
		res.error = Errorf("static variable %s not exported", vr.VName.Name)
		return
	}
	res.static_var = wrap(vr)
	return
}

//export cgx_static_name
func cgx_static_name(cgxStatic C.cgx_static) *C.cchar_t {
	vr := unwrap[*ir.VarExpr](cgxStatic)
	return C.CString(vr.VName.Name)
}

//export cgx_free_list_statics_result
func cgx_free_list_statics_result(res *C.struct_cgx_list_statics_result) {
	handle.UnpinSliceData(unsafe.Pointer(res.statics))
	res.statics = nil
	res.num_statics = 0
}

//export cgx_package_list_functions
func cgx_package_list_functions(cgxPackage C.cgx_package) C.struct_cgx_list_functions_result {
	cpkg := unwrap[*core.PackageCompileSetup](cgxPackage)
	var funcs []*core.FuncCache
	for fn := range cpkg.IR().ExportedFuncs() {
		fnDecl, isDecl := fn.(*ir.FuncDecl)
		if !isDecl {
			continue
		}
		funcs = append(funcs, cpkg.NewCacheFromFunc(fnDecl))
	}
	return C.struct_cgx_list_functions_result{
		funcs:         (*C.cgx_function)(handle.PinSliceData(handle.WrapSlice(funcs))),
		num_functions: C.int(len(funcs)),
	}
}

//export cgx_free_list_functions_result
func cgx_free_list_functions_result(res *C.struct_cgx_list_functions_result) {
	handle.UnpinSliceData(unsafe.Pointer(res.funcs))
	res.funcs = nil
	res.num_functions = 0
}

//export cgx_package_get_ir
func cgx_package_get_ir(cgxPackage C.cgx_package) C.cgx_package_ir {
	cpkg := unwrap[*core.PackageCompileSetup](cgxPackage)
	return (C.cgx_package_ir)(wrap[*core.Package](cpkg.Package()))
}

//export cgx_interface_find
func cgx_interface_find(cgxPackage C.cgx_package, cname *C.cchar_t) (res C.struct_cgx_interface_find_result) {
	cpkg := unwrap[*core.PackageCompileSetup](cgxPackage)
	name := C.GoString(cname)
	irPkg := cpkg.IR()
	for _, typ := range irPkg.ExportedTypes() {
		if typ.Name() == name {
			res.iface = (C.cgx_interface)(wrap[*interfaceHandle](newInterfaceHandle(cpkg, typ)))
			return
		}
	}
	res.error = Errorf("type %q not found in package %q", name, irPkg.Name.String())
	return
}

/* cgx_function */

//export cgx_function_has
func cgx_function_has(cgxPackage C.cgx_package, funcNamePtr *C.cchar_t) bool {
	cpkg := unwrap[*core.PackageCompileSetup](cgxPackage)
	name := C.GoString(funcNamePtr)
	return cpkg.IR().FindFunc(name) != nil
}

//export cgx_function_find
func cgx_function_find(cgxPackage C.cgx_package, funcNamePtr *C.cchar_t) (res C.struct_cgx_function_find_result) {
	cpkg := unwrap[*core.PackageCompileSetup](cgxPackage)
	name := C.GoString(funcNamePtr)
	if !ir.IsExported(name) {
		res.error = Errorf("function %q not exported", name)
		return
	}
	function, err := cpkg.NewCache("", name)
	if err != nil {
		res.error = (C.cgx_error)(wrap[error](err))
		return
	}
	res.function = (C.cgx_function)(wrap[*core.FuncCache](function))
	return
}

//export cgx_function_run
func cgx_function_run(cgxFunction C.cgx_function, cgxReceiver C.cgx_value, argCount C.int, args *C.cgx_value) C.struct_cgx_function_run_result {
	function := unwrap[*core.FuncCache](cgxFunction)
	recvValue := unwrap[values.Value](cgxReceiver)
	cgxValues := unsafe.Slice(args, argCount)
	argValues := make([]values.Value, int(argCount))
	for i, cgxValue := range cgxValues {
		argValues[i] = unwrap[values.Value](cgxValue)
	}
	runner, err := function.Runner(recvValue, argValues)
	if err != nil {
		return C.struct_cgx_function_run_result{error: (C.cgx_error)(wrap[error](err))}
	}
	results, err := runner.Run(recvValue, argValues, nil)
	if err != nil {
		return C.struct_cgx_function_run_result{error: (C.cgx_error)(wrap[error](err))}
	}
	return C.struct_cgx_function_run_result{
		values:     (*C.cgx_value)(handle.PinSliceData(handle.WrapSlice(results))),
		value_size: C.uint32_t(len(results)),
	}
}

//export cgx_function_name
func cgx_function_name(cgxFunction C.cgx_function) *C.cchar_t {
	function := unwrap[*core.FuncCache](cgxFunction)
	return C.CString(function.Func().Name())
}

//export cgx_function_string
func cgx_function_string(cgxFunction C.cgx_function) *C.cchar_t {
	function := unwrap[*core.FuncCache](cgxFunction)
	return C.CString(function.Func().String())
}

//export cgx_function_doc
func cgx_function_doc(cgxFunction C.cgx_function) *C.cchar_t {
	function := unwrap[*core.FuncCache](cgxFunction)
	return C.CString(function.Func().Doc().Text())
}

func copySignatureElements(fields *ir.FieldList) *C.struct_cgx_function_signature_element {
	elements := make([]C.struct_cgx_function_signature_element, fields.Len())
	for n, field := range fields.Fields() {
		elements[n].name = C.CString(field.Name.String())
		elements[n].kind = toCGXValueKind(field.Type().Kind())
	}
	return (*C.struct_cgx_function_signature_element)(handle.PinSliceData(elements))
}

//export cgx_function_signature
func cgx_function_signature(cgxFunction C.cgx_function) C.struct_cgx_function_signature_result {
	function := unwrap[*core.FuncCache](cgxFunction)
	fn := function.Func()
	result := C.struct_cgx_function_signature_result{
		parameter:      copySignatureElements(fn.FuncType().Params),
		parameter_size: C.uint32_t(fn.FuncType().Params.Len()),
		result:         copySignatureElements(fn.FuncType().Results),
		result_size:    C.uint32_t(fn.FuncType().Results.Len()),
	}
	return result
}

//export cgx_free_function_signature_result
func cgx_free_function_signature_result(cgxSignature *C.struct_cgx_function_signature_result) {
	freeSignature := func(list *C.struct_cgx_function_signature_element, size C.uint32_t) {
		for _, item := range unsafe.Slice(list, size) {
			C.free(unsafe.Pointer(item.name))
		}
		handle.UnpinSliceData(unsafe.Pointer(list))
	}
	freeSignature(cgxSignature.parameter, cgxSignature.parameter_size)
	cgxSignature.parameter = nil
	cgxSignature.parameter_size = 0
	freeSignature(cgxSignature.result, cgxSignature.result_size)
	cgxSignature.result = nil
	cgxSignature.result_size = 0
}

//export cgx_function_num_params
func cgx_function_num_params(cgxFunction C.cgx_function) int {
	function := unwrap[*core.FuncCache](cgxFunction)
	return function.Func().FuncType().Params.Len()
}

//export cgx_function_param_dtype
func cgx_function_param_dtype(cgxFunction C.cgx_function, arg int) C.enum_cgx_value_kind {
	function := unwrap[*core.FuncCache](cgxFunction)
	params := function.Func().FuncType().Params.Fields()
	if arg < 0 || arg >= len(params) {
		return C.CGX_INVALID
	}
	_, dtype := ir.Shape(params[arg].Type())
	return toCGXValueKind(dtype.Kind())
}

//export cgx_free_function_run_result
func cgx_free_function_run_result(cgxFunctionResult *C.struct_cgx_function_run_result) {
	handle.UnpinSliceData(unsafe.Pointer(cgxFunctionResult.values))
	cgxFunctionResult.values = nil
}

/* cgx_value */

func toValueResult[T dtype.GoDataType](devAtom *types.DeviceAtom[T], err error) C.struct_cgx_value_new_result {
	if err != nil {
		return C.struct_cgx_value_new_result{error: (C.cgx_error)(wrap[error](err))}
	}
	return C.struct_cgx_value_new_result{
		value: (C.cgx_value)(wrap[values.Value](devAtom.GXValue())),
	}
}

//export cgx_value_new_bool
func cgx_value_new_bool(cgxDevice C.cgx_device, value C.bool) C.struct_cgx_value_new_result {
	dev := unwrap[*api.Device](cgxDevice)
	return toValueResult(types.Bool(bool(value)).SendTo(dev))
}

//export cgx_value_new_float32
func cgx_value_new_float32(cgxDevice C.cgx_device, value C.float) C.struct_cgx_value_new_result {
	dev := unwrap[*api.Device](cgxDevice)
	return toValueResult(types.Float32(float32(value)).SendTo(dev))
}

//export cgx_value_new_float64
func cgx_value_new_float64(cgxDevice C.cgx_device, value C.double) C.struct_cgx_value_new_result {
	dev := unwrap[*api.Device](cgxDevice)
	return toValueResult(types.Float64(float64(value)).SendTo(dev))
}

//export cgx_value_new_int32
func cgx_value_new_int32(cgxDevice C.cgx_device, value C.int32_t) C.struct_cgx_value_new_result {
	dev := unwrap[*api.Device](cgxDevice)
	return toValueResult(types.Int32(int32(value)).SendTo(dev))
}

//export cgx_value_new_int64
func cgx_value_new_int64(cgxDevice C.cgx_device, value C.int64_t) C.struct_cgx_value_new_result {
	dev := unwrap[*api.Device](cgxDevice)
	return toValueResult(types.Int64(int64(value)).SendTo(dev))
}

//export cgx_value_new_uint32
func cgx_value_new_uint32(cgxDevice C.cgx_device, value C.uint32_t) C.struct_cgx_value_new_result {
	dev := unwrap[*api.Device](cgxDevice)
	return toValueResult(types.Uint32(uint32(value)).SendTo(dev))
}

//export cgx_value_new_uint64
func cgx_value_new_uint64(cgxDevice C.cgx_device, value C.uint64_t) C.struct_cgx_value_new_result {
	dev := unwrap[*api.Device](cgxDevice)
	return toValueResult(types.Uint64(uint64(value)).SendTo(dev))
}

//export cgx_value_send
func cgx_value_send(cgxDevice C.cgx_device, cgxShape C.cgx_shape, data *C.cvoid_t, dataSize C.uint64_t) C.struct_cgx_value_new_result {
	dev := unwrap[*api.Device](cgxDevice)
	shape := unwrap[*shape.Shape](cgxShape)
	byteData := (*byte)(unsafe.Pointer(data))

	h, err := dev.PlatformDevice().Send(unsafe.Slice(byteData, dataSize), shape)
	if err != nil {
		return C.struct_cgx_value_new_result{error: (C.cgx_error)(wrap[error](err))}
	}
	dataType := ir.TypeFromKind(ir.Kind(shape.DType))
	valueType := ir.NewArrayType(nil, dataType, ir.NewRank(shape.AxisLengths))
	value, err := values.NewDeviceArray(valueType, h)
	if err != nil {
		return C.struct_cgx_value_new_result{error: (C.cgx_error)(wrap[error](err))}
	}
	return C.struct_cgx_value_new_result{
		value: (C.cgx_value)(wrap[values.Value](value)),
	}
}

func toCGXValueKind(kind ir.Kind) C.enum_cgx_value_kind {
	switch kind {
	case ir.BoolKind:
		return C.CGX_BOOL
	case ir.Bfloat16Kind:
		return C.CGX_BFLOAT16
	case ir.Float32Kind:
		return C.CGX_FLOAT32
	case ir.Float64Kind:
		return C.CGX_FLOAT64
	case ir.Int32Kind:
		return C.CGX_INT32
	case ir.Int64Kind:
		return C.CGX_INT64
	case ir.Uint32Kind:
		return C.CGX_UINT32
	case ir.Uint64Kind:
		return C.CGX_UINT64
	case ir.ArrayKind:
		return C.CGX_ARRAY
	case ir.SliceKind:
		return C.CGX_SLICE
	case ir.StructKind:
		return C.CGX_STRUCT
	default:
		return C.CGX_INVALID
	}
}

//export cgx_value_kind_of
func cgx_value_kind_of(cgxValue C.cgx_value) C.enum_cgx_value_kind {
	value := unwrap[values.Value](cgxValue)
	return toCGXValueKind(value.Type().Kind())
}

//export cgx_value_shape
func cgx_value_shape(cgxValue C.cgx_value) C.cgx_shape {
	value := unwrap[values.Value](cgxValue)
	if array, ok := value.(values.Array); ok {
		return (C.cgx_shape)(wrap[*shape.Shape](array.Shape()))
	}
	return 0
}

func atomFromDeviceArray[T dtype.GoDataType](cgxValue C.cgx_value) T {
	value := unwrap[values.Value](cgxValue)
	atomDevice := types.NewDeviceAtom[T](value.(*values.DeviceArray))
	atomHost, err := atomDevice.Fetch()
	if err != nil {
		panic(err)
	}
	return atomHost.Value()
}

//export cgx_value_get_bool
func cgx_value_get_bool(cgxValue C.cgx_value) C.bool {
	return C.bool(atomFromDeviceArray[bool](cgxValue))
}

//export cgx_value_get_float32
func cgx_value_get_float32(cgxValue C.cgx_value) C.float {
	return C.float(atomFromDeviceArray[float32](cgxValue))
}

//export cgx_value_get_float64
func cgx_value_get_float64(cgxValue C.cgx_value) C.double {
	return C.double(atomFromDeviceArray[float64](cgxValue))
}

//export cgx_value_get_int32
func cgx_value_get_int32(cgxValue C.cgx_value) C.int32_t {
	return C.int32_t(atomFromDeviceArray[int32](cgxValue))
}

//export cgx_value_get_int64
func cgx_value_get_int64(cgxValue C.cgx_value) C.int64_t {
	return C.int64_t(atomFromDeviceArray[int64](cgxValue))
}

//export cgx_value_get_uint32
func cgx_value_get_uint32(cgxValue C.cgx_value) C.uint32_t {
	return C.uint32_t(atomFromDeviceArray[uint32](cgxValue))
}

//export cgx_value_get_uint64
func cgx_value_get_uint64(cgxValue C.cgx_value) C.uint64_t {
	return C.uint64_t(atomFromDeviceArray[uint64](cgxValue))
}

//export cgx_value_host_buffer
func cgx_value_host_buffer(cgxValue C.cgx_value) C.struct_cgx_value_host_buffer_result {
	value := unwrap[values.Value](cgxValue)
	deviceArray := value.(*values.DeviceArray)
	hostArray, err := deviceArray.ToHostArray(kernels.Allocator())
	if err != nil {
		return C.struct_cgx_value_host_buffer_result{}
	}
	return C.struct_cgx_value_host_buffer_result{
		buffer: (C.cgx_host_buffer)(wrap[platform.HostBuffer](hostArray.Buffer())),
	}
}

//export cgx_value_get_struct
func cgx_value_get_struct(cgxValue C.cgx_value) C.struct_cgx_value_get_struct_result {
	value := unwrap[values.Value](cgxValue)
	kind := value.Type().Kind()
	if value.Type().Kind() != ir.StructKind {
		return C.struct_cgx_value_get_struct_result{
			error: Errorf("value has kind %v, expect kind %v", kind, ir.StructKind),
		}
	}
	strct := values.Underlying(value).(*values.Struct)
	return C.struct_cgx_value_get_struct_result{
		strct: (C.cgx_struct)(wrap[*structHandle](&structHandle{value: strct})),
	}
}

//export cgx_value_get_interface_type
func cgx_value_get_interface_type(cgxPackage C.cgx_package, cgxValue C.cgx_value) C.cgx_interface {
	value := unwrap[values.Value](cgxValue)
	namedType, ok := value.Type().(*ir.NamedType)
	if !ok {
		return 0
	}
	cpkg := unwrap[*core.PackageCompileSetup](cgxPackage)
	return (C.cgx_interface)(wrap[*interfaceHandle](newInterfaceHandle(cpkg, namedType)))
}

//export cgx_value_string
func cgx_value_string(cgxValue C.cgx_value) *C.cchar_t {
	value := unwrap[values.Value](cgxValue)
	return C.CString(value.String())
}

/* cgx_shape */

func fromCGXValueKind(valueType C.enum_cgx_value_kind) dtype.DataType {
	switch valueType {
	case C.CGX_BOOL:
		return dtype.Bool
	case C.CGX_BFLOAT16:
		return dtype.Bfloat16
	case C.CGX_FLOAT32:
		return dtype.Float32
	case C.CGX_FLOAT64:
		return dtype.Float64
	case C.CGX_INT32:
		return dtype.Int32
	case C.CGX_INT64:
		return dtype.Int64
	case C.CGX_UINT32:
		return dtype.Uint32
	case C.CGX_UINT64:
		return dtype.Uint64
	default:
		return dtype.Invalid
	}
}

//export cgx_shape_new
func cgx_shape_new(dtype C.enum_cgx_value_kind, axisLengths *C.cint64_t, axisLengthsSize C.int) C.cgx_shape {
	return (C.cgx_shape)(wrap[*shape.Shape](&shape.Shape{
		DType:       fromCGXValueKind(dtype),
		AxisLengths: copyPrimitiveSlice[int](unsafe.Pointer(axisLengths), int(axisLengthsSize)),
	}))
}

//export cgx_shape_axes
func cgx_shape_axes(cgxShape C.cgx_shape) C.struct_cgx_shape_axes_result {
	shape := unwrap[*shape.Shape](cgxShape)
	return C.struct_cgx_shape_axes_result{
		axis_lengths: (*C.cint64_t)(handle.PinSliceData[int](shape.AxisLengths)),
		num_axes:     C.uint32_t(len(shape.AxisLengths)),
	}
}

//export cgx_free_shape_axes_result
func cgx_free_shape_axes_result(cgxShapeResult *C.struct_cgx_shape_axes_result) {
	handle.UnpinSliceData(unsafe.Pointer(cgxShapeResult.axis_lengths))
	cgxShapeResult.axis_lengths = nil
	cgxShapeResult.num_axes = 0
}

//export cgx_shape_size
func cgx_shape_size(cgxShape C.cgx_shape) C.int {
	shape := unwrap[*shape.Shape](cgxShape)
	return C.int(shape.Size())
}

//export cgx_shape_element_kind
func cgx_shape_element_kind(cgxShape C.cgx_shape) C.enum_cgx_value_kind {
	shape := unwrap[*shape.Shape](cgxShape)
	if shape.DType < dtype.MaxDataType {
		return toCGXValueKind(ir.Kind(shape.DType))
	}
	return C.CGX_INVALID
}

/* cgx_host_buffer */

//export cgx_host_buffer_acquire_data
func cgx_host_buffer_acquire_data(cgxHostBuffer C.cgx_host_buffer) *C.char {
	hostBuffer := unwrap[platform.HostBuffer](cgxHostBuffer)
	data := hostBuffer.Acquire()
	return (*C.char)(handle.PinSliceData[byte](data))
}

//export cgx_host_buffer_release_data
func cgx_host_buffer_release_data(cgxHostBuffer C.cgx_host_buffer, data *C.char) {
	hostBuffer := unwrap[platform.HostBuffer](cgxHostBuffer)
	handle.UnpinSliceData(unsafe.Pointer(data))
	hostBuffer.Release()
}

/* cgx_struct */

type structHandle struct {
	value *values.Struct
}

//export cgx_struct_field_get
func cgx_struct_field_get(cgxStruct C.cgx_struct, fieldNamePtr *C.cchar_t) (res C.struct_cgx_value_new_result) {
	h := unwrap[*structHandle](cgxStruct)
	fieldValue := h.value.FieldValue(C.GoString(fieldNamePtr))
	return C.struct_cgx_value_new_result{
		value: (C.cgx_value)(wrap[values.Value](fieldValue)),
	}
}

//export cgx_struct_field_set
func cgx_struct_field_set(cgxStruct C.cgx_struct, fieldNamePtr *C.cchar_t, cgxValue C.cgx_value) C.cgx_error {
	h := unwrap[*structHandle](cgxStruct)
	value := unwrap[values.Value](cgxValue)
	h.value.SetField(C.GoString(fieldNamePtr), value)
	return (C.cgx_error)(wrap[error](nil))
}

//export cgx_struct_field_list
func cgx_struct_field_list(cgxStruct C.cgx_struct) (res C.struct_cgx_struct_field_list_result) {
	h := unwrap[*structHandle](cgxStruct)
	fields := h.value.StructType().Fields.Fields()
	result := make([]C.struct_cgx_struct_field_element, len(fields))
	for n, field := range fields {
		result[n] = C.struct_cgx_struct_field_element{
			name: C.CString(field.Name.String()),
			kind: toCGXValueKind(field.Type().Kind()),
		}
	}
	return C.struct_cgx_struct_field_list_result{
		field:      (*C.struct_cgx_struct_field_element)(handle.PinSliceData(result)),
		field_size: C.uint32_t(len(result)),
	}
}

//export cgx_free_struct_field_list_result
func cgx_free_struct_field_list_result(cgxFieldList *C.struct_cgx_struct_field_list_result) {
	for _, item := range unsafe.Slice(cgxFieldList.field, cgxFieldList.field_size) {
		C.free(unsafe.Pointer(item.name))
	}
	handle.UnpinSliceData(unsafe.Pointer(cgxFieldList.field))
	cgxFieldList.field = nil
	cgxFieldList.field_size = 0
}

/* cgx_interface */

type interfaceHandle struct {
	pkg *core.PackageCompileSetup
	typ *ir.NamedType
}

func newInterfaceHandle(pkg *core.PackageCompileSetup, typ *ir.NamedType) *interfaceHandle {
	return &interfaceHandle{pkg: pkg, typ: typ}
}

//export cgx_interface_method_find
func cgx_interface_method_find(cgxIFace C.cgx_interface, methodNamePtr *C.cchar_t) (res C.struct_cgx_function_find_result) {
	iface := unwrap[*interfaceHandle](cgxIFace)
	methodName := C.GoString(methodNamePtr)
	method := iface.typ.MethodByName(methodName)
	if method == nil {
		packageName := iface.typ.File.Package.Name.Name
		typeName := iface.typ.Name()
		res.error = Errorf("type %s.%s has no method %s", packageName, typeName, methodName)
		return
	}
	methodDecl, isDecl := method.(*ir.FuncDecl)
	if !isDecl {
		typeName := iface.typ.Name()
		res.error = Errorf("method %s.%s is builtin", typeName, methodName)
		return
	}
	res.function = (C.cgx_function)(wrap[*core.FuncCache](iface.pkg.NewCacheFromFunc(methodDecl)))
	return
}

//export cgx_interface_name
func cgx_interface_name(cgxIFace C.cgx_interface) *C.cchar_t {
	iface := unwrap[*interfaceHandle](cgxIFace)
	return C.CString(iface.typ.Name())
}

//export cgx_interface_package_name
func cgx_interface_package_name(cgxIFace C.cgx_interface) *C.cchar_t {
	iface := unwrap[*interfaceHandle](cgxIFace)
	return C.CString(iface.typ.File.Package.FullName())
}

//export cgx_interface_list_methods
func cgx_interface_list_methods(cgxIFace C.cgx_interface) C.struct_cgx_list_functions_result {
	iface := unwrap[*interfaceHandle](cgxIFace)
	var funcs []*core.FuncCache
	for _, fn := range iface.typ.Methods {
		if !ir.IsExported(fn.Name()) {
			continue
		}
		methodDecl, isDecl := fn.(*ir.FuncDecl)
		if !isDecl {
			continue
		}
		funcs = append(funcs, iface.pkg.NewCacheFromFunc(methodDecl))
	}
	return C.struct_cgx_list_functions_result{
		funcs:         (*C.cgx_function)(handle.PinSliceData(handle.WrapSlice(funcs))),
		num_functions: C.int(len(funcs)),
	}
}

// Required for cgo to build C libraries.
func main() {}
