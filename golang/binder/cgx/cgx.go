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
	"runtime"
	"unsafe"

	"github.com/pkg/errors"
	"github.com/gomlx/compute"
	"github.com/gomlx/compute/dtypes"
	"github.com/gomlx/compute/dtypes/gotype"
	"github.com/gomlx/compute/shapes"
	"github.com/gx-org/gx/api"
	"github.com/gx-org/gx/api/hostio"
	"github.com/gx-org/gx/api/options"
	"github.com/gx-org/gx/build/importers"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/build/ir/irkind"
	"github.com/gx-org/gx/cgx/handle"
	"github.com/gx-org/gx/golang/backend/kernels"
	"github.com/gx-org/gx/golang/binder/gobindings/core"
	"github.com/gx-org/gx/golang/binder/gobindings/types"
	"github.com/gx-org/gx/interp/elements"
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

// cgx_list_interfaces is the return value when listing interfaces (i.e. named types) of a GX element.
struct cgx_list_interfaces_result {
	cgx_type* ifaces;
	int num_ifaces;
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

// cgx_type_find_result is the return value for cgx_type_find().
struct cgx_type_find_result {
	cgx_type iface;
	cgx_error error;
};

// cgx_type_list_fields is the return value when listing fields of a GX type.
struct cgx_type_list_fields_result {
	const char** fields;
	int num_fields;
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

// cgx_value_field_get_result is the return value for cgx_value_get_struct.
struct cgx_value_field_get_result {
	cgx_value value;
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

var verboseError bool

//export cgx_error_verbose
func cgx_error_verbose(enable C.bool) {
	verboseError = bool(enable)
}

//export cgx_error_message
func cgx_error_message(cgxError C.cgx_error) *C.cchar_t {
	err := unwrap[error](cgxError)
	var errS string
	if verboseError {
		errS = fmt.Sprintf("%+v\n", err)
	} else {
		errS = err.Error()
	}
	return C.CString(errS)
}

/* cgx_runtime */

//export cgx_runtime_release
func cgx_runtime_release(cgxRuntime C.cgx_runtime) C.cgx_error {
	rtm := unwrap[*api.Runtime](cgxRuntime)
	rtm.Backend().Finalize()
	return 0
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
	devSetup := core.NewDeviceSetup(dev, nil)
	return C.struct_cgx_device_get_result{
		device: (C.cgx_device)(wrap[*core.DeviceSetup](devSetup)),
	}
}

//export cgx_device_get_runtime
func cgx_device_get_runtime(cgxDevice C.cgx_device) C.cgx_runtime {
	dev := unwrap[*core.DeviceSetup](cgxDevice)
	return (C.cgx_runtime)(wrap[*api.Runtime](dev.Runtime()))
}

/* cgx_package_ir */

//export cgx_package_ir_load
func cgx_package_ir_load(cgxRuntime C.cgx_runtime, pathPtr *C.cchar_t) C.struct_cgx_package_ir_load_result {
	rtm := unwrap[*api.Runtime](cgxRuntime)
	pkg, err := rtm.Builder().Build(C.GoString(pathPtr))
	if err != nil {
		pkg = nil
	}
	return C.struct_cgx_package_ir_load_result{
		error:    (C.cgx_error)(wrap[error](err)),
		_package: (C.cgx_package_ir)(wrap[importers.Package](pkg)),
	}
}

/* cgx_package */

//export cgx_package_ir_build_for
func cgx_package_ir_build_for(cgxPackageIR C.cgx_package_ir, cgxDevice C.cgx_device) C.cgx_package {
	dev := unwrap[*core.DeviceSetup](cgxDevice)
	pkg := unwrap[importers.Package](cgxPackageIR)
	return C.cgx_package(wrap[*core.PackageCompileSetup](dev.PackageSetup(pkg)))
}

//export cgx_package_ir_name
func cgx_package_ir_name(cgxPackageIR C.cgx_package_ir) *C.cchar_t {
	pkg := unwrap[importers.Package](cgxPackageIR)
	return C.CString(pkg.IR().Name.Name)
}

//export cgx_package_ir_fullname
func cgx_package_ir_fullname(cgxPackageIR C.cgx_package_ir) *C.cchar_t {
	pkg := unwrap[importers.Package](cgxPackageIR)
	return C.CString(pkg.IR().Path())
}

type staticVariable struct {
	pkg *core.PackageCompileSetup
	vr  *ir.VarExpr
}

//export cgx_package_list_statics
func cgx_package_list_statics(cgxPackage C.cgx_package) C.struct_cgx_list_statics_result {
	cpkg := unwrap[*core.PackageCompileSetup](cgxPackage)
	var statics []*staticVariable
	for _, vr := range cpkg.IR().ExportedStatics() {
		statics = append(statics, &staticVariable{pkg: cpkg, vr: vr})
	}
	return C.struct_cgx_list_statics_result{
		statics:     (*C.cgx_function)(handle.PinSliceData(handle.WrapSlice(statics))),
		num_statics: C.int(len(statics)),
	}
}

//export cgx_static_has
func cgx_static_has(cgxPackage C.cgx_package, staticNamePtr *C.cchar_t) bool {
	cpkg := unwrap[*core.PackageCompileSetup](cgxPackage)
	name := C.GoString(staticNamePtr)
	vr := cpkg.IR().FindStatic(name)
	return vr != nil
}

//export cgx_static_find
func cgx_static_find(cgxPackage C.cgx_package, staticNamePtr *C.cchar_t) (res C.struct_cgx_static_find_result) {
	cpkg := unwrap[*core.PackageCompileSetup](cgxPackage)
	name := C.GoString(staticNamePtr)
	vr := cpkg.IR().FindStatic(name)
	if vr == nil {
		res.error = Errorf("static variable %s not found in package %s", name, cpkg.IR().Name.String())
		return
	}
	if !ir.IsExported(name) {
		res.error = Errorf("static variable %s not exported", vr.VName.Name)
		return
	}
	res.static_var = wrap(&staticVariable{pkg: cpkg, vr: vr})
	return
}

//export cgx_static_name
func cgx_static_name(cgxStatic C.cgx_static) *C.cchar_t {
	cvr := unwrap[*staticVariable](cgxStatic)
	return C.CString(cvr.vr.VName.Name)
}

//export cgx_static_set
func cgx_static_set(cgxStatic C.cgx_static, val int64) C.cgx_error {
	cvr := unwrap[*staticVariable](cgxStatic)
	var gxValue hostio.Value
	var err error
	tp := cvr.vr.Type()
	switch tp {
	case ir.Int32Type():
		gxValue, err = hostio.AtomIntegerValue[int32](ir.Int32Type(), int32(val))
	case ir.Int64Type():
		gxValue, err = hostio.AtomIntegerValue[int64](ir.Int64Type(), val)
	case ir.IntType():
		gxValue, err = hostio.AtomIntegerValue[int](ir.IntType(), int(val))
	default:
		err = errors.Errorf("cannot set static variable: type %s (%T:%s) not supported", tp.ReferString(nil), tp, tp.Kind())
	}
	if err != nil {
		return (C.cgx_error)(wrap[error](err))
	}
	cvr.pkg.Setup().AppendOptions(func(plat compute.Backend) options.PackageOption {
		return options.PackageVarSetValue{
			Pkg:   cvr.pkg.IR().Path(),
			Var:   cvr.vr.VName.Name,
			Value: gxValue,
		}
	})
	return C.cgx_error(0)
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
	var funcs []core.Func
	for fn := range cpkg.IR().ExportedFuncs() {
		fnDecl, isDecl := fn.(*ir.FuncDecl)
		if !isDecl {
			continue
		}
		funcs = append(funcs, cpkg.NewFuncFromDecl(fnDecl))
	}
	return C.struct_cgx_list_functions_result{
		funcs:         (*C.cgx_function)(handle.PinSliceData(handle.WrapSlice(funcs))),
		num_functions: C.int(len(funcs)),
	}
}

//export cgx_free_type_list_fields_result
func cgx_free_type_list_fields_result(res *C.struct_cgx_type_list_fields_result) {
	for _, field := range unsafe.Slice(res.fields, res.num_fields) {
		C.free(unsafe.Pointer(field))
	}
	handle.UnpinSliceData(unsafe.Pointer(res.fields))
	res.fields = nil
	res.num_fields = 0
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
	return (C.cgx_package_ir)(wrap[importers.Package](cpkg.Package()))
}

//export cgx_package_list_interfaces
func cgx_package_list_interfaces(cgxPackage C.cgx_package) C.struct_cgx_list_interfaces_result {
	cpkg := unwrap[*core.PackageCompileSetup](cgxPackage)
	var ifaces []*typeHandle
	for _, iface := range cpkg.IR().ExportedTypes() {
		ifaces = append(ifaces, newTypeHandle(cpkg, iface))
	}
	return C.struct_cgx_list_interfaces_result{
		ifaces:     (*C.cgx_type)(handle.PinSliceData(handle.WrapSlice(ifaces))),
		num_ifaces: C.int(len(ifaces)),
	}
}

//export cgx_free_list_interfaces_result
func cgx_free_list_interfaces_result(res *C.struct_cgx_list_interfaces_result) {
	handle.UnpinSliceData(unsafe.Pointer(res.ifaces))
	res.ifaces = nil
	res.num_ifaces = 0
}

//export cgx_type_find
func cgx_type_find(cgxPackage C.cgx_package, cname *C.cchar_t) (res C.struct_cgx_type_find_result) {
	cpkg := unwrap[*core.PackageCompileSetup](cgxPackage)
	name := C.GoString(cname)
	irPkg := cpkg.IR()
	for _, typ := range irPkg.ExportedTypes() {
		if typ.Name() == name {
			res.iface = (C.cgx_type)(wrap[*typeHandle](newTypeHandle(cpkg, typ)))
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
	function, err := cpkg.NewFunc("", name)
	if err != nil {
		res.error = (C.cgx_error)(wrap[error](err))
		return
	}
	res.function = (C.cgx_function)(wrap[core.Func](function))
	return
}

//export cgx_function_run
func cgx_function_run(cgxFunction C.cgx_function, cgxReceiver C.cgx_value, argCount C.int, args *C.cgx_value) C.struct_cgx_function_run_result {
	function := unwrap[core.Func](cgxFunction)
	recvValue := unwrap[hostio.Value](cgxReceiver)
	cgxValues := unsafe.Slice(args, argCount)
	argValues := make([]hostio.Value, int(argCount))
	for i, cgxValue := range cgxValues {
		argValues[i] = unwrap[hostio.Value](cgxValue)
	}
	results, err := function.Run(recvValue, argValues, function.Tracer())
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
	function := unwrap[core.Func](cgxFunction)
	return C.CString(function.Func().Name())
}

//export cgx_function_string
func cgx_function_string(cgxFunction C.cgx_function) *C.cchar_t {
	function := unwrap[core.Func](cgxFunction)
	return C.CString(function.Func().DefineString(nil))
}

//export cgx_function_doc
func cgx_function_doc(cgxFunction C.cgx_function) *C.cchar_t {
	function := unwrap[core.Func](cgxFunction)
	return C.CString(function.Func().Doc().Text())
}

func copySignatureElements(fields *ir.FieldList) *C.struct_cgx_function_signature_element {
	elements := make([]C.struct_cgx_function_signature_element, fields.Len())
	for n, field := range fields.Fields() {
		name := ""
		if field.Name != nil {
			name = field.Name.String()
		}
		elements[n].name = C.CString(name)
		elements[n].kind = toCGXValueKind(field.Type().Kind())
	}
	return (*C.struct_cgx_function_signature_element)(handle.PinSliceData(elements))
}

//export cgx_function_signature
func cgx_function_signature(cgxFunction C.cgx_function) C.struct_cgx_function_signature_result {
	function := unwrap[core.Func](cgxFunction)
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
	function := unwrap[core.Func](cgxFunction)
	return function.Func().FuncType().NumAllParams()
}

//export cgx_function_param_dtype
func cgx_function_param_dtype(cgxFunction C.cgx_function, arg int) C.enum_cgx_value_kind {
	function := unwrap[core.Func](cgxFunction)
	params := function.Func().FuncType().AllParams()
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

func toValueResult[T gotype.Supported](devAtom *types.DeviceAtom[T], err error) C.struct_cgx_value_new_result {
	if err != nil {
		return C.struct_cgx_value_new_result{error: (C.cgx_error)(wrap[error](err))}
	}
	return C.struct_cgx_value_new_result{
		value: (C.cgx_value)(wrap[hostio.Value](devAtom.GXValue())),
	}
}

//export cgx_value_new_bool
func cgx_value_new_bool(cgxDevice C.cgx_device, value C.bool) C.struct_cgx_value_new_result {
	dev := unwrap[*core.DeviceSetup](cgxDevice)
	return toValueResult(types.Bool(bool(value)).SendTo(dev.Device()))
}

//export cgx_value_new_float32
func cgx_value_new_float32(cgxDevice C.cgx_device, value C.float) C.struct_cgx_value_new_result {
	dev := unwrap[*core.DeviceSetup](cgxDevice)
	return toValueResult(types.Float32(float32(value)).SendTo(dev.Device()))
}

//export cgx_value_new_float64
func cgx_value_new_float64(cgxDevice C.cgx_device, value C.double) C.struct_cgx_value_new_result {
	dev := unwrap[*core.DeviceSetup](cgxDevice)
	return toValueResult(types.Float64(float64(value)).SendTo(dev.Device()))
}

//export cgx_value_new_int32
func cgx_value_new_int32(cgxDevice C.cgx_device, value C.int32_t) C.struct_cgx_value_new_result {
	dev := unwrap[*core.DeviceSetup](cgxDevice)
	return toValueResult(types.Int32(int32(value)).SendTo(dev.Device()))
}

//export cgx_value_new_int64
func cgx_value_new_int64(cgxDevice C.cgx_device, value C.int64_t) C.struct_cgx_value_new_result {
	dev := unwrap[*core.DeviceSetup](cgxDevice)
	return toValueResult(types.Int64(int64(value)).SendTo(dev.Device()))
}

//export cgx_value_new_uint32
func cgx_value_new_uint32(cgxDevice C.cgx_device, value C.uint32_t) C.struct_cgx_value_new_result {
	dev := unwrap[*core.DeviceSetup](cgxDevice)
	return toValueResult(types.Uint32(uint32(value)).SendTo(dev.Device()))
}

//export cgx_value_new_uint64
func cgx_value_new_uint64(cgxDevice C.cgx_device, value C.uint64_t) C.struct_cgx_value_new_result {
	dev := unwrap[*core.DeviceSetup](cgxDevice)
	return toValueResult(types.Uint64(uint64(value)).SendTo(dev.Device()))
}

//export cgx_value_send
func cgx_value_send(cgxDevice C.cgx_device, cgxShape C.cgx_shape, data *C.cvoid_t, dataSize C.uint64_t) C.struct_cgx_value_new_result {
	dev := unwrap[*core.DeviceSetup](cgxDevice)
	shape := unwrap[*shapes.Shape](cgxShape)
	byteData := (*byte)(unsafe.Pointer(data))

	h, err := dev.Device().Send(unsafe.Slice(byteData, dataSize), *shape)
	if err != nil {
		return C.struct_cgx_value_new_result{error: (C.cgx_error)(wrap[error](err))}
	}
	dataType := ir.TypeFromKind(irkind.Kind(shape.DType))
	valueType := ir.NewArrayType(nil, dataType, ir.NewRank(shape.Dimensions))
	value, err := hostio.NewDeviceArray(valueType, h)
	if err != nil {
		return C.struct_cgx_value_new_result{error: (C.cgx_error)(wrap[error](err))}
	}
	return C.struct_cgx_value_new_result{
		value: (C.cgx_value)(wrap[hostio.Value](value)),
	}
}

func toCGXValueKind(kind irkind.Kind) C.enum_cgx_value_kind {
	switch kind {
	case irkind.Bool:
		return C.CGX_BOOL
	case irkind.Bfloat16:
		return C.CGX_BFLOAT16
	case irkind.Float32:
		return C.CGX_FLOAT32
	case irkind.Float64:
		return C.CGX_FLOAT64
	case irkind.Int32:
		return C.CGX_INT32
	case irkind.Int64:
		return C.CGX_INT64
	case irkind.Int:
		return C.CGX_INT
	case irkind.Uint32:
		return C.CGX_UINT32
	case irkind.Uint64:
		return C.CGX_UINT64
	case irkind.String:
		return C.CGX_STRING
	case irkind.Array:
		return C.CGX_ARRAY
	case irkind.Slice:
		return C.CGX_SLICE
	case irkind.Struct:
		return C.CGX_STRUCT
	case irkind.Unknown:
		return C.CGX_UNKNOWN
	case irkind.Interface:
		return C.CGX_INTERFACE
	default:
		return C.CGX_INVALID
	}
}

//export cgx_value_kind_of
func cgx_value_kind_of(cgxValue C.cgx_value) C.enum_cgx_value_kind {
	value := unwrap[hostio.Value](cgxValue)
	return toCGXValueKind(value.Type().Kind())
}

//export cgx_value_shape
func cgx_value_shape(cgxValue C.cgx_value) C.cgx_shape {
	value := unwrap[hostio.Value](cgxValue)
	if array, ok := value.(hostio.Array); ok {
		sh := array.Shape()
		return (C.cgx_shape)(wrap[*shapes.Shape](&sh))
	}
	return 0
}

func atomFromDeviceArray[T gotype.Supported](cgxValue C.cgx_value) T {
	value := unwrap[hostio.Value](cgxValue)
	atomDevice := types.NewDeviceAtom[T](value.(*hostio.DeviceArray))
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
	value := unwrap[hostio.Value](cgxValue)
	deviceArray := value.(*hostio.DeviceArray)
	hostArray, err := deviceArray.ToHostArray(kernels.DefaultAllocator())
	if err != nil {
		return C.struct_cgx_value_host_buffer_result{}
	}
	return C.struct_cgx_value_host_buffer_result{
		buffer: (C.cgx_host_buffer)(wrap[kernels.HostBuffer](hostArray.Buffer())),
	}
}

func structFrom(cval C.cgx_value) (hostio.Value, *elements.Struct) {
	val := unwrap[hostio.Value](cval)
	strct, _ := hostio.Underlying(val).(*elements.Struct)
	return val, strct
}

//export cgx_value_field_set
func cgx_value_field_set(crecv C.cgx_value, fieldNamePtr *C.cchar_t, cval C.cgx_value) C.cgx_error {
	recv, strct := structFrom(crecv)
	if strct == nil {
		return Errorf("type %s has no fields", recv.Type().ReferString(nil))
	}
	strct.SetField(C.GoString(fieldNamePtr), unwrap[hostio.Value](cval))
	return (C.cgx_error)(wrap[error](nil))
}

//export cgx_value_field_get
func cgx_value_field_get(crecv C.cgx_value, fieldNamePtr *C.cchar_t) (res C.struct_cgx_value_field_get_result) {
	recv, strct := structFrom(crecv)
	if strct == nil {
		res.error = Errorf("type %s has no fields", recv.Type().ReferString(nil))
		return
	}
	fieldName := C.GoString(fieldNamePtr)
	if !ir.IsExported(fieldName) {
		res.error = Errorf("field %s is not exported", fieldName)
		return
	}
	fieldElement := strct.FieldValue(fieldName)
	if fieldElement == nil {
		res.error = Errorf("type %s has no field %s", recv.Type().ReferString(nil), fieldName)
		return
	}
	fieldVal, ok := fieldElement.(hostio.Value)
	if !ok {
		res.error = Errorf("cannot use field %s of type %T: not a GX value", fieldName, fieldElement)
		return
	}
	res.value = wrap(fieldVal)
	return
}

//export cgx_value_get_type
func cgx_value_get_type(cgxPackage C.cgx_package, cgxValue C.cgx_value) C.cgx_type {
	value := unwrap[hostio.Value](cgxValue)
	cpkg := unwrap[*core.PackageCompileSetup](cgxPackage)
	return (C.cgx_type)(wrap[*typeHandle](newTypeHandle(cpkg, value.Type())))
}

//export cgx_value_string
func cgx_value_string(cgxValue C.cgx_value) *C.cchar_t {
	value := unwrap[hostio.Value](cgxValue)
	return C.CString(ir.DefineString(nil, value))
}

/* cgx_shape */

func fromCGXValueKind(valueType C.enum_cgx_value_kind) dtypes.DType {
	switch valueType {
	case C.CGX_BOOL:
		return dtypes.Bool
	case C.CGX_BFLOAT16:
		return dtypes.BFloat16
	case C.CGX_FLOAT32:
		return dtypes.Float32
	case C.CGX_FLOAT64:
		return dtypes.Float64
	case C.CGX_INT32:
		return dtypes.Int32
	case C.CGX_INT64:
		return dtypes.Int64
	case C.CGX_UINT32:
		return dtypes.Uint32
	case C.CGX_UINT64:
		return dtypes.Uint64
	default:
		return dtypes.InvalidDType
	}
}

//export cgx_shape_new
func cgx_shape_new(dtype C.enum_cgx_value_kind, axisLengths *C.cint64_t, axisLengthsSize C.int) C.cgx_shape {
	return (C.cgx_shape)(wrap[*shapes.Shape](&shapes.Shape{
		DType:      fromCGXValueKind(dtype),
		Dimensions: copyPrimitiveSlice[int](unsafe.Pointer(axisLengths), int(axisLengthsSize)),
	}))
}

//export cgx_shape_axes
func cgx_shape_axes(cgxShape C.cgx_shape) C.struct_cgx_shape_axes_result {
	shape := unwrap[*shapes.Shape](cgxShape)
	return C.struct_cgx_shape_axes_result{
		axis_lengths: (*C.cint64_t)(handle.PinSliceData[int](shape.Dimensions)),
		num_axes:     C.uint32_t(len(shape.Dimensions)),
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
	shape := unwrap[*shapes.Shape](cgxShape)
	return C.int(shape.Size())
}

//export cgx_shape_element_kind
func cgx_shape_element_kind(cgxShape C.cgx_shape) C.enum_cgx_value_kind {
	shape := unwrap[*shapes.Shape](cgxShape)
	return toCGXValueKind(irkind.Kind(shape.DType))
}

/* cgx_host_buffer */

//export cgx_host_buffer_acquire_data
func cgx_host_buffer_acquire_data(cgxHostBuffer C.cgx_host_buffer) *C.char {
	hostBuffer := unwrap[kernels.HostBuffer](cgxHostBuffer)
	data := hostBuffer.Acquire()
	return (*C.char)(handle.PinSliceData[byte](data))
}

//export cgx_host_buffer_release_data
func cgx_host_buffer_release_data(cgxHostBuffer C.cgx_host_buffer, data *C.char) {
	hostBuffer := unwrap[kernels.HostBuffer](cgxHostBuffer)
	handle.UnpinSliceData(unsafe.Pointer(data))
	hostBuffer.Release()
}

/* cgx_type */

type typeHandle struct {
	pkg *core.PackageCompileSetup
	typ ir.Type
}

func newTypeHandle(pkg *core.PackageCompileSetup, typ ir.Type) *typeHandle {
	return &typeHandle{pkg: pkg, typ: typ}
}

//export cgx_type_method_find
func cgx_type_method_find(cgxIFace C.cgx_type, methodNamePtr *C.cchar_t) (res C.struct_cgx_function_find_result) {
	iface := unwrap[*typeHandle](cgxIFace)
	methodName := C.GoString(methodNamePtr)
	if !ir.IsExported(methodName) {
		res.error = Errorf("method %s is not exported", methodName)
		return
	}
	var method ir.PkgFunc
	if named, _ := iface.typ.(*ir.NamedType); named != nil {
		method = ir.MethodByName(named, methodName)
	}
	if method == nil {
		res.error = Errorf("type %s has no method %s", iface.typ.ReferString(nil), methodName)
		return
	}
	methodDecl, isDecl := method.(*ir.FuncDecl)
	if !isDecl {
		res.error = Errorf("method %s.%s is builtin", iface.typ.ReferString(nil), methodName)
		return
	}
	res.function = (C.cgx_function)(wrap[core.Func](iface.pkg.NewFuncFromDecl(methodDecl)))
	return
}

//export cgx_type_name
func cgx_type_name(ctype C.cgx_type) *C.cchar_t {
	htyp := unwrap[*typeHandle](ctype)
	var name string
	if named, _ := htyp.typ.(*ir.NamedType); named != nil {
		name = named.Name()
	}
	return C.CString(name)
}

//export cgx_type_string
func cgx_type_string(ctype C.cgx_type) *C.cchar_t {
	htyp := unwrap[*typeHandle](ctype)
	return C.CString(htyp.typ.ReferString(nil))
}

//export cgx_type_package_name
func cgx_type_package_name(cgxIFace C.cgx_type) *C.cchar_t {
	iface := unwrap[*typeHandle](cgxIFace)
	var pkgPath string
	if named, _ := iface.typ.(*ir.NamedType); named != nil {
		pkgPath = named.File.Package.Path()
	}
	return C.CString(pkgPath)
}

func listFieldsOf(pkg *core.PackageCompileSetup, strct *ir.StructType) []*C.cchar_t {
	var fields []*C.cchar_t
	for _, field := range strct.Fields.Fields() {
		name := field.Name.Name
		if !ir.IsExported(name) {
			continue
		}
		fields = append(fields, C.CString(name))
	}
	return fields
}

//export cgx_type_list_fields
func cgx_type_list_fields(cgxIFace C.cgx_type) C.struct_cgx_type_list_fields_result {
	iface := unwrap[*typeHandle](cgxIFace)
	strct, isStruct := ir.Underlying(iface.typ).(*ir.StructType)
	var fields []*C.cchar_t
	if isStruct {
		fields = listFieldsOf(iface.pkg, strct)
	}
	return C.struct_cgx_type_list_fields_result{
		fields:     (**C.cchar_t)(handle.PinSliceData(fields)),
		num_fields: C.int(len(fields)),
	}
}

func listMethodsOf(pkg *core.PackageCompileSetup, named *ir.NamedType) []core.Func {
	var funcs []core.Func
	for _, fn := range named.Meths {
		if !ir.IsExported(fn.Name()) {
			continue
		}
		methodDecl, isDecl := fn.(*ir.FuncDecl)
		if !isDecl {
			continue
		}
		funcs = append(funcs, pkg.NewFuncFromDecl(methodDecl))
	}
	return funcs
}

//export cgx_type_list_methods
func cgx_type_list_methods(cgxIFace C.cgx_type) C.struct_cgx_list_functions_result {
	iface := unwrap[*typeHandle](cgxIFace)
	var funcs []core.Func
	if named, _ := iface.typ.(*ir.NamedType); named != nil {
		funcs = listMethodsOf(iface.pkg, named)
	}
	return C.struct_cgx_list_functions_result{
		funcs:         (*C.cgx_function)(handle.PinSliceData(handle.WrapSlice(funcs))),
		num_functions: C.int(len(funcs)),
	}
}

//export cgx_run_gc
func cgx_run_gc() {
	runtime.GC()
}

// Required for cgo to build C libraries.
func main() {}
