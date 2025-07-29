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

#include <golang/binder/ccgx/cppgx.h>

#include <stdlib.h>

#include <cassert>
#include <cstdint>
#include <functional>
#include <optional>
#include <string>
#include <utility>
#include <vector>

#include <absl/status/status.h>
#include <absl/status/statusor.h>
#include <absl/types/span.h>
#include <golang/binder/cgx/cgx.h>

namespace gxlang {
namespace cppgx {
namespace {

std::vector<Function> make_function_vector(cgx_list_functions_result *result) {
  std::vector<Function> funcs(result->funcs,
                              result->funcs + result->num_functions);
  cgx_free_list_functions_result(result);
  return funcs;
}

} // namespace

#define CPPGX_RETURN_IF_ERROR(error)                                           \
  if (error != cgx_error{})                                                    \
  return ToErrorStatus(error)

/* Runtime */

absl::StatusOr<PackageIR> Runtime::Load(const std::string &path) const {
  const auto result = cgx_package_ir_load(*runtime_, path.c_str());
  CPPGX_RETURN_IF_ERROR(result.error);
  return PackageIR(result.package);
}

absl::StatusOr<Device> Runtime::GetDevice(int index) const {
  const auto result = cgx_device_get(*runtime_, index);
  CPPGX_RETURN_IF_ERROR(result.error);
  return Device(result.device);
}

/* Package IR */

Package PackageIR::BuildFor(const Device &device) const {
  const auto package = cgx_package_ir_build_for(*package_ir_, device.raw());
  return Package(package);
}

std::string PackageIR::name() const {
  return FromHeapCString(cgx_package_ir_name(*package_ir_));
}

std::string PackageIR::fullname() const {
  return FromHeapCString(cgx_package_ir_fullname(*package_ir_));
}

/* Device */

absl::StatusOr<Package> Device::BuildPackage(const std::string &path) const {
  const cgx_runtime rtm(cgx_device_get_runtime(*device_));
  const auto result = cgx_package_ir_load(rtm, path.c_str());
  cgx_release_reference(rtm);
  CPPGX_RETURN_IF_ERROR(result.error);
  auto package = Package(cgx_package_ir_build_for(result.package, raw()));
  cgx_release_reference(result.package);
  return package;
}

Runtime Device::runtime() const {
  const cgx_runtime rtm(cgx_device_get_runtime(*device_));
  return Runtime(rtm);
}

template <> absl::StatusOr<DeviceAtomic<bool>> Device::Send(bool val) {
  auto value = Value::FromBool(*this, val);
  if (!value.ok())
    return value.status();
  return DeviceAtomic<bool>(*this, *std::move(value));
}

template <> absl::StatusOr<DeviceAtomic<float>> Device::Send(float val) {
  auto value = Value::FromFloat32(*this, val);
  if (!value.ok())
    return value.status();
  return DeviceAtomic<float>(*this, *std::move(value));
}

template <> absl::StatusOr<DeviceAtomic<double>> Device::Send(double val) {
  auto value = Value::FromFloat64(*this, val);
  if (!value.ok())
    return value.status();
  return DeviceAtomic<double>(*this, *std::move(value));
}

template <> absl::StatusOr<DeviceAtomic<int32_t>> Device::Send(int32_t val) {
  auto value = Value::FromInt32(*this, val);
  if (!value.ok())
    return value.status();
  return DeviceAtomic<int32_t>(*this, *std::move(value));
}

template <> absl::StatusOr<DeviceAtomic<int64_t>> Device::Send(int64_t val) {
  auto value = Value::FromInt64(*this, val);
  if (!value.ok())
    return value.status();
  return DeviceAtomic<int64_t>(*this, *std::move(value));
}

template <> absl::StatusOr<DeviceAtomic<uint32_t>> Device::Send(uint32_t val) {
  auto value = Value::FromUint32(*this, val);
  if (!value.ok())
    return value.status();
  return DeviceAtomic<uint32_t>(*this, *std::move(value));
}

template <> absl::StatusOr<DeviceAtomic<uint64_t>> Device::Send(uint64_t val) {
  auto value = Value::FromUint64(*this, val);
  if (!value.ok())
    return value.status();
  return DeviceAtomic<uint64_t>(*this, *std::move(value));
}

/* Package */

absl::StatusOr<Function> Package::FindFunction(const std::string &name) const {
  const auto result = cgx_function_find(*package_, name.c_str());
  CPPGX_RETURN_IF_ERROR(result.error);
  return Function(result.function);
}

absl::StatusOr<Interface>
Package::FindInterface(const std::string &name) const {
  const auto result = cgx_interface_find(*package_, name.c_str());
  CPPGX_RETURN_IF_ERROR(result.error);
  return Interface(result.iface);
}

absl::StatusOr<std::vector<Function>> Package::ListFunctions() const {
  auto result = cgx_package_list_functions(*package_);
  CPPGX_RETURN_IF_ERROR(result.error);
  return make_function_vector(&result);
}

/* Function */

absl::StatusOr<FunctionResult> Function::Run(
    const std::vector<std::reference_wrapper<const Value>> &args) const {
  return RunMethod(Value(cgx_value{}), args);
}

absl::StatusOr<FunctionResult> Function::RunMethod(
    const Value &receiver,
    const std::vector<std::reference_wrapper<const Value>> &args) const {
  std::vector<cgx_value> cgx_args;
  cgx_args.reserve(args.size());
  for (const Value &arg : args) {
    cgx_args.push_back(arg.raw());
  }

  const cgx_function_run_result result = cgx_function_run(
      *function_, receiver.raw(), cgx_args.size(), cgx_args.data());
  CPPGX_RETURN_IF_ERROR(result.error);
  return FunctionResult(std::move(result));
}

absl::StatusOr<FunctionSignature> Function::Signature() const {
  const cgx_function_signature_result result =
      cgx_function_signature(*function_);
  CPPGX_RETURN_IF_ERROR(result.error);
  return FunctionSignature(result);
}

std::string Function::name() const {
  return FromHeapCString(cgx_function_name(raw()));
}

std::string Function::doc() const {
  return FromHeapCString(cgx_function_doc(raw()));
}

cgx_value_kind Function::param_dtype(int param) const {
  return cgx_function_param_dtype(raw(), param);
}

int Function::num_params() const { return cgx_function_num_params(raw()); }

/* FunctionResult */

FunctionResult::FunctionResult(cgx_function_run_result result)
    : result_(result) {
  return_values_.reserve(result.value_size);
  for (int i = 0; i < result.value_size; ++i) {
    return_values_.push_back(Value(result.values[i]));
  }
}

FunctionResult::FunctionResult(FunctionResult &&other)
    : result_(other.result_), return_values_(std::move(other.return_values_)) {
  other.result_.values = nullptr;
}

FunctionResult::~FunctionResult() { cgx_free_function_run_result(&result_); }

/* FunctionSignature */

FunctionSignature::FunctionSignature(cgx_function_signature_result signature)
    : parameters_(ToFields(signature.parameter, signature.parameter_size)),
      results_(ToFields(signature.result, signature.result_size)) {
  cgx_free_function_signature_result(&signature);
}

std::vector<FunctionSignature::Field>
FunctionSignature::ToFields(struct cgx_function_signature_element *fields,
                            uint32_t size) {
  std::vector<FunctionSignature::Field> result;
  result.reserve(size);
  for (uint32_t i = 0; i < size; ++i) {
    result.push_back(
        FunctionSignature::Field{std::string(fields[i].name), fields[i].kind});
  }
  return result;
}

/* Value */

absl::StatusOr<Value> Value::FromBool(const Device &device, bool value) {
  const cgx_value_new_result result = cgx_value_new_bool(device.raw(), value);
  CPPGX_RETURN_IF_ERROR(result.error);
  return Value(result.value);
}

absl::StatusOr<Value> Value::FromFloat32(const Device &device, float value) {
  const cgx_value_new_result result =
      cgx_value_new_float32(device.raw(), value);
  CPPGX_RETURN_IF_ERROR(result.error);
  return Value(result.value);
}

absl::StatusOr<Value> Value::FromFloat64(const Device &device, double value) {
  const cgx_value_new_result result =
      cgx_value_new_float64(device.raw(), value);
  CPPGX_RETURN_IF_ERROR(result.error);
  return Value(result.value);
}

absl::StatusOr<Value> Value::FromInt32(const Device &device, int32_t value) {
  const cgx_value_new_result result = cgx_value_new_int32(device.raw(), value);
  CPPGX_RETURN_IF_ERROR(result.error);
  return Value(result.value);
}

absl::StatusOr<Value> Value::FromInt64(const Device &device, int64_t value) {
  const cgx_value_new_result result = cgx_value_new_int64(device.raw(), value);
  CPPGX_RETURN_IF_ERROR(result.error);
  return Value(result.value);
}

absl::StatusOr<Value> Value::FromUint32(const Device &device, uint32_t value) {
  const cgx_value_new_result result = cgx_value_new_uint32(device.raw(), value);
  CPPGX_RETURN_IF_ERROR(result.error);
  return Value(result.value);
}

absl::StatusOr<Value> Value::FromUint64(const Device &device, uint64_t value) {
  const cgx_value_new_result result = cgx_value_new_uint64(device.raw(), value);
  CPPGX_RETURN_IF_ERROR(result.error);
  return Value(result.value);
}

absl::StatusOr<Value> Value::SendBytes(const Device &device, const Shape &shape,
                                       absl::Span<const uint8_t> data) {
  const cgx_value_new_result result =
      cgx_value_send(device.raw(), shape.raw(), data.data(), data.size());
  CPPGX_RETURN_IF_ERROR(result.error);
  return Value(result.value);
}

absl::StatusOr<HostBuffer> Value::host_buffer() const {
  const cgx_value_host_buffer_result buffer_result =
      cgx_value_host_buffer(raw());
  CPPGX_RETURN_IF_ERROR(buffer_result.error);
  return HostBuffer(buffer_result.buffer);
}

absl::StatusOr<Shape> Value::shape() const {
  const cgx_shape result = cgx_value_shape(*value_);
  if (result == cgx_shape{}) {
    return absl::Status(absl::StatusCode::kUnknown, "value has no shape");
  }
  return Shape(result);
}

std::string Value::string() const {
  return FromHeapCString(cgx_value_string(*value_));
}

absl::StatusOr<Struct> Value::as_struct() const {
  const cgx_value_get_struct_result result = cgx_value_get_struct(*value_);
  CPPGX_RETURN_IF_ERROR(result.error);
  return Struct(result.strct);
}

std::optional<Interface> Value::interface_type(const Device &device) const {
  cgx_interface iface = cgx_value_get_interface_type(device.raw(), *value_);
  if (iface == cgx_interface{}) {
    return std::nullopt;
  }
  return std::optional<Interface>(Interface(iface));
}

template <> bool Value::get<bool>() const { return bool_value(); }

template <> float Value::get<float>() const { return float32_value(); }

template <> double Value::get<double>() const { return float64_value(); }

template <> int32_t Value::get<int32_t>() const { return int32_value(); }

template <> int64_t Value::get<int64_t>() const { return int64_value(); }

template <> uint32_t Value::get<uint32_t>() const { return uint32_value(); }

template <> uint64_t Value::get<uint64_t>() const { return uint64_value(); }

/* Shape */

absl::StatusOr<Shape> Shape::New(enum cgx_value_kind dtype,
                                 const std::vector<int64_t> &axis_lengths) {
  const cgx_shape shape =
      cgx_shape_new(dtype, axis_lengths.data(), axis_lengths.size());
  return Shape(shape);
}

absl::StatusOr<std::vector<int64_t>> Shape::axes() const {
  auto axes_result = cgx_shape_axes(*shape_);
  CPPGX_RETURN_IF_ERROR(axes_result.error);
  const std::vector<int64_t> result(axes_result.axis_lengths,
                                    axes_result.axis_lengths +
                                        axes_result.num_axes);
  cgx_free_shape_axes_result(&axes_result);
  return result;
}

/* HostBuffer */

void *HostBuffer::Acquire() {
  // Save the acquired buffer so the destructor can automatically release it.
  data_ = cgx_host_buffer_acquire_data(*host_buffer_);
  return data_;
}

/* Struct */

absl::StatusOr<Value> Struct::GetField(const std::string &name) const {
  const auto result = cgx_struct_field_get(raw(), name.c_str());
  CPPGX_RETURN_IF_ERROR(result.error);
  return Value(result.value);
}

absl::Status Struct::SetField(const std::string &name, const Value &value) {
  const cgx_error err = cgx_struct_field_set(raw(), name.c_str(), value.raw());
  CPPGX_RETURN_IF_ERROR(err);
  return absl::OkStatus();
}

absl::StatusOr<std::vector<Struct::Field>> Struct::ListFields() const {
  cgx_struct_field_list_result fields = cgx_struct_field_list(raw());
  CPPGX_RETURN_IF_ERROR(fields.error);

  std::vector<Struct::Field> result;
  result.reserve(fields.field_size);
  for (uint32_t i = 0; i < fields.field_size; ++i) {
    result.push_back(
        Field{std::string(fields.field[i].name), fields.field[i].kind});
  }
  cgx_free_struct_field_list_result(&fields);
  return result;
}

/* Interface */

absl::StatusOr<Function> Interface::FindMethod(const std::string &name) const {
  const auto result = cgx_interface_method_find(*iface_, name.c_str());
  CPPGX_RETURN_IF_ERROR(result.error);
  return Function(result.function);
}

std::string Interface::package_name() const {
  return FromHeapCString(cgx_interface_package_name(*iface_));
}

std::string Interface::name() const {
  return FromHeapCString(cgx_interface_name(*iface_));
}

absl::StatusOr<std::vector<Function>> Interface::ListMethods() const {
  auto result = cgx_interface_list_methods(*iface_);
  CPPGX_RETURN_IF_ERROR(result.error);
  return make_function_vector(&result);
}

#undef CPPGX_RETURN_IF_ERROR

std::string FromHeapCString(const char *cstr) {
  const std::string result(cstr);
  free((void *)cstr);
  return result;
}

absl::Status ToErrorStatus(cgx_error err) {
  auto msg = FromHeapCString(cgx_error_message(err));
  const absl::Status result(absl::StatusCode::kInternal, msg);
  cgx_release_reference(err);
  return result;
}

} // namespace cppgx
} // namespace gxlang
