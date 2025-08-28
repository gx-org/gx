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

#ifndef THIRD_PARTY_GXLANG_GX_GOLANG_BINDER_CGX_CGX_H_
#define THIRD_PARTY_GXLANG_GX_GOLANG_BINDER_CGX_CGX_H_

#include <stdint.h>

#ifdef __cplusplus
extern "C" {
#endif
// cgx core provides basic types for bridging C and GX (through cgo).
//
// Memory management rules:
// * cgx references such as cgx_builder, cgx_runtime, cgx_value, etc.
//   must be released by the caller using:
//       cgx_release_reference(...)
// * C strings (i.e. *cchar_t) must be freed with C.free().
// * Some complex cgx result structures may have custom releasers.
//   These are released by calling a provided cgx function with a.
//   corresponding name. For example, call:
//       cgx_function_signature_free
//   to release the structure cgx_function_signature_result.
//   If a structure contains cgx references (e.g. a list of function), then
//   these references need to be released individually by the caller as
//   described above.
//
// cgo doesn't provide a way to refer to const types, so we have to define names
// for those we care about.
typedef const char cchar_t;
typedef const int32_t cint32_t;
typedef const int64_t cint64_t;
typedef const void cvoid_t;

// CGX references that always need to be released by the caller.

// cgx_handle represents an opaque handle for a Go/GX value. The zero value
// means the corresponding Go value is nil.
typedef uint64_t cgx_handle;

// cgx_error is an opaque handle for a Go/GX error.
//
// The zero value indicates success.
typedef cgx_handle cgx_error;

// cgx_builder is an opaque handle for a GX builder.
typedef cgx_handle cgx_builder;

// cgx_runtime is an opaque handle for a GX runtime.
typedef cgx_handle cgx_runtime;

// cgx_package_ir is an opaque handle for a GX package IR.
typedef cgx_handle cgx_package_ir;

// cgx_device is an opaque handle for a GX device.
typedef cgx_handle cgx_device;

// cgx_package_option is an option to compile a package.
typedef cgx_handle cgx_package_option;

// cgx_package is an opaque handle for a GX package.
typedef cgx_handle cgx_package;

// cgx_function is an opaque handle for a compiled GX function.
typedef cgx_handle cgx_function;

// cgx_value is an opaque handle for a GX value.
typedef cgx_handle cgx_value;

// cgx_value is an opaque handle for a GX shape.
typedef cgx_handle cgx_shape;

// cgx_host_buffer is an opaque handle for a GX host buffer.
typedef cgx_handle cgx_host_buffer;

// cgx_struct is an opaque handle for a GX struct value,
// that a structure with fields.
typedef cgx_handle cgx_struct;

// cgx_interface is an opaque handle for a GX interface value,
// that is a named type with methods.
// Note that a GX value that is a structure with methods
// can be converted into a cgx_interface and a cgx_struct.
typedef cgx_handle cgx_interface;

// cgx_value_kind describes the kind of a GX value, which can range from simple
// atomic data types to composite types.
enum cgx_value_kind {
  CGX_INVALID,
  CGX_BOOL,
  CGX_BFLOAT16,
  CGX_FLOAT32,
  CGX_FLOAT64,
  CGX_INT32,
  CGX_INT64,
  CGX_UINT32,
  CGX_UINT64,

  // For the multi-dimensional or composite types below, you must further
  // inspect their fields, dimensions, and base data types.
  CGX_ARRAY,
  CGX_SLICE,
  CGX_STRUCT,
};

// cgx_builder_new_result is the return value for cgx_builder_new_*().
struct cgx_builder_new_result {
  cgx_builder builder;
  cgx_error error;
};

// cgx_runtime_new_result is the return value for cgx_runtime_new_*().
struct cgx_runtime_new_result {
  cgx_runtime runtime;
  cgx_error error;
};

// cgx_package_ir_load_result is the return value for cgx_ir_package_load().
struct cgx_package_ir_load_result {
  cgx_package_ir package;
  cgx_error error;
};

#ifdef __cplusplus
}

#endif

#endif  // THIRD_PARTY_GXLANG_GX_GOLANG_BINDER_CGX_CGX_H_
