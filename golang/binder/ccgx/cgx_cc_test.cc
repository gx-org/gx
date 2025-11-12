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

#include <stdlib.h>

#include <cstdint>
#include <iterator>
#include <string>

#include <gtest/gtest.h>
#include <golang/binder/cgx/cgx.cgo.h>
#include <golang/binder/cgx/cgx.h>
#include <golang/binder/cgx/testing/testing.cgo.h>

namespace cgx {
namespace {

static constexpr char kBasicPackage[] =
    "github.com/gx-org/gx/tests/bindings/basic";
static constexpr char kParametersPackage[] =
    "github.com/gx-org/gx/tests/bindings/parameters";
static constexpr char kPkgVarsPackage[] =
    "github.com/gx-org/gx/tests/bindings/pkgvars";

std::string FromHeapCString(const char* cstr) {
  const std::string result(cstr);
  free((void*)cstr);
  return result;
}

std::string ErrorMessage(cgx_error err) {
  return FromHeapCString(cgx_error_message(err));
}

#define CGX_ASSERT_OK(err) ASSERT_EQ(err, cgx_error{}) << ErrorMessage(err)
#define CGX_ASSERT_ERROR(err, msg)   \
  ASSERT_NE(err, cgx_error{});       \
  EXPECT_EQ(ErrorMessage(err), msg); \
  cgx_release_reference(err)

struct build_result {
  cgx_error error;
  cgx_package package;
};

class cgx : public testing::Test {
 protected:
  void SetUp() override {
    start_handle_count_ = cgx_handle_count();

    // Nearly every cgx test needs a runtime and a device; the Go-based
    // implementation suffices for most of them.
    const auto runtime_result = cgx_testing_runtime();
    CGX_ASSERT_OK(runtime_result.error);
    runtime_ = runtime_result.runtime;

    const auto device_result = cgx_device_get(runtime_, 0);
    CGX_ASSERT_OK(device_result.error);
    device_ = device_result.device;
  }

  struct build_result BuildPackage(cchar_t* path) {
    const auto package_load_result = cgx_package_ir_load(runtime(), path);
    struct build_result result;
    result.error = package_load_result.error;
    if (package_load_result.package != cgx_package{}) {
      result.package =
          cgx_package_ir_build_for(package_load_result.package, device());
      cgx_release_reference(package_load_result.package);
    }
    return result;
  }

  void TearDown() override {
    // Make sure to free runtime handles when done -- this is necessary for
    // nearly every cgx type. It's generally good practice to free them in the
    // reverse order they were acquired in.
    cgx_release_reference(device_);
    runtime_ = cgx_release_reference(runtime_);

    // Make sure the test didn't leak any cgx handles.
    ASSERT_EQ(start_handle_count_, cgx_handle_count())
        << FromHeapCString(cgx_handle_dump());
  }

  cgx_runtime runtime() { return runtime_; }
  cgx_device device() { return device_; }

 private:
  int64_t start_handle_count_;

