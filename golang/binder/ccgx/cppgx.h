/*
 * Copyright 2024 Google LLC
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

#ifndef THIRD_PARTY_GXLANG_GX_GOLANG_BINDER_CGX_CPPGX_H_
#define THIRD_PARTY_GXLANG_GX_GOLANG_BINDER_CGX_CPPGX_H_

#include <cstdint>
#include <functional>
#include <memory>
#include <optional>
#include <string>
#include <utility>
#include <vector>

#include <absl/status/status.h>
#include <absl/status/statusor.h>
#include <absl/types/span.h>
#include <golang/binder/cgx/cgx.cgo.h>
#include <golang/binder/cgx/cgx.h>

namespace gxlang {
namespace cppgx {

class Runtime;
class Device;
class HostBuffer;
class PackageIR;
class Package;
class Function;
class FunctionResult;
class FunctionSignature;
class Shape;
class Value;
class Struct;
class Interface;
class StaticVar;

// Handle is a move-only wrapper class that manages cgx opaque handles. Each
// handle has a single exclusive owner at any given time.
template <typename T>
class Handle {
 public:
  explicit Handle(T value) : value_(value) {}
  Handle(Handle&& other) : value_(std::exchange(other.value_, T{})) {}
  Handle(const Handle& other) = delete;
  ~Handle() {
    cgx_release_reference(value_);
    value_ = T{};
  }

  T operator*() { return value_; }
  const T operator*() const { return value_; }

 private:
  T value_;
};

class Builder {
 public:
  Builder(cgx_builder builder) : builder_(builder) {}

  cgx_builder raw() const { return *builder_; }

 private:
  Handle<cgx_builder> builder_;
};

class Runtime {
 public:
  Runtime(cgx_runtime runtime) : runtime_(runtime) {}

  cgx_runtime raw() const { return *runtime_; }

  absl::StatusOr<Device> GetDevice(int index) const;

  absl::StatusOr<PackageIR> Load(const std::string& path) const;

  absl::Status Release();

 private:
  Handle<cgx_runtime> runtime_;
};

class PackageIR {
 public:
  PackageIR(cgx_package_ir package_ir) : package_ir_(package_ir) {}

  Package BuildFor(const Device& device) const;

  cgx_package_ir raw() const { return *package_ir_; }
  std::string name() const;
  std::string fullname() const;

 private:
  Handle<cgx_package_ir> package_ir_;
};

class Package {
 public:
  Package(cgx_package package) : package_(package) {}

  cgx_package raw() const { return *package_; }
  PackageIR ir() const {
    // cgx_package_get_ir always return a new handle.
    return PackageIR(cgx_package_get_ir(raw()));
  }

  absl::StatusOr<Interface> FindInterface(const std::string& name) const;
  absl::StatusOr<std::vector<Interface>> ListInterfaces() const;

  bool HasStaticVar(const std::string& name) const;
  absl::StatusOr<StaticVar> FindStaticVar(const std::string& name) const;
  absl::StatusOr<std::vector<StaticVar>> ListStaticVars() const;

  bool HasFunction(const std::string& name) const;
  absl::StatusOr<Function> FindFunction(const std::string& name) const;
  absl::StatusOr<std::vector<Function>> ListFunctions() const;

 private:
  Handle<cgx_package> package_;
};

class StaticVar {
 public:
  StaticVar(cgx_static static_var) : static_var_(static_var) {}

  cgx_static raw() const { return *static_var_; }
  std::string name() const;
  absl::Status set_value(int64_t value) const;

 private:
  Handle<cgx_static> static_var_;
};

class Function {
 public:
  Function(cgx_function function) : function_(function) {}

  cgx_function raw() const { return *function_; }
  std::string name() const;
  std::string str() const;
  std::string doc() const;
  cgx_value_kind param_dtype(int param) const;
  int num_params() const;

  absl::StatusOr<FunctionResult> Run(
      const std::vector<std::reference_wrapper<const Value>>& args) const;
  absl::StatusOr<FunctionResult> RunMethod(
      const Value& receiver,
      const std::vector<std::reference_wrapper<const Value>>& args) const;
  absl::StatusOr<FunctionSignature> Signature() const;

 private:
  Handle<cgx_function> function_;
};

class FunctionResult {
 public:
  FunctionResult(cgx_function_run_result result);
  FunctionResult(FunctionResult&& other);
  FunctionResult(const FunctionResult&) = delete;
  ~FunctionResult();

  std::vector<Value>& mutable_values() { return return_values_; }
  const std::vector<Value>& return_values() const { return return_values_; }

 private:
  cgx_function_run_result result_;
  std::vector<Value> return_values_;
};

class FunctionSignature {
 public:
  struct Field {
    const std::string name;
    const cgx_value_kind kind;
  };

  FunctionSignature(cgx_function_signature_result signature);

  const std::vector<Field>& parameters() const { return parameters_; }
  const std::vector<Field>& results() const { return results_; }

 private:
  static std::vector<Field> ToFields(
      struct cgx_function_signature_element* fields, uint32_t size);

  const std::vector<Field> parameters_;
  const std::vector<Field> results_;
};

class Value {
 public:
  static absl::StatusOr<Value> FromBool(const Device& device, bool value);
  static absl::StatusOr<Value> FromFloat32(const Device& device, float value);
  static absl::StatusOr<Value> FromFloat64(const Device& device, double value);
  static absl::StatusOr<Value> FromInt32(const Device& device, int32_t value);
  static absl::StatusOr<Value> FromInt64(const Device& device, int64_t value);
  static absl::StatusOr<Value> FromUint32(const Device& device, uint32_t value);
  static absl::StatusOr<Value> FromUint64(const Device& device, uint64_t value);
  static absl::StatusOr<Value> SendBytes(const Device& device,
                                         const Shape& shape,
                                         absl::Span<const uint8_t> data);
  template <typename T>
  static absl::StatusOr<Value> Send(const Device& device, const Shape& shape,
                                    absl::Span<T> data) {
    const absl::Span<const uint8_t> data_bytes(
        reinterpret_cast<const uint8_t*>(data.data()), data.size() * sizeof(T));
    return SendBytes(device, shape, data_bytes);
  }

  Value(cgx_value value) : value_(value) {}

  cgx_value raw() const { return *value_; }
  absl::StatusOr<HostBuffer> host_buffer() const;
  absl::StatusOr<Shape> shape() const;
  enum cgx_value_kind kind() const { return cgx_value_kind_of(*value_); }

  template <typename T>
  T get() const;

  bool bool_value() const { return cgx_value_get_bool(*value_); }
  float float32_value() const { return cgx_value_get_float32(*value_); }
  double float64_value() const { return cgx_value_get_float64(*value_); }
  int32_t int32_value() const { return cgx_value_get_int32(*value_); }
  int64_t int64_value() const { return cgx_value_get_int64(*value_); }
  uint32_t uint32_value() const { return cgx_value_get_uint32(*value_); }
  uint64_t uint64_value() const { return cgx_value_get_uint64(*value_); }
  absl::StatusOr<Struct> as_struct() const;
  std::optional<Interface> interface_type(const Package& package) const;
  std::string string() const;

 private:
  Handle<cgx_value> value_;
};

template <typename T>
class Array {
 public:
  Array(Value&& value) : value_(std::move(value)) {}
  Array(Array&& other) : value_(std::move(other.value_)) {}

  const Value& get_value() const { return value_; }

  absl::StatusOr<absl::Span<T>> Acquire();

 protected:
  Value value_;

  std::unique_ptr<HostBuffer> buffer_;
};

template <typename T>
class Atomic {
 public:
  Atomic(Value&& value) : value_(std::move(value)) {}
  Atomic(Atomic&& other) : value_(std::move(other.value_)) {}

  const Value& get_value() const { return value_; }
  virtual absl::StatusOr<T> get() { return value_.get<T>(); };

 protected:
  Value value_;
};

template <typename T>
class DeviceArray : public Array<T> {
 private:
  DeviceArray<T>(const Device& dev, Value&& value)
      : device_(dev), Array<T>(std::move(value)) {}
  const Device& device_;
  friend class Device;
};

template <typename T>
class DeviceAtomic : public Atomic<T> {
 private:
  DeviceAtomic<T>(const Device& dev, Value&& value)
      : device_(dev), Atomic<T>(std::move(value)) {}
  const Device& device_;
  friend class Device;
};

class Device {
 public:
  Device(cgx_device device) : device_(device) {}

  absl::StatusOr<Package> BuildPackage(const std::string& path) const;

  Runtime runtime() const;
  cgx_device raw() const { return *device_; }
  template <typename T>
  absl::StatusOr<DeviceAtomic<T>> Send(T val);
  template <typename T>
  absl::StatusOr<DeviceArray<T>> Send(absl::Span<T> val, const Shape& shape) {
    auto value = Value::Send(*this, shape, val);
    if (!value.ok()) return value.status();
    return DeviceArray<T>(*this, *std::move(value));
  }

 private:
  Handle<cgx_device> device_;
};

class Shape {
 public:
  static absl::StatusOr<Shape> New(enum cgx_value_kind dtype,
                                   const std::vector<int64_t>& axis_lengths);

  Shape(cgx_shape shape) : shape_(shape) {}

  cgx_shape raw() const { return *shape_; }
  int size() const { return cgx_shape_size(*shape_); }
  absl::StatusOr<std::vector<int64_t>> axes() const;

 private:
  Handle<cgx_shape> shape_;
};

class HostBuffer {
 public:
  HostBuffer(cgx_host_buffer buffer) : host_buffer_(buffer), data_(nullptr) {}
  HostBuffer(HostBuffer&& other)
      : host_buffer_(std::move(other.host_buffer_)), data_(other.data_) {
    other.data_ = nullptr;
  }
  HostBuffer(const HostBuffer&) = delete;
  ~HostBuffer() {
    if (data_) cgx_host_buffer_release_data(*host_buffer_, data_);
  };

  cgx_host_buffer raw() const { return *host_buffer_; }
  void* Acquire();

 private:
  Handle<cgx_host_buffer> host_buffer_;
  char* data_;
};

class Struct {
 public:
  struct Field {
    const std::string name;
    const cgx_value_kind kind;
  };

  Struct(cgx_struct strct) : strct_(strct) {}

  absl::StatusOr<Value> GetField(const std::string& name) const;
  absl::Status SetField(const std::string& name, const Value& value);
  absl::StatusOr<std::vector<Field>> ListFields() const;
  cgx_struct raw() const { return *strct_; }

 private:
  Handle<cgx_struct> strct_;
};

class Interface {
 public:
  Interface(cgx_interface iface) : iface_(iface) {}

  std::string name() const;
  std::string package_name() const;
  absl::StatusOr<Function> FindMethod(const std::string& name) const;
  absl::StatusOr<std::vector<Function>> ListMethods() const;

  cgx_interface raw() const { return *iface_; }

 private:
  Handle<cgx_interface> iface_;
};

template <typename T>
absl::StatusOr<absl::Span<T>> Array<T>::Acquire() {
  auto buffer_result = value_.host_buffer();
  if (!buffer_result.ok()) return buffer_result.status();
  buffer_ = std::make_unique<HostBuffer>(*std::move(buffer_result));

  auto shape_result = value_.shape();
  if (!shape_result.ok()) return shape_result.status();

  T* data = reinterpret_cast<T*>(buffer_->Acquire());
  return absl::MakeSpan(data, shape_result->size());
}

class BuildOption {
 public:
  virtual absl::Status apply(const Package&) const = 0;
  virtual ~BuildOption() {};
};

template <typename T>
class SetStaticVariable : public BuildOption {
 public:
  SetStaticVariable(const std::string& package_path, const std::string& name,
                    const T value)
      : package_path_(package_path), name_(name), value_(value) {}
  ~SetStaticVariable() override = default;

  absl::Status apply(const Package& pkg) const override;

 private:
  // TODO(degris): fix CGX and transform PackageCompileSetup to
  // DeviceCompileSetup.
  const std::string package_path_;
  const std::string name_;
  const T value_;
};

template <typename T>
absl::Status SetStaticVariable<T>::apply(const Package& pkg) const {
  auto static_var = pkg.FindStaticVar(name_);
  if (!static_var.ok()) {
    return static_var.status();
  }
  return static_var->set_value(value_);
}

/* Helper methods */

std::string FromHeapCString(const char* cstr);
absl::Status ToErrorStatus(cgx_error err);

}  // namespace cppgx
}  // namespace gxlang

#endif  // THIRD_PARTY_GXLANG_GX_GOLANG_BINDER_CGX_CPPGX_H_
