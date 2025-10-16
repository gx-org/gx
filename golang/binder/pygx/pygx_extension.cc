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

#include <pybind11/pybind11.h>

#include <cstdint>
#include <functional>
#include <string>
#include <utility>
#include <vector>

#include <absl/status/statusor.h>
#include <absl/types/span.h>
#include <golang/binder/ccgx/cppgx.h>
#include <golang/binder/cgx/cgx.h>
#include "third_party/gxlang/gx/golang/binder/pygx/pygx-inl.h"
#include "third_party/pybind11/include/pybind11/numpy.h"
#include "third_party/pybind11/include/pybind11/pybind11.h"
#include "third_party/pybind11/include/pybind11/pytypes.h"
#include "third_party/pybind11/include/pybind11/stl.h"

using gxlang::cppgx::Builder;
using gxlang::cppgx::Device;
using gxlang::cppgx::Function;
using gxlang::cppgx::FunctionResult;
using gxlang::cppgx::FunctionSignature;
using gxlang::cppgx::HostBuffer;
using gxlang::cppgx::Interface;
using gxlang::cppgx::Package;
using gxlang::cppgx::PackageIR;
using gxlang::cppgx::Runtime;
using gxlang::cppgx::Shape;
using gxlang::cppgx::StaticVar;
using gxlang::cppgx::Struct;
using gxlang::cppgx::Value;
using gxlang::pygx::unwrap_status;
using gxlang::pygx::unwrap_statusor;

namespace py = pybind11;

namespace {

py::dtype to_dtype(enum cgx_value_kind type) {
  switch (type) {
    case CGX_BOOL:
      return py::dtype::of<bool>();
    case CGX_FLOAT32:
      return py::dtype::of<float>();
    case CGX_FLOAT64:
      return py::dtype::of<double>();
    case CGX_INT32:
      return py::dtype::of<int32_t>();
    case CGX_INT64:
      return py::dtype::of<int64_t>();
    case CGX_UINT32:
      return py::dtype::of<uint32_t>();
    case CGX_UINT64:
      return py::dtype::of<uint64_t>();
    default:
      return py::dtype();
  }
}

enum cgx_value_kind from_dtype(py::dtype type) {
  if (type == py::dtype::of<bool>()) return CGX_BOOL;
  if (type == py::dtype::of<float>()) return CGX_FLOAT32;
  if (type == py::dtype::of<double>()) return CGX_FLOAT64;
  if (type == py::dtype::of<int32_t>()) return CGX_INT32;
  if (type == py::dtype::of<int64_t>()) return CGX_INT64;
  if (type == py::dtype::of<uint32_t>()) return CGX_UINT32;
  if (type == py::dtype::of<uint64_t>()) return CGX_UINT64;
  return CGX_INVALID;
}

struct buffer_state {
  cgx_host_buffer host_buffer;
  void* buffer_ptr;
};

absl::StatusOr<py::array> to_ndarray(const Value& value) {
  const absl::StatusOr<Shape> shape = value.shape();
  if (!shape.ok()) {
    return shape.status();
  }
  const absl::StatusOr<std::vector<int64_t>> axes = shape->axes();
  if (!axes.ok()) {
    return axes.status();
  }

  // Buffer management is handled by HostBuffer; simply attach it to the ndarray
  // via py::capsule.
  absl::StatusOr<HostBuffer> buffer_result = value.host_buffer();
  if (!buffer_result.ok()) {
    return buffer_result.status();
  }
  HostBuffer* buffer = new HostBuffer(std::move(buffer_result).value());
  py::capsule releaser(buffer, nullptr, [](void* state_ptr) {
    // Releases the underlying buffer and frees the host buffer handle.
    delete reinterpret_cast<HostBuffer*>(state_ptr);
  });

  const enum cgx_value_kind element_kind = cgx_shape_element_kind(shape->raw());
  return py::array(to_dtype(element_kind), *axes, buffer->Acquire(), releaser);
}

absl::StatusOr<Value> from_ndarray(const Device& device, py::array input) {
  // Ensure the input array is densely packed in row-major format.
  input = py::array::ensure(input, py::array::c_style);

  const std::vector<int64_t> axis_lengths(input.shape(),
                                          input.shape() + input.ndim());
  const absl::StatusOr<Shape> shape =
      Shape::New(from_dtype(input.dtype()), axis_lengths);
  if (!shape.ok()) {
    return shape.status();
  }

  const uint8_t* data_ptr = reinterpret_cast<const uint8_t*>(input.data());
  return Value::SendBytes(device, *shape,
                          absl::MakeSpan(data_ptr, input.nbytes()));
}

}  // namespace