  cgx_runtime runtime_;
  cgx_device device_;
};

TEST_F(cgx, DeviceGet) {
  // Note that the Go runtime never fails to get a device, so we cannot test
  // the failure case here.
  const auto dr = cgx_device_get(runtime(), 0);
  CGX_ASSERT_OK(dr.error);

  cgx_release_reference(dr.device);
}

TEST_F(cgx, PackageLoad) {
  const auto pr = cgx_package_ir_load(runtime(), kBasicPackage);
  CGX_ASSERT_OK(pr.error);
  const std::string name = FromHeapCString(cgx_package_ir_name(pr.package));
  EXPECT_EQ(name, "basic");
  const std::string fullname =
      FromHeapCString(cgx_package_ir_fullname(pr.package));
  EXPECT_EQ(fullname, kBasicPackage);
  cgx_release_reference(pr.package);
}

TEST_F(cgx, PackageBuild_NotFound) {
  const auto pr = cgx_package_ir_load(runtime(), "gx/fake");
  CGX_ASSERT_ERROR(pr.error, "cannot find an importer for gx/fake");
}

TEST_F(cgx, PackageListFunctions) {
  struct build_result pr(BuildPackage(kBasicPackage));
  CGX_ASSERT_OK(pr.error);

  const char* wants[] = {"ReturnFloat32", "ReturnArrayFloat32",
                         "ReturnMultiple", "New", "AddPrivate"};
  auto result = cgx_package_list_functions(pr.package);
  CGX_ASSERT_OK(result.error);
  ASSERT_EQ(result.num_functions, std::size(wants));
  for (int i = 0; i < std::size(wants); i++) {
    const std::string got = FromHeapCString(cgx_function_name(result.funcs[i]));
    EXPECT_EQ(got, wants[i]);
    cgx_release_reference(result.funcs[i]);
  }
  cgx_free_list_functions_result(&result);

  cgx_release_reference(pr.package);
}

TEST_F(cgx, PackageListStaticVars) {
  struct build_result pr(BuildPackage(kPkgVarsPackage));
  CGX_ASSERT_OK(pr.error);

  const char* wants[] = {"Var1", "Size"};
  auto result = cgx_package_list_statics(pr.package);
  CGX_ASSERT_OK(result.error);
  ASSERT_EQ(result.num_statics, std::size(wants));
  for (int i = 0; i < std::size(wants); i++) {
    const std::string got = FromHeapCString(cgx_static_name(result.statics[i]));
    EXPECT_EQ(got, wants[i]);
    cgx_release_reference(result.statics[i]);
  }
  cgx_free_list_statics_result(&result);

  cgx_release_reference(pr.package);
}

TEST_F(cgx, PackageListInterfaces) {
  struct build_result pr(BuildPackage(kBasicPackage));
  CGX_ASSERT_OK(pr.error);

  const char* wants[] = {"Empty", "Basic"};
  auto result = cgx_package_list_interfaces(pr.package);
  CGX_ASSERT_OK(result.error);
  ASSERT_EQ(result.num_ifaces, std::size(wants));
  for (int i = 0; i < std::size(wants); i++) {
    const std::string got =
        FromHeapCString(cgx_interface_name(result.ifaces[i]));
    EXPECT_EQ(got, wants[i]);
    cgx_release_reference(result.ifaces[i]);
  }
  cgx_free_list_interfaces_result(&result);

  cgx_release_reference(pr.package);
}

TEST_F(cgx, PackageFindStaticVar) {
  struct build_result pr(BuildPackage(kPkgVarsPackage));
  CGX_ASSERT_OK(pr.error);

  ASSERT_TRUE(cgx_static_has(pr.package, "Var1"));
  const auto vr = cgx_static_find(pr.package, "Var1");
  CGX_ASSERT_OK(vr.error);

  const std::string vr_name = FromHeapCString(cgx_static_name(vr.static_var));
  EXPECT_EQ(vr_name, "Var1");

  cgx_release_reference(vr.static_var);
  cgx_release_reference(pr.package);
}

TEST_F(cgx, PackageFindStaticVarNotFound) {
  struct build_result pr(BuildPackage(kPkgVarsPackage));
  CGX_ASSERT_OK(pr.error);

  ASSERT_FALSE(cgx_static_has(pr.package, "Fake"));
  const auto fr = cgx_static_find(pr.package, "Fake");
  CGX_ASSERT_ERROR(fr.error,
                   "static variable Fake not found in package pkgvars");

  cgx_release_reference(pr.package);
}

TEST_F(cgx, PackageSetStaticVar) {
  struct build_result pr(BuildPackage(kPkgVarsPackage));
  CGX_ASSERT_OK(pr.error);

  const auto vr = cgx_static_find(pr.package, "Var1");
  CGX_ASSERT_OK(vr.error);
  cgx_static_set(vr.static_var, 42);

  const auto size = cgx_static_find(pr.package, "Size");
  CGX_ASSERT_OK(size.error);
  const auto err = cgx_static_set(size.static_var, 2);
  CGX_ASSERT_OK(err);

  const auto fr = cgx_function_find(pr.package, "ReturnVar1");
  CGX_ASSERT_OK(fr.error);
  auto rr = cgx_function_run(fr.function, cgx_value{}, 0, nullptr);
  CGX_ASSERT_OK(rr.error);
  ASSERT_EQ(rr.value_size, 1);
  ASSERT_EQ(cgx_value_kind_of(rr.values[0]), CGX_INT32);
  EXPECT_EQ(cgx_value_get_int32(rr.values[0]), 42);

  cgx_release_references(rr.values, rr.value_size);
  cgx_free_function_run_result(&rr);
  cgx_release_reference(size.static_var);
  cgx_release_reference(vr.static_var);
  cgx_release_reference(fr.function);
  cgx_release_reference(pr.package);
}

TEST_F(cgx, FindFunction_Signature) {
  struct build_result pr(BuildPackage(kParametersPackage));
  CGX_ASSERT_OK(pr.error);

  const auto fr = cgx_function_find(pr.package, "AddInt");
  CGX_ASSERT_OK(fr.error);

  auto sr = cgx_function_signature(fr.function);
  CGX_ASSERT_OK(sr.error);
  ASSERT_EQ(sr.parameter_size, 2);
  EXPECT_EQ(sr.parameter[0].kind, CGX_INT64);
  EXPECT_STREQ(sr.parameter[0].name, "x");
  EXPECT_EQ(sr.parameter[1].kind, CGX_INT64);
  EXPECT_STREQ(sr.parameter[1].name, "y");
  // GX defaults to "_" when no name is given to a result.
  ASSERT_EQ(sr.result_size, 1);
  EXPECT_EQ(sr.result[0].kind, CGX_INT64);
  EXPECT_STREQ(sr.result[0].name, "_");

  cgx_free_function_signature_result(&sr);
  cgx_release_reference(fr.function);
  cgx_release_reference(pr.package);
}

TEST_F(cgx, RunFunction_ReturnFloat32) {
  struct build_result pr(BuildPackage(kBasicPackage));
  CGX_ASSERT_OK(pr.error);

  const auto fr = cgx_function_find(pr.package, "ReturnFloat32");
  CGX_ASSERT_OK(fr.error);

  auto rr = cgx_function_run(fr.function, cgx_value{}, 0, nullptr);
  CGX_ASSERT_OK(rr.error);

  ASSERT_EQ(rr.value_size, 1);
  ASSERT_EQ(cgx_value_kind_of(rr.values[0]), CGX_FLOAT32);
  EXPECT_NEAR(cgx_value_get_float32(rr.values[0]), 4.2, 0.000001);
  const std::string value_str = FromHeapCString(cgx_value_string(rr.values[0]));
  EXPECT_EQ(value_str, "DeviceArray{DeviceID: 0}: float32(4.2)");

  const cgx_shape shape = cgx_value_shape(rr.values[0]);
  ASSERT_NE(shape, cgx_shape{});
  EXPECT_EQ(cgx_shape_size(shape), 1);
  cgx_shape_axes_result sar = cgx_shape_axes(shape);
  EXPECT_EQ(sar.num_axes, 0);

  cgx_release_references(rr.values, rr.value_size);
  cgx_free_function_run_result(&rr);
  cgx_free_shape_axes_result(&sar);
  cgx_release_reference(shape);
  cgx_release_reference(fr.function);
  cgx_release_reference(pr.package);
}

TEST_F(cgx, RunFunction_ReturnArrayFloat32) {
  struct build_result pr(BuildPackage(kBasicPackage));
  CGX_ASSERT_OK(pr.error);

  const auto fr = cgx_function_find(pr.package, "ReturnArrayFloat32");
  CGX_ASSERT_OK(fr.error);

  auto rr = cgx_function_run(fr.function, cgx_value{}, 0, nullptr);
  CGX_ASSERT_OK(rr.error);
  ASSERT_EQ(rr.value_size, 1);
  ASSERT_EQ(cgx_value_kind_of(rr.values[0]), CGX_ARRAY);

  // Check this is not an interface.
  const auto iface = cgx_value_get_interface_type(device(), rr.values[0]);
  EXPECT_EQ(iface, cgx_interface{});

  const cgx_shape shape = cgx_value_shape(rr.values[0]);
  ASSERT_NE(shape, cgx_shape{});
  EXPECT_EQ(cgx_shape_size(shape), 2);
  EXPECT_EQ(cgx_shape_element_kind(shape), CGX_FLOAT32);
  cgx_shape_axes_result sar = cgx_shape_axes(shape);
  EXPECT_EQ(sar.num_axes, 1);
  EXPECT_EQ(sar.axis_lengths[0], 2);

  const cgx_value_host_buffer_result host_buffer_result =
      cgx_value_host_buffer(rr.values[0]);
  CGX_ASSERT_OK(host_buffer_result.error);
  char* buffer_data = cgx_host_buffer_acquire_data(host_buffer_result.buffer);
  float* float_data = reinterpret_cast<float*>(buffer_data);
  EXPECT_NEAR(float_data[0], 4.2, 0.000001);
  EXPECT_NEAR(float_data[1], 42, 0.000001);

  cgx_host_buffer_release_data(host_buffer_result.buffer, buffer_data);
  cgx_release_reference(host_buffer_result.buffer);
  cgx_free_shape_axes_result(&sar);
  cgx_release_reference(shape);
  cgx_release_references(rr.values, rr.value_size);
  cgx_free_function_run_result(&rr);
  cgx_release_reference(fr.function);
  cgx_release_reference(pr.package);
}

TEST_F(cgx, RunFunction_ReturnMultiple) {
  struct build_result pr(BuildPackage(kBasicPackage));
  CGX_ASSERT_OK(pr.error);

  const auto fr = cgx_function_find(pr.package, "ReturnMultiple");
  CGX_ASSERT_OK(fr.error);

  auto rr = cgx_function_run(fr.function, cgx_value{}, 0, nullptr);
  CGX_ASSERT_OK(rr.error);
  ASSERT_EQ(rr.value_size, 3);

  ASSERT_EQ(cgx_value_kind_of(rr.values[0]), CGX_INT32);
  EXPECT_EQ(cgx_value_get_int32(rr.values[0]), 0);

  ASSERT_EQ(cgx_value_kind_of(rr.values[1]), CGX_FLOAT32);
  EXPECT_NEAR(cgx_value_get_float32(rr.values[1]), 1, 0.000001);

  ASSERT_EQ(cgx_value_kind_of(rr.values[2]), CGX_FLOAT64);
  EXPECT_NEAR(cgx_value_get_float64(rr.values[2]), 2.71828, 0.000001);

  cgx_release_references(rr.values, rr.value_size);
  cgx_free_function_run_result(&rr);
  cgx_release_reference(fr.function);
  cgx_release_reference(pr.package);
}

TEST_F(cgx, RunMethod_AddPrivate) {
  struct build_result pr(BuildPackage(kBasicPackage));
  CGX_ASSERT_OK(pr.error);

  const auto fnewr = cgx_function_find(pr.package, "New");
  CGX_ASSERT_OK(fnewr.error);

  auto rnewr = cgx_function_run(fnewr.function, cgx_value{}, 0, nullptr);
  CGX_ASSERT_OK(rnewr.error);
  ASSERT_EQ(rnewr.value_size, 1);
  ASSERT_EQ(cgx_value_kind_of(rnewr.values[0]), CGX_STRUCT);

  // Check we can call a method on the returned structure.
  const auto iface = cgx_value_get_interface_type(pr.package, rnewr.values[0]);
  ASSERT_NE(iface, cgx_interface{});
  const std::string package_name =
      FromHeapCString(cgx_interface_package_name(iface));
  EXPECT_EQ(package_name, kBasicPackage);
  const std::string type_name = FromHeapCString(cgx_interface_name(iface));
  EXPECT_EQ(type_name, "Basic");
  const auto faddr = cgx_interface_method_find(iface, "AddPrivate");
  CGX_ASSERT_OK(faddr.error);
  auto rsumr = cgx_function_run(faddr.function, rnewr.values[0], 0, nullptr);
  CGX_ASSERT_OK(rsumr.error);
  EXPECT_EQ(cgx_value_get_int32(rsumr.values[0]), 6);

  cgx_release_references(rsumr.values, rsumr.value_size);
  cgx_free_function_run_result(&rsumr);
  cgx_release_reference(faddr.function);
  cgx_release_reference(iface);
  cgx_release_references(rnewr.values, rnewr.value_size);
  cgx_free_function_run_result(&rnewr);
  cgx_release_reference(fnewr.function);
  cgx_release_reference(pr.package);
}

TEST_F(cgx, RunMethod_PackageFindInterface) {
  struct build_result pr(BuildPackage(kBasicPackage));
  CGX_ASSERT_OK(pr.error);
  const auto ifacer = cgx_interface_find(pr.package, "Basic");
  CGX_ASSERT_OK(ifacer.error);
  const auto fnewr = cgx_function_find(pr.package, "New");
  CGX_ASSERT_OK(fnewr.error);
  auto rnewr = cgx_function_run(fnewr.function, cgx_value{}, 0, nullptr);
  CGX_ASSERT_OK(rnewr.error);
  const auto faddr = cgx_interface_method_find(ifacer.iface, "AddPrivate");
  CGX_ASSERT_OK(faddr.error);
  auto rsumr = cgx_function_run(faddr.function, rnewr.values[0], 0, nullptr);
  CGX_ASSERT_OK(rsumr.error);
  EXPECT_EQ(cgx_value_get_int32(rsumr.values[0]), 6);
  cgx_release_references(rsumr.values, rsumr.value_size);
  cgx_free_function_run_result(&rsumr);
  cgx_release_reference(faddr.function);
  cgx_release_references(rnewr.values, rnewr.value_size);
  cgx_free_function_run_result(&rnewr);
  cgx_release_reference(fnewr.function);
  cgx_release_reference(ifacer.iface);
  cgx_release_reference(pr.package);
}

TEST_F(cgx, InterfaceListMethods) {
  struct build_result pr(BuildPackage(kBasicPackage));
  CGX_ASSERT_OK(pr.error);
  const auto ifacer = cgx_interface_find(pr.package, "Basic");
  CGX_ASSERT_OK(ifacer.error);

  const char* wants[] = {"AddPrivate", "SetFloat"};
  auto result = cgx_interface_list_methods(ifacer.iface);
  CGX_ASSERT_OK(result.error);
  ASSERT_EQ(result.num_functions, std::size(wants));
  for (int i = 0; i < std::size(wants); i++) {
    const std::string got = FromHeapCString(cgx_function_name(result.funcs[i]));
    EXPECT_EQ(got, wants[i]);
    cgx_release_reference(result.funcs[i]);
  }
  cgx_free_list_functions_result(&result);
  cgx_release_reference(ifacer.iface);
  cgx_release_reference(pr.package);
}

TEST_F(cgx, RunFunction_AddInt) {
  struct build_result pr(BuildPackage(kParametersPackage));
  CGX_ASSERT_OK(pr.error);

  const auto fr = cgx_function_find(pr.package, "AddInt");
  CGX_ASSERT_OK(fr.error);

  const auto num_args = cgx_function_num_params(fr.function);
  EXPECT_EQ(num_args, 2);
  const auto param0_kind = cgx_function_param_dtype(fr.function, 0);
  EXPECT_EQ(param0_kind, CGX_INT64);
  const auto param1_kind = cgx_function_param_dtype(fr.function, 1);
  EXPECT_EQ(param1_kind, CGX_INT64);

  const auto vr_0 = cgx_value_new_int64(device(), 4);
  CGX_ASSERT_OK(vr_0.error);
  const auto vr_1 = cgx_value_new_int64(device(), 2);
  CGX_ASSERT_OK(vr_1.error);
  cgx_value values[2] = {vr_0.value, vr_1.value};

  auto rr = cgx_function_run(fr.function, cgx_value{}, 2, values);
  CGX_ASSERT_OK(rr.error);
  ASSERT_EQ(rr.value_size, 1);
  ASSERT_EQ(cgx_value_kind_of(rr.values[0]), CGX_INT64);
  EXPECT_EQ(cgx_value_get_int64(rr.values[0]), 6);

  cgx_release_reference(vr_0.value);
  cgx_release_reference(vr_1.value);
  cgx_release_references(rr.values, rr.value_size);
  cgx_free_function_run_result(&rr);
  cgx_release_reference(fr.function);
  cgx_release_reference(pr.package);
}

TEST_F(cgx, RunFunction_MissingParameters) {
  struct build_result pr(BuildPackage(kParametersPackage));
  CGX_ASSERT_OK(pr.error);

  ASSERT_TRUE(cgx_function_has(pr.package, "AddInt"));
  const auto fr = cgx_function_find(pr.package, "AddInt");
  CGX_ASSERT_OK(fr.error);

  const auto rr = cgx_function_run(fr.function, cgx_value{}, 0, nullptr);
  CGX_ASSERT_ERROR(
      rr.error,
      "google3/third_party/gxlang/gx/tests/bindings/parameters/"
      "parameters.AddInt evaluation error:\nmissing parameter(s): x, y");

  cgx_release_reference(fr.function);
  cgx_release_reference(pr.package);
}

TEST_F(cgx, FindInterface_NotFound) {
  struct build_result pr(BuildPackage(kBasicPackage));
  CGX_ASSERT_OK(pr.error);
  const auto ir = cgx_interface_find(pr.package, "Complicated");
  CGX_ASSERT_ERROR(ir.error,
                   "type \"Complicated\" not found in package \"basic\"");
  cgx_release_reference(pr.package);
}

TEST_F(cgx, FindFunction_NotExported) {
  struct build_result pr(BuildPackage(kBasicPackage));
  CGX_ASSERT_OK(pr.error);

  const auto fr = cgx_function_find(pr.package, "fake");
  CGX_ASSERT_ERROR(fr.error, "function \"fake\" not exported");

  cgx_release_reference(pr.package);
}

TEST_F(cgx, FindFunction_NotFound) {
  struct build_result pr(BuildPackage(kBasicPackage));
  CGX_ASSERT_OK(pr.error);

  ASSERT_FALSE(cgx_function_has(pr.package, "Fake"));
  const auto fr = cgx_function_find(pr.package, "Fake");
  CGX_ASSERT_ERROR(fr.error, "function Fake not found in package basic");

  cgx_release_reference(pr.package);
}

TEST_F(cgx, NewValue_Float32) {
  const auto vr = cgx_value_new_float32(device(), 4.2);
  CGX_ASSERT_OK(vr.error);
  ASSERT_EQ(cgx_value_kind_of(vr.value), CGX_FLOAT32);
  EXPECT_NEAR(cgx_value_get_float32(vr.value), 4.2, 0.000001);

  cgx_release_reference(vr.value);
}

TEST_F(cgx, SendValue_Float32Array) {
  const float data[] = {4, 2, 4.2};
  const int64_t axes[] = {3};
  const cgx_shape shape = cgx_shape_new(CGX_FLOAT32, axes, std::size(axes));
  const auto vr = cgx_value_send(device(), shape, data, sizeof(data));
  CGX_ASSERT_OK(vr.error);

  cgx_release_reference(vr.value);
  cgx_release_reference(shape);
}

TEST_F(cgx, SendValue_InvalidLength) {
  const float data[] = {4, 2};
  const int64_t axes[] = {3};
  const cgx_shape shape = cgx_shape_new(CGX_FLOAT32, axes, std::size(axes));
  const auto vr = cgx_value_send(device(), shape, data, sizeof(data));
  CGX_ASSERT_ERROR(vr.error,
                   "buffer size is 8 but shape specify a buffer size of 12");

  cgx_release_reference(shape);
}

TEST_F(cgx, SendValue_InvalidKind) {
  const float data[] = {4, 2};
  const int64_t axes[] = {2};
  const cgx_shape shape = cgx_shape_new(CGX_INVALID, axes, std::size(axes));
  const auto vr = cgx_value_send(device(), shape, data, sizeof(data));
  CGX_ASSERT_ERROR(vr.error, "GX invalid data type not supported");

  cgx_release_reference(shape);
}

TEST_F(cgx, NewShape_Float32Array) {
  const int64_t axes[] = {3, 4};
  const cgx_shape shape = cgx_shape_new(CGX_FLOAT32, axes, std::size(axes));
  EXPECT_EQ(cgx_shape_size(shape), 3 * 4);
  EXPECT_EQ(cgx_shape_element_kind(shape), CGX_FLOAT32);

  cgx_shape_axes_result sar = cgx_shape_axes(shape);
  EXPECT_EQ(sar.num_axes, 2);
  EXPECT_EQ(sar.axis_lengths[0], 3);
  EXPECT_EQ(sar.axis_lengths[1], 4);

  cgx_free_shape_axes_result(&sar);
  cgx_release_reference(shape);
}

TEST_F(cgx, StructGetField) {
  struct build_result pr(BuildPackage(kBasicPackage));
  CGX_ASSERT_OK(pr.error);

  const auto fnewr = cgx_function_find(pr.package, "New");
  CGX_ASSERT_OK(fnewr.error);

  auto rnewr = cgx_function_run(fnewr.function, cgx_value{}, 0, nullptr);
  CGX_ASSERT_OK(rnewr.error);
  ASSERT_EQ(rnewr.value_size, 1);
  ASSERT_EQ(cgx_value_kind_of(rnewr.values[0]), CGX_STRUCT);

  // Read a field from the returned structure.
  const auto gsr = cgx_value_get_struct(rnewr.values[0]);
  ASSERT_NE(gsr.strct, cgx_interface{});
  auto fgr = cgx_struct_field_get(gsr.strct, "Int");
  CGX_ASSERT_OK(fgr.error);
  ASSERT_EQ(cgx_value_kind_of(fgr.value), CGX_INT32);
  EXPECT_EQ(cgx_value_get_int32(fgr.value), 42);

  cgx_release_reference(fgr.value);
  cgx_release_reference(gsr.strct);
  cgx_release_references(rnewr.values, rnewr.value_size);
  cgx_free_function_run_result(&rnewr);
  cgx_release_reference(fnewr.function);
  cgx_release_reference(pr.package);
}

TEST_F(cgx, StructSetField) {
  struct build_result pr(BuildPackage(kBasicPackage));
  CGX_ASSERT_OK(pr.error);

  const auto fnewr = cgx_function_find(pr.package, "New");
  CGX_ASSERT_OK(fnewr.error);

  auto rnewr = cgx_function_run(fnewr.function, cgx_value{}, 0, nullptr);
  CGX_ASSERT_OK(rnewr.error);
  ASSERT_EQ(rnewr.value_size, 1);
  ASSERT_EQ(cgx_value_kind_of(rnewr.values[0]), CGX_STRUCT);

  const auto gsr = cgx_value_get_struct(rnewr.values[0]);
  ASSERT_NE(gsr.strct, cgx_interface{});

  auto new_value = cgx_value_new_int32(device(), 24);
  CGX_ASSERT_OK(new_value.error);
  CGX_ASSERT_OK(cgx_struct_field_set(gsr.strct, "Int", new_value.value));

  const auto fgr = cgx_struct_field_get(gsr.strct, "Int");
  CGX_ASSERT_OK(fgr.error);
  ASSERT_EQ(cgx_value_kind_of(fgr.value), CGX_INT32);
  EXPECT_EQ(cgx_value_get_int32(fgr.value), 24);

  cgx_release_reference(fgr.value);
  cgx_release_reference(new_value.value);
  cgx_release_reference(gsr.strct);
  cgx_release_references(rnewr.values, rnewr.value_size);
  cgx_free_function_run_result(&rnewr);
  cgx_release_reference(fnewr.function);
  cgx_release_reference(pr.package);
}

TEST_F(cgx, StructListFields) {
  struct build_result pr(BuildPackage(kBasicPackage));
  CGX_ASSERT_OK(pr.error);

  const auto fnewr = cgx_function_find(pr.package, "New");
  CGX_ASSERT_OK(fnewr.error);

  auto rnewr = cgx_function_run(fnewr.function, cgx_value{}, 0, nullptr);
  CGX_ASSERT_OK(rnewr.error);
  ASSERT_EQ(rnewr.value_size, 1);
  ASSERT_EQ(cgx_value_kind_of(rnewr.values[0]), CGX_STRUCT);

  const auto gsr = cgx_value_get_struct(rnewr.values[0]);
  ASSERT_NE(gsr.strct, cgx_interface{});

  const char* wants[] = {"Int",      "Float",  "Array", "privateA",
                         "privateB", "length", "index"};
  auto fields = cgx_struct_field_list(gsr.strct);
  CGX_ASSERT_OK(fields.error);
  ASSERT_EQ(fields.field_size, std::size(wants));
  for (int i = 0; i < std::size(wants); i++) {
    EXPECT_STREQ(fields.field[i].name, wants[i]);
  }

  cgx_free_struct_field_list_result(&fields);
  cgx_release_reference(gsr.strct);
  cgx_release_references(rnewr.values, rnewr.value_size);
  cgx_free_function_run_result(&rnewr);
  cgx_release_reference(fnewr.function);
  cgx_release_reference(pr.package);
}

#undef CGX_ASSERT_OK
#undef CGX_ASSERT_ERROR

}  // namespace
}  // namespace cgx