PYBIND11_MODULE(pygx, m) {
  m.def("handle_count", &cgx_handle_count);

  py::class_<Builder>(m, "Builder");

  py::class_<Runtime>(m, "Runtime")
      .def("get_device", &unwrap_statusor<Runtime, &Runtime::GetDevice, int>)
      .def("load",
           &unwrap_statusor<Runtime, &Runtime::Load, const std::string&>);

  py::class_<Device>(m, "Device")
      .def("build_package",
           &unwrap_statusor<Device, &Device::BuildPackage, const std::string&>);

  py::class_<PackageIR>(m, "PackageIR")
      .def("name", &PackageIR::name)
      .def("fullname", &PackageIR::fullname);

  py::class_<Package>(m, "Package")
      .def("ir", &Package::ir)
      .def("find_interface", &unwrap_statusor<Package, &Package::FindInterface,
                                              const std::string&>)
      .def("list_interfaces",
           &unwrap_statusor<Package, &Package::ListInterfaces>)
      .def("has_static_var", &Package::HasStaticVar)
      .def("find_static_var", &unwrap_statusor<Package, &Package::FindStaticVar,
                                               const std::string&>)
      .def("list_static_vars",
           &unwrap_statusor<Package, &Package::ListStaticVars>)
      .def("has_function", &Package::HasFunction)
      .def(
          "find_function",
          &unwrap_statusor<Package, &Package::FindFunction, const std::string&>)
      .def("list_functions",
           &unwrap_statusor<Package, &Package::ListFunctions>);

  py::class_<StaticVar>(m, "StaticVar")
      .def("set_value",
           &unwrap_status<StaticVar, &StaticVar::set_value, int64_t>)
      .def("name", &StaticVar::name);

  py::class_<Function>(m, "Function")
      .def("run", &unwrap_statusor<
                      Function, &Function::Run,
                      const std::vector<std::reference_wrapper<const Value>>&>)
      .def("run_method",
           &unwrap_statusor<
               Function, &Function::RunMethod, const Value&,
               const std::vector<std::reference_wrapper<const Value>>&>)
      .def("signature", &unwrap_statusor<Function, &Function::Signature>)
      .def("num_params", &Function::num_params)
      .def("param_dtype", &Function::param_dtype)
      .def("name", &Function::name)
      .def("__str__", &Function::str)
      .def("doc", &Function::doc);

  py::class_<FunctionResult>(m, "FunctionResult")
      .def("return_values", &FunctionResult::return_values,
           py::return_value_policy::move);

  py::class_<FunctionSignature>(m, "FunctionSignature")
      .def("parameters", &FunctionSignature::parameters)
      .def("results", &FunctionSignature::results);

  py::class_<FunctionSignature::Field>(m, "FunctionSignature_Field")
      .def_readonly("name", &FunctionSignature::Field::name)
      .def_readonly("kind", &FunctionSignature::Field::kind);

  py::class_<Struct>(m, "Struct")
      .def("get_field",
           &unwrap_statusor<Struct, &Struct::GetField, const std::string&>)
      .def("set_field", &unwrap_status<Struct, &Struct::SetField,
                                       const std::string&, const Value&>)
      .def("list_fields", &unwrap_statusor<Struct, &Struct::ListFields>);

  py::class_<Struct::Field>(m, "Struct_Field")
      .def_readonly("name", &Struct::Field::name)
      .def_readonly("kind", &Struct::Field::kind);

  py::class_<Interface>(m, "Interface")
      .def("package_name", &Interface::package_name)
      .def("name", &Interface::name)
      .def("find_method", &unwrap_statusor<Interface, &Interface::FindMethod,
                                           const std::string&>)
      .def("list_methods",
           &unwrap_statusor<Interface, &Interface::ListMethods>);

  py::class_<Value>(m, "Value")
      .def_static("from_bool",
                  &unwrap_statusor<&Value::FromBool, const Device&, bool>)
      .def_static("from_float32",
                  &unwrap_statusor<&Value::FromFloat32, const Device&, float>)
      .def_static("from_float64",
                  &unwrap_statusor<&Value::FromFloat64, const Device&, double>)
      .def_static("from_int32",
                  &unwrap_statusor<&Value::FromInt32, const Device&, int32_t>)
      .def_static("from_int64",
                  &unwrap_statusor<&Value::FromInt64, const Device&, int64_t>)
      .def_static("from_uint32",
                  &unwrap_statusor<&Value::FromUint32, const Device&, uint32_t>)
      .def_static("from_uint64",
                  &unwrap_statusor<&Value::FromUint64, const Device&, uint64_t>)
      .def_static("from_ndarray",
                  &unwrap_statusor<&from_ndarray, const Device&, py::array>)
      .def("kind", &Value::kind)
      .def("shape", &unwrap_statusor<Value, &Value::shape>)
      .def("bool_value", &Value::bool_value)
      .def("float32_value", &Value::float32_value)
      .def("float64_value", &Value::float64_value)
      .def("int32_value", &Value::int32_value)
      .def("int64_value", &Value::int64_value)
      .def("uint32_value", &Value::uint32_value)
      .def("uint64_value", &Value::uint64_value)
      .def("interface_type", &Value::interface_type)
      .def("struct_type", &unwrap_statusor<Value, &Value::as_struct>)
      .def("to_ndarray", &unwrap_statusor<&to_ndarray, const Value&>)
      .def("__repr__", &Value::string)
      .def("__str__", &Value::string);

  py::class_<Shape>(m, "Shape")
      .def_static("new", &unwrap_statusor<&Shape::New, enum cgx_value_kind,
                                          const std::vector<int64_t>&>)
      .def("axes", &unwrap_statusor<Shape, &Shape::axes>)
      .def("size", &Shape::size);

  py::enum_<enum cgx_value_kind>(m, "Kind")
      .value("CGX_INVALID", CGX_INVALID)
      .value("CGX_BOOL", CGX_BOOL)
      .value("CGX_FLOAT32", CGX_FLOAT32)
      .value("CGX_FLOAT64", CGX_FLOAT64)
      .value("CGX_INT32", CGX_INT32)
      .value("CGX_INT64", CGX_INT64)
      .value("CGX_UINT32", CGX_UINT32)
      .value("CGX_UINT64", CGX_UINT64)
      .value("CGX_ARRAY", CGX_ARRAY);
}
