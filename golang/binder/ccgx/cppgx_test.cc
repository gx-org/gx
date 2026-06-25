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

#include <cstdint>
#include <limits>
#include <string>
#include <vector>

#include <gmock/gmock.h>
#include <gtest/gtest.h>
#include <absl/status/status.h>
#include <absl/status/statusor.h>
#include <absl/types/span.h>
#include <golang/binder/ccgx/testing/testing.h>
#include <golang/binder/cgx/cgx.h>

namespace gxlang {
namespace cppgx {
namespace {

using ::testing::ElementsAre;
using ::testing::FieldsAre;

static constexpr char kBasicPackage[] =
    "github.com/gx-org/gx/tests/bindings/basic";
static constexpr char kParametersPackage[] =
    "github.com/gx-org/gx/tests/bindings/parameters";
static constexpr char kPkgVarsPackage[] =
    "github.com/gx-org/gx/tests/bindings/pkgvars";
static constexpr char kDTypesPackage[] =
    "github.com/gx-org/gx/tests/bindings/dtypes";

template <typename T>
void compare_array(const Value& val, absl::Span<const T> want) {
  ASSERT_OK_AND_ASSIGN(HostBuffer host_buffer, val.host_buffer());
  const T* data = reinterpret_cast<const T*>(host_buffer.Acquire());
  EXPECT_THAT(absl::MakeSpan(data, want.size()), ::testing::ContainerEq(want));
}

class cppgx : public testing::Test {
 protected:
  cppgx() : start_handle_count_(cgx_handle_count()) {}
  ~cppgx() override {
    // Make sure the test didn't leak any cgx handles; assert the number of
    // outstanding cgx handles is unchanged before and after each test runs:
    AssertHandleCountUnchanged();
  }

  void AssertHandleCountUnchanged() {
    ASSERT_EQ(start_handle_count_, cgx_handle_count())
        << FromHeapCString(cgx_handle_dump());
  }

 private:
  const int64_t start_handle_count_;
};

TEST_F(cppgx, RuntimeNew) {
  ASSERT_OK_AND_ASSIGN(Runtime runtime, TestRuntime());
}

TEST_F(cppgx, GetDevice) {
  ASSERT_OK_AND_ASSIGN(Runtime runtime, TestRuntime());
  ASSERT_OK_AND_ASSIGN(Device device, runtime.GetDevice(0));
}

TEST_F(cppgx, PackageLoad) {
  ASSERT_OK_AND_ASSIGN(Runtime runtime, TestRuntime());
  ASSERT_OK_AND_ASSIGN(PackageIR package, runtime.Load(kBasicPackage));
  EXPECT_EQ("basic", package.name());
  EXPECT_EQ(kBasicPackage, package.fullname());
}

TEST_F(cppgx, PackageBuild) {
  ASSERT_OK_AND_ASSIGN(Runtime runtime, TestRuntime());
  ASSERT_OK_AND_ASSIGN(Device device, runtime.GetDevice(0));
  ASSERT_OK_AND_ASSIGN(Package package, device.BuildPackage(kBasicPackage));
  const auto ir = package.ir();
  EXPECT_EQ("basic", ir.name());
  EXPECT_EQ(kBasicPackage, ir.fullname());
}

TEST_F(cppgx, PackageBuild_NotFound) {
  ASSERT_OK_AND_ASSIGN(Runtime runtime, TestRuntime());
  ASSERT_OK_AND_ASSIGN(Device device, runtime.GetDevice(0));

  absl::StatusOr<Package> result = device.BuildPackage("gx/fake");
  ASSERT_FALSE(result.ok());
  ASSERT_EQ(result.status().message(), "cannot find an importer for gx/fake");
}

TEST_F(cppgx, ListStaticVars) {
  ASSERT_OK_AND_ASSIGN(Runtime runtime, TestRuntime());
  ASSERT_OK_AND_ASSIGN(Device device, runtime.GetDevice(0));
  ASSERT_OK_AND_ASSIGN(Package package, device.BuildPackage(kPkgVarsPackage));
  ASSERT_OK_AND_ASSIGN(std::vector<StaticVar> staticvars,
                       package.ListStaticVars());
  std::vector<std::string> wants{"Var1", "Size"};
  ASSERT_EQ(staticvars.size(), wants.size());
  for (int i = 0; i < staticvars.size(); i++) {
    EXPECT_EQ(staticvars[i].name(), wants[i]);
  }
}

TEST_F(cppgx, ListInterfaces) {
  ASSERT_OK_AND_ASSIGN(Runtime runtime, TestRuntime());
  ASSERT_OK_AND_ASSIGN(Device device, runtime.GetDevice(0));
  ASSERT_OK_AND_ASSIGN(Package package, device.BuildPackage(kBasicPackage));
  ASSERT_OK_AND_ASSIGN(std::vector<Interface> ifaces, package.ListInterfaces());
  std::vector<std::string> wants{"Empty", "Basic"};
  ASSERT_EQ(ifaces.size(), wants.size());
  for (int i = 0; i < ifaces.size(); i++) {
    EXPECT_EQ(ifaces[i].name(), wants[i]);
  }
}

TEST_F(cppgx, FindStaticVar) {
  ASSERT_OK_AND_ASSIGN(Runtime runtime, TestRuntime());
  ASSERT_OK_AND_ASSIGN(Device device, runtime.GetDevice(0));
  ASSERT_OK_AND_ASSIGN(Package package, device.BuildPackage(kPkgVarsPackage));

  ASSERT_TRUE(package.HasStaticVar("Var1"));
  ASSERT_OK_AND_ASSIGN(StaticVar static_var, package.FindStaticVar("Var1"));
  EXPECT_EQ(static_var.name(), "Var1");
}

TEST_F(cppgx, FindStaticVar_NotFound) {
  ASSERT_OK_AND_ASSIGN(Runtime runtime, TestRuntime());
  ASSERT_OK_AND_ASSIGN(Device device, runtime.GetDevice(0));
  ASSERT_OK_AND_ASSIGN(Package package, device.BuildPackage(kPkgVarsPackage));

  ASSERT_FALSE(package.HasStaticVar("Fake"));
  absl::StatusOr<StaticVar> result = package.FindStaticVar("Fake");
  ASSERT_FALSE(result.ok());
  ASSERT_EQ(result.status().message(),
            "static variable Fake not found in package pkgvars");
}

TEST_F(cppgx, SetStaticVar) {
  ASSERT_OK_AND_ASSIGN(Runtime runtime, TestRuntime());
  ASSERT_OK_AND_ASSIGN(Device device, runtime.GetDevice(0));
  ASSERT_OK_AND_ASSIGN(Package package, device.BuildPackage(kPkgVarsPackage));
  ASSERT_OK_AND_ASSIGN(StaticVar static_var, package.FindStaticVar("Var1"));
  ASSERT_OK(static_var.set_value(24));
  ASSERT_OK_AND_ASSIGN(Function function, package.FindFunction("ReturnVar1"));
  ASSERT_OK_AND_ASSIGN(FunctionResult result, function.Run({}));

  auto& return_values = result.return_values();
  ASSERT_EQ(return_values.size(), 1);
  ASSERT_EQ(return_values[0].kind(), CGX_INT32);
  ASSERT_EQ(return_values[0].int32_value(), 24);
}

TEST_F(cppgx, ListFunctions) {
  ASSERT_OK_AND_ASSIGN(Runtime runtime, TestRuntime());
  ASSERT_OK_AND_ASSIGN(Device device, runtime.GetDevice(0));
  ASSERT_OK_AND_ASSIGN(Package package, device.BuildPackage(kBasicPackage));
  ASSERT_OK_AND_ASSIGN(std::vector<Function> functions,
                       package.ListFunctions());
  std::vector<std::string> wants{"ReturnFloat32", "ReturnArrayFloat32",
                                 "ReturnMultiple", "New", "AddPrivate"};
  ASSERT_EQ(functions.size(), wants.size());
  for (int i = 0; i < functions.size(); i++) {
    EXPECT_EQ(functions[i].name(), wants[i]);
  }
}

TEST_F(cppgx, FindFunction) {
  ASSERT_OK_AND_ASSIGN(Runtime runtime, TestRuntime());
  ASSERT_OK_AND_ASSIGN(Device device, runtime.GetDevice(0));
  ASSERT_OK_AND_ASSIGN(Package package, device.BuildPackage(kBasicPackage));

  ASSERT_TRUE(package.HasFunction("ReturnArrayFloat32"));
  ASSERT_OK_AND_ASSIGN(Function function,
                       package.FindFunction("ReturnArrayFloat32"));
  EXPECT_EQ(function.name(), "ReturnArrayFloat32");
  EXPECT_EQ(function.doc(), "ReturnArrayFloat32 returns a float32 tensor.\n");
  EXPECT_EQ(function.str(),
            "func ReturnArrayFloat32() [2]float32 {\n\treturn [2]float32{4.2, "
            "42}\n}");
  EXPECT_EQ(function.num_params(), 0);
}

TEST_F(cppgx, FindFunction_Signature) {
  ASSERT_OK_AND_ASSIGN(Runtime runtime, TestRuntime());
  ASSERT_OK_AND_ASSIGN(Device device, runtime.GetDevice(0));
  ASSERT_OK_AND_ASSIGN(Package package,
                       device.BuildPackage(kParametersPackage));
  ASSERT_OK_AND_ASSIGN(Function function, package.FindFunction("AddInt"));
  ASSERT_OK_AND_ASSIGN(FunctionSignature signature, function.Signature());
  EXPECT_THAT(signature.parameters(), ElementsAre(FieldsAre("x", CGX_INT64),
                                                  FieldsAre("y", CGX_INT64)));
  // GX defaults to "" when no name is given to a result.
  EXPECT_THAT(signature.results(), ElementsAre(FieldsAre("", CGX_INT64)));
}

TEST_F(cppgx, FindFunction_NotFound) {
  ASSERT_OK_AND_ASSIGN(Runtime runtime, TestRuntime());
  ASSERT_OK_AND_ASSIGN(Device device, runtime.GetDevice(0));
  ASSERT_OK_AND_ASSIGN(Package package, device.BuildPackage(kBasicPackage));

  ASSERT_FALSE(package.HasFunction("Fake"));
  absl::StatusOr<Function> result = package.FindFunction("Fake");
  ASSERT_FALSE(result.ok());
  ASSERT_EQ(result.status().message(),
            "function Fake not found in package basic");
}

TEST_F(cppgx, FindInterface_NotFound) {
  ASSERT_OK_AND_ASSIGN(Runtime runtime, TestRuntime());
  ASSERT_OK_AND_ASSIGN(Device device, runtime.GetDevice(0));
  ASSERT_OK_AND_ASSIGN(Package package, device.BuildPackage(kBasicPackage));

  absl::StatusOr<Interface> result = package.FindInterface("Complicated");
  ASSERT_FALSE(result.ok());
  ASSERT_EQ(result.status().message(),
            "type \"Complicated\" not found in package \"basic\"");
}

TEST_F(cppgx, RunFunction) {
  ASSERT_OK_AND_ASSIGN(Runtime runtime, TestRuntime());
  ASSERT_OK_AND_ASSIGN(Device device, runtime.GetDevice(0));
  ASSERT_OK_AND_ASSIGN(Package package, device.BuildPackage(kBasicPackage));
  ASSERT_OK_AND_ASSIGN(Function function,
                       package.FindFunction("ReturnFloat32"));
  ASSERT_OK_AND_ASSIGN(FunctionResult result, function.Run({}));

  auto& return_values = result.return_values();
  ASSERT_EQ(return_values.size(), 1);
  ASSERT_EQ(return_values[0].kind(), CGX_FLOAT32);
  EXPECT_NEAR(return_values[0].float32_value(), 4.2, 0.000001);
}

TEST_F(cppgx, RunFunction_ReturnArrayFloat32) {
  ASSERT_OK_AND_ASSIGN(Runtime runtime, TestRuntime());
  ASSERT_OK_AND_ASSIGN(Device device, runtime.GetDevice(0));
  ASSERT_OK_AND_ASSIGN(Package package, device.BuildPackage(kBasicPackage));
  ASSERT_OK_AND_ASSIGN(Function function,
                       package.FindFunction("ReturnArrayFloat32"));
  ASSERT_OK_AND_ASSIGN(FunctionResult result, function.Run({}));

  const auto& return_values = result.return_values();
  ASSERT_EQ(return_values.size(), 1);
  ASSERT_EQ(return_values[0].kind(), CGX_ARRAY);
  EXPECT_EQ(return_values[0].string(),
            "DeviceArray{DeviceID: 0}: [2]float32{4.2, 42}");
  const auto& iface = return_values[0].interface_type(package);
  ASSERT_FALSE(iface.has_value());

  ASSERT_OK_AND_ASSIGN(const Shape shape, return_values[0].shape());
  EXPECT_EQ(shape.size(), 2);
  ASSERT_OK_AND_ASSIGN(const std::vector<int64_t> axes, shape.axes());
  EXPECT_THAT(axes, ::testing::ElementsAre(2));

  ASSERT_OK_AND_ASSIGN(HostBuffer host_buffer, return_values[0].host_buffer());
  const float* result_data = reinterpret_cast<float*>(host_buffer.Acquire());
  EXPECT_NEAR(result_data[0], 4.2, 0.000001);
  EXPECT_NEAR(result_data[1], 42, 0.000001);
}

TEST_F(cppgx, RunFunction_ReturnMultiple) {
  ASSERT_OK_AND_ASSIGN(Runtime runtime, TestRuntime());
  ASSERT_OK_AND_ASSIGN(Device device, runtime.GetDevice(0));
  ASSERT_OK_AND_ASSIGN(Package package, device.BuildPackage(kBasicPackage));
  ASSERT_OK_AND_ASSIGN(Function function,
                       package.FindFunction("ReturnMultiple"));
  ASSERT_OK_AND_ASSIGN(FunctionResult result, function.Run({}));

  const auto& return_values = result.return_values();
  ASSERT_EQ(return_values.size(), 3);
  ASSERT_EQ(return_values[0].kind(), CGX_INT32);
  EXPECT_EQ(return_values[0].int32_value(), 0);

  ASSERT_EQ(return_values[1].kind(), CGX_FLOAT32);
  EXPECT_NEAR(return_values[1].float32_value(), 1, 0.000001);

  ASSERT_EQ(return_values[2].kind(), CGX_FLOAT64);
  EXPECT_NEAR(return_values[2].float64_value(), 2.71828, 0.000001);
}

TEST_F(cppgx, RunMethod_AddPrivate) {
  ASSERT_OK_AND_ASSIGN(Runtime runtime, TestRuntime());
  ASSERT_OK_AND_ASSIGN(Device device, runtime.GetDevice(0));
  ASSERT_OK_AND_ASSIGN(Package package, device.BuildPackage(kBasicPackage));
  ASSERT_OK_AND_ASSIGN(Function newFunc, package.FindFunction("New"));
  ASSERT_OK_AND_ASSIGN(FunctionResult newResult, newFunc.Run({}));

  const auto& return_values = newResult.return_values();
  ASSERT_EQ(return_values.size(), 1);
  ASSERT_EQ(return_values[0].kind(), CGX_STRUCT);

  const auto& basic = return_values[0].interface_type(package);
  ASSERT_TRUE(basic.has_value());
  EXPECT_EQ(kBasicPackage, basic->package_name());
  EXPECT_EQ("Basic", basic->name());
  ASSERT_OK_AND_ASSIGN(Function addFunc, basic->FindMethod("AddPrivate"));
  ASSERT_OK_AND_ASSIGN(FunctionResult addResult,
                       addFunc.RunMethod(return_values[0], {}));
  EXPECT_EQ(addResult.return_values()[0].int32_value(), 6);
}

TEST_F(cppgx, RunMethod_FindInterface) {
  ASSERT_OK_AND_ASSIGN(Runtime runtime, TestRuntime());
  ASSERT_OK_AND_ASSIGN(Device device, runtime.GetDevice(0));
  ASSERT_OK_AND_ASSIGN(Package package, device.BuildPackage(kBasicPackage));
  ASSERT_OK_AND_ASSIGN(Interface basic, package.FindInterface("Basic"));
  ASSERT_OK_AND_ASSIGN(Function newFunc, package.FindFunction("New"));
  ASSERT_OK_AND_ASSIGN(FunctionResult newResult, newFunc.Run({}));
  const auto& return_values = newResult.return_values();
  ASSERT_OK_AND_ASSIGN(Function addFunc, basic.FindMethod("AddPrivate"));
  ASSERT_OK_AND_ASSIGN(FunctionResult addResult,
                       addFunc.RunMethod(return_values[0], {}));
  EXPECT_EQ(addResult.return_values()[0].int32_value(), 6);
}

TEST_F(cppgx, Interface_ListMethods) {
  ASSERT_OK_AND_ASSIGN(Runtime runtime, TestRuntime());
  ASSERT_OK_AND_ASSIGN(Device device, runtime.GetDevice(0));
  ASSERT_OK_AND_ASSIGN(Package package, device.BuildPackage(kBasicPackage));
  ASSERT_OK_AND_ASSIGN(Interface basic, package.FindInterface("Basic"));
  ASSERT_OK_AND_ASSIGN(std::vector<Function> functions, basic.ListMethods());
  std::vector<std::string> wants{"AddPrivate", "SetFloat"};
  ASSERT_EQ(functions.size(), wants.size());
  for (int i = 0; i < functions.size(); i++) {
    EXPECT_EQ(functions[i].name(), wants[i]);
  }
}

TEST_F(cppgx, RunFunction_AddInt) {
  ASSERT_OK_AND_ASSIGN(Runtime runtime, TestRuntime());
  ASSERT_OK_AND_ASSIGN(Device device, runtime.GetDevice(0));
  ASSERT_OK_AND_ASSIGN(Package package,
                       device.BuildPackage(kParametersPackage));
  ASSERT_OK_AND_ASSIGN(Function function, package.FindFunction("AddInt"));
  ASSERT_OK_AND_ASSIGN(Value v0, Value::FromInt64(device, 24));
  ASSERT_OK_AND_ASSIGN(Value v1, Value::FromInt64(device, 42));
  ASSERT_OK_AND_ASSIGN(FunctionResult result, function.Run({v0, v1}));

  const auto& return_values = result.return_values();
  ASSERT_EQ(return_values.size(), 1);
  ASSERT_EQ(return_values[0].kind(), CGX_INT64);
  EXPECT_EQ(return_values[0].int64_value(), 66);
}

TEST_F(cppgx, RunFunction_MissingParameters) {
  ASSERT_OK_AND_ASSIGN(Runtime runtime, TestRuntime());
  ASSERT_OK_AND_ASSIGN(Device device, runtime.GetDevice(0));
  ASSERT_OK_AND_ASSIGN(Package package,
                       device.BuildPackage(kParametersPackage));
  ASSERT_OK_AND_ASSIGN(Function function, package.FindFunction("AddInt"));
  EXPECT_EQ(function.num_params(), 2);
  EXPECT_EQ(function.param_dtype(0), CGX_INT64);
  EXPECT_EQ(function.param_dtype(1), CGX_INT64);

  absl::StatusOr<FunctionResult> result = function.Run({});
  ASSERT_FALSE(result.ok());
  ASSERT_EQ(result.status().message(),
            "google3/third_party/gxlang/gx/tests/bindings/parameters/"
            "parameters.AddInt evaluation error:\nmissing parameter(s): x, y");
}

absl::StatusOr<Package> BuildPackageHelper() {
  absl::StatusOr<Runtime> runtime(TestRuntime());
  if (!runtime.ok()) return runtime.status();
  absl::StatusOr<Device> device(runtime->GetDevice(0));
  if (!device.ok()) return device.status();
  return device->BuildPackage(kBasicPackage);
}

absl::StatusOr<Function> FindFunctionHelper() {
  absl::StatusOr<Package> package(BuildPackageHelper());
  if (!package.ok()) return package.status();
  return package->FindFunction("New");
}

TEST_F(cppgx, RunFunction_EarlyDestroy_PackageHelper) {
  // BuildPackageHelper allows the Runtime and Device wrapper objects to go out
  // of scope, which causes them to be destroyed. However, the underlying CGX
  // handles and their Go objects continue to live on.
  ASSERT_OK_AND_ASSIGN(Package package, BuildPackageHelper());
  ASSERT_OK_AND_ASSIGN(Function function, package.FindFunction("New"));
  ASSERT_OK_AND_ASSIGN(FunctionResult result, function.Run({}));

  const auto& return_values = result.return_values();
  ASSERT_EQ(return_values.size(), 1);
  ASSERT_EQ(return_values[0].kind(), CGX_STRUCT);
}

TEST_F(cppgx, RunFunction_EarlyDestroy_FunctionHelper) {
  // Similarly, this test further allows the Package wrapper to go out of scope.
  ASSERT_OK_AND_ASSIGN(Function function, FindFunctionHelper());
  ASSERT_OK_AND_ASSIGN(FunctionResult result, function.Run({}));

  const auto& return_values = result.return_values();
  ASSERT_EQ(return_values.size(), 1);
  ASSERT_EQ(return_values[0].kind(), CGX_STRUCT);
}

TEST_F(cppgx, SendValue_Bfloat16Atom) {
  const uint16_t data[] = {0x4049};
  ASSERT_OK_AND_ASSIGN(Runtime runtime, TestRuntime());
  ASSERT_OK_AND_ASSIGN(Device device, runtime.GetDevice(0));
  ASSERT_OK_AND_ASSIGN(Shape shape, Shape::New(CGX_BFLOAT16, {1}));
  ASSERT_OK_AND_ASSIGN(Value value,
                       Value::Send(device, shape, absl::MakeSpan(data)));
  EXPECT_EQ(value.string(), "DeviceArray{DeviceID: 0}: [1]bfloat16{3.140625}");
}

TEST_F(cppgx, SendValue_Float32Array) {
  const float data[] = {4, 2, 4.2};
  ASSERT_OK_AND_ASSIGN(Runtime runtime, TestRuntime());
  ASSERT_OK_AND_ASSIGN(Device device, runtime.GetDevice(0));
  ASSERT_OK_AND_ASSIGN(Shape shape, Shape::New(CGX_FLOAT32, {3}));
  ASSERT_OK_AND_ASSIGN(Value value,
                       Value::Send(device, shape, absl::MakeSpan(data)));
  EXPECT_EQ(value.string(), "DeviceArray{DeviceID: 0}: [3]float32{4, 2, 4.2}");
}

TEST_F(cppgx, SendValue_InvalidLength) {
  const float data[] = {4, 2};
  ASSERT_OK_AND_ASSIGN(Runtime runtime, TestRuntime());
  ASSERT_OK_AND_ASSIGN(Device device, runtime.GetDevice(0));
  ASSERT_OK_AND_ASSIGN(Shape shape, Shape::New(CGX_FLOAT32, {3}));

  absl::StatusOr<Value> result =
      Value::Send(device, shape, absl::MakeSpan(data));
  ASSERT_FALSE(result.ok());
  ASSERT_EQ(result.status().message(),
            "buffer size is 8 but shape specify a buffer size of 12");
}

TEST_F(cppgx, SendValue_InvalidKind) {
  const float data[] = {4, 2};
  ASSERT_OK_AND_ASSIGN(Runtime runtime, TestRuntime());
  ASSERT_OK_AND_ASSIGN(Device device, runtime.GetDevice(0));
  ASSERT_OK_AND_ASSIGN(Shape shape, Shape::New(CGX_INVALID, {2}));

  absl::StatusOr<Value> result =
      Value::Send(device, shape, absl::MakeSpan(data));
  ASSERT_FALSE(result.ok());
  ASSERT_EQ(result.status().message(), "GX invalid data type not supported");
}

TEST_F(cppgx, NewShape_Float32Array) {
  ASSERT_OK_AND_ASSIGN(const Shape shape, Shape::New(CGX_FLOAT32, {3, 4}));
  EXPECT_EQ(shape.size(), 3 * 4);

  ASSERT_OK_AND_ASSIGN(const std::vector<int64_t> axes, shape.axes());
  EXPECT_THAT(axes, ::testing::ElementsAre(3, 4));
}

TEST_F(cppgx, Struct_InvalidConversion) {
  ASSERT_OK_AND_ASSIGN(Runtime runtime, TestRuntime());
  ASSERT_OK_AND_ASSIGN(Device device, runtime.GetDevice(0));
  ASSERT_OK_AND_ASSIGN(Package package, device.BuildPackage(kBasicPackage));
  ASSERT_OK_AND_ASSIGN(Function function,
                       package.FindFunction("ReturnFloat32"));
  ASSERT_OK_AND_ASSIGN(FunctionResult result, function.Run({}));

  const auto& return_values = result.return_values();
  auto struct_value = return_values[0].as_struct();
  ASSERT_FALSE(struct_value.ok());
  ASSERT_EQ(struct_value.status().message(),
            "value has kind float32, expect kind struct");
}

TEST_F(cppgx, Struct_GetField) {
  ASSERT_OK_AND_ASSIGN(Runtime runtime, TestRuntime());
  ASSERT_OK_AND_ASSIGN(Device device, runtime.GetDevice(0));
  ASSERT_OK_AND_ASSIGN(Package package, device.BuildPackage(kBasicPackage));
  ASSERT_OK_AND_ASSIGN(Function function, package.FindFunction("New"));
  ASSERT_OK_AND_ASSIGN(FunctionResult result, function.Run({}));

  const auto& return_values = result.return_values();
  ASSERT_OK_AND_ASSIGN(Struct struct_value, return_values[0].as_struct());
  ASSERT_OK_AND_ASSIGN(Value value, struct_value.GetField("Int"));
  ASSERT_EQ(value.kind(), CGX_INT32);
  EXPECT_EQ(value.int32_value(), 42);
}

TEST_F(cppgx, Struct_SetField) {
  ASSERT_OK_AND_ASSIGN(Runtime runtime, TestRuntime());
  ASSERT_OK_AND_ASSIGN(Device device, runtime.GetDevice(0));
  ASSERT_OK_AND_ASSIGN(Package package, device.BuildPackage(kBasicPackage));
  ASSERT_OK_AND_ASSIGN(Function function, package.FindFunction("New"));
  ASSERT_OK_AND_ASSIGN(FunctionResult result, function.Run({}));

  const auto& return_values = result.return_values();
  ASSERT_OK_AND_ASSIGN(Struct struct_value, return_values[0].as_struct());
  ASSERT_OK_AND_ASSIGN(Value new_value, Value::FromInt32(device, 42));
  ASSERT_OK(struct_value.SetField("Int", new_value));
  ASSERT_OK_AND_ASSIGN(Value value, struct_value.GetField("Int"));
  ASSERT_EQ(value.kind(), CGX_INT32);
  EXPECT_EQ(value.int32_value(), 42);
}

TEST_F(cppgx, Struct_ListFields) {
  ASSERT_OK_AND_ASSIGN(Runtime runtime, TestRuntime());
  ASSERT_OK_AND_ASSIGN(Device device, runtime.GetDevice(0));
  ASSERT_OK_AND_ASSIGN(Package package, device.BuildPackage(kBasicPackage));
  ASSERT_OK_AND_ASSIGN(Function function, package.FindFunction("New"));
  ASSERT_OK_AND_ASSIGN(FunctionResult result, function.Run({}));

  const auto& return_values = result.return_values();
  ASSERT_OK_AND_ASSIGN(Struct struct_value, return_values[0].as_struct());
  ASSERT_OK_AND_ASSIGN(std::vector<Struct::Field> fields,
                       struct_value.ListFields());
  ASSERT_EQ(fields.size(), 7);
  EXPECT_EQ(fields[0].name, "Int");
  EXPECT_EQ(fields[0].kind, CGX_INT32);
}

// DTypes tests.

TEST_F(cppgx, DTypes_Bool) {
  ASSERT_OK_AND_ASSIGN(Runtime runtime, TestRuntime());
  ASSERT_OK_AND_ASSIGN(Device device, runtime.GetDevice(0));
  ASSERT_OK_AND_ASSIGN(Package package, device.BuildPackage(kDTypesPackage));
  ASSERT_OK_AND_ASSIGN(Function function, package.FindFunction("Bool"));
  ASSERT_OK_AND_ASSIGN(Value v0, Value::FromBool(device, true));
  ASSERT_OK_AND_ASSIGN(FunctionResult result, function.Run({v0}));
  const auto& return_values = result.return_values();
  EXPECT_FALSE(return_values[0].bool_value());
}

#define MINFLOAT32 std::numeric_limits<float>::min()
#define MAXFLOAT32 std::numeric_limits<float>::max()

TEST_F(cppgx, DTypes_Float32) {
  ASSERT_OK_AND_ASSIGN(Runtime runtime, TestRuntime());
  ASSERT_OK_AND_ASSIGN(Device device, runtime.GetDevice(0));
  ASSERT_OK_AND_ASSIGN(Package package, device.BuildPackage(kDTypesPackage));
  ASSERT_OK_AND_ASSIGN(Function function, package.FindFunction("Float32"));
  ASSERT_OK_AND_ASSIGN(Value v0, Value::FromFloat32(device, MAXFLOAT32));
  ASSERT_OK_AND_ASSIGN(FunctionResult result, function.Run({v0}));
  const auto& return_values = result.return_values();
  EXPECT_EQ(return_values[0].float32_value(), -MAXFLOAT32);
}

TEST_F(cppgx, DTypes_ArrayFloat32) {
  ASSERT_OK_AND_ASSIGN(Runtime runtime, TestRuntime());
  ASSERT_OK_AND_ASSIGN(Device device, runtime.GetDevice(0));
  ASSERT_OK_AND_ASSIGN(Package package, device.BuildPackage(kDTypesPackage));
  ASSERT_OK_AND_ASSIGN(Function function, package.FindFunction("ArrayFloat32"));
  const float data[] = {-MAXFLOAT32, MAXFLOAT32, -MINFLOAT32, MINFLOAT32};
  ASSERT_OK_AND_ASSIGN(Shape shape, Shape::New(CGX_FLOAT32, {2, 2}));
  ASSERT_OK_AND_ASSIGN(Value value,
                       Value::Send(device, shape, absl::MakeSpan(data)));
  ASSERT_OK_AND_ASSIGN(FunctionResult result, function.Run({value}));
  const float want[] = {MAXFLOAT32, -MAXFLOAT32, MINFLOAT32, -MINFLOAT32};
  compare_array(result.return_values()[0], absl::MakeSpan(want));
}

#define MINFLOAT64 std::numeric_limits<double>::min()
#define MAXFLOAT64 std::numeric_limits<double>::max()

TEST_F(cppgx, DTypes_Float64) {
  ASSERT_OK_AND_ASSIGN(Runtime runtime, TestRuntime());
  ASSERT_OK_AND_ASSIGN(Device device, runtime.GetDevice(0));
  ASSERT_OK_AND_ASSIGN(Package package, device.BuildPackage(kDTypesPackage));
  ASSERT_OK_AND_ASSIGN(Function function, package.FindFunction("Float64"));
  ASSERT_OK_AND_ASSIGN(Value v0, Value::FromFloat64(device, MAXFLOAT64));
  ASSERT_OK_AND_ASSIGN(FunctionResult result, function.Run({v0}));
  const auto& return_values = result.return_values();
  EXPECT_EQ(return_values[0].float64_value(), -MAXFLOAT64);
}

TEST_F(cppgx, DTypes_ArrayFloat64) {
  ASSERT_OK_AND_ASSIGN(Runtime runtime, TestRuntime());
  ASSERT_OK_AND_ASSIGN(Device device, runtime.GetDevice(0));
  ASSERT_OK_AND_ASSIGN(Package package, device.BuildPackage(kDTypesPackage));
  ASSERT_OK_AND_ASSIGN(Function function, package.FindFunction("ArrayFloat64"));
  const double data[] = {-MAXFLOAT64, MAXFLOAT64, -MINFLOAT64, MINFLOAT64};
  ASSERT_OK_AND_ASSIGN(Shape shape, Shape::New(CGX_FLOAT64, {2, 2}));
  ASSERT_OK_AND_ASSIGN(Value value,
                       Value::Send(device, shape, absl::MakeSpan(data)));
  ASSERT_OK_AND_ASSIGN(FunctionResult result, function.Run({value}));
  const double want[] = {MAXFLOAT64, -MAXFLOAT64, MINFLOAT64, -MINFLOAT64};
  compare_array(result.return_values()[0], absl::MakeSpan(want));
}

#define MININT32 std::numeric_limits<int32_t>::min()
#define MAXINT32 std::numeric_limits<int32_t>::max()

TEST_F(cppgx, DTypes_Int32) {
  ASSERT_OK_AND_ASSIGN(Runtime runtime, TestRuntime());
  ASSERT_OK_AND_ASSIGN(Device device, runtime.GetDevice(0));
  ASSERT_OK_AND_ASSIGN(Package package, device.BuildPackage(kDTypesPackage));
  ASSERT_OK_AND_ASSIGN(Function function, package.FindFunction("Int32"));
  ASSERT_OK_AND_ASSIGN(Value v0, Value::FromInt32(device, MAXINT32));
  ASSERT_OK_AND_ASSIGN(FunctionResult result, function.Run({v0}));
  const auto& return_values = result.return_values();
  EXPECT_EQ(return_values[0].int32_value(), -MAXINT32);
}

TEST_F(cppgx, DTypes_ArrayInt32) {
  ASSERT_OK_AND_ASSIGN(Runtime runtime, TestRuntime());
  ASSERT_OK_AND_ASSIGN(Device device, runtime.GetDevice(0));
  ASSERT_OK_AND_ASSIGN(Package package, device.BuildPackage(kDTypesPackage));
  ASSERT_OK_AND_ASSIGN(Function function, package.FindFunction("ArrayInt32"));
  const int32_t data[] = {MININT32 + 1, MAXINT32};
  ASSERT_OK_AND_ASSIGN(Shape shape, Shape::New(CGX_INT32, {2}));
  ASSERT_OK_AND_ASSIGN(Value value,
                       Value::Send(device, shape, absl::MakeSpan(data)));
  ASSERT_OK_AND_ASSIGN(FunctionResult result, function.Run({value}));
  const int32_t want[] = {MAXINT32, MININT32 + 1};
  compare_array(result.return_values()[0], absl::MakeSpan(want));
}

#define MININT64 std::numeric_limits<int64_t>::min()
#define MAXINT64 std::numeric_limits<int64_t>::max()

TEST_F(cppgx, DTypes_Int64) {
  ASSERT_OK_AND_ASSIGN(Runtime runtime, TestRuntime());
  ASSERT_OK_AND_ASSIGN(Device device, runtime.GetDevice(0));
  ASSERT_OK_AND_ASSIGN(Package package, device.BuildPackage(kDTypesPackage));
  ASSERT_OK_AND_ASSIGN(Function function, package.FindFunction("Int64"));
  ASSERT_OK_AND_ASSIGN(Value v0, Value::FromInt64(device, MAXINT64));
  ASSERT_OK_AND_ASSIGN(FunctionResult result, function.Run({v0}));
  const auto& return_values = result.return_values();
  EXPECT_EQ(return_values[0].int64_value(), -MAXINT64);
}

TEST_F(cppgx, DTypes_ArrayInt64) {
  ASSERT_OK_AND_ASSIGN(Runtime runtime, TestRuntime());
  ASSERT_OK_AND_ASSIGN(Device device, runtime.GetDevice(0));
  ASSERT_OK_AND_ASSIGN(Package package, device.BuildPackage(kDTypesPackage));
  ASSERT_OK_AND_ASSIGN(Function function, package.FindFunction("ArrayInt64"));
  const int64_t data[] = {MININT64 + 1, MAXINT64};
  ASSERT_OK_AND_ASSIGN(Shape shape, Shape::New(CGX_INT64, {2}));
  ASSERT_OK_AND_ASSIGN(Value value,
                       Value::Send(device, shape, absl::MakeSpan(data)));
  ASSERT_OK_AND_ASSIGN(FunctionResult result, function.Run({value}));
  const int64_t want[] = {MAXINT64, MININT64 + 1};
  compare_array(result.return_values()[0], absl::MakeSpan(want));
}

#define MAXUINT32 std::numeric_limits<uint32_t>::max()

TEST_F(cppgx, DTypes_Uint32) {
  ASSERT_OK_AND_ASSIGN(Runtime runtime, TestRuntime());
  ASSERT_OK_AND_ASSIGN(Device device, runtime.GetDevice(0));
  ASSERT_OK_AND_ASSIGN(Package package, device.BuildPackage(kDTypesPackage));
  ASSERT_OK_AND_ASSIGN(Function function, package.FindFunction("Uint32"));
  ASSERT_OK_AND_ASSIGN(Value v0, Value::FromUint32(device, MAXUINT32));
  ASSERT_OK_AND_ASSIGN(FunctionResult result, function.Run({v0}));
  const auto& return_values = result.return_values();
  EXPECT_EQ(return_values[0].uint32_value(), -MAXUINT32);
}

TEST_F(cppgx, DTypes_ArrayUint32) {
  ASSERT_OK_AND_ASSIGN(Runtime runtime, TestRuntime());
  ASSERT_OK_AND_ASSIGN(Device device, runtime.GetDevice(0));
  ASSERT_OK_AND_ASSIGN(Package package, device.BuildPackage(kDTypesPackage));
  ASSERT_OK_AND_ASSIGN(Function function, package.FindFunction("ArrayUint32"));
  const uint32_t data[] = {1, MAXUINT32};
  ASSERT_OK_AND_ASSIGN(Shape shape, Shape::New(CGX_UINT32, {2}));
  ASSERT_OK_AND_ASSIGN(Value value,
                       Value::Send(device, shape, absl::MakeSpan(data)));
  ASSERT_OK_AND_ASSIGN(FunctionResult result, function.Run({value}));
  const uint32_t want[] = {MAXUINT32, 1};
  compare_array(result.return_values()[0], absl::MakeSpan(want));
}

#define MAXUINT64 std::numeric_limits<uint64_t>::max()

TEST_F(cppgx, DTypes_Uint64) {
  ASSERT_OK_AND_ASSIGN(Runtime runtime, TestRuntime());
  ASSERT_OK_AND_ASSIGN(Device device, runtime.GetDevice(0));
  ASSERT_OK_AND_ASSIGN(Package package, device.BuildPackage(kDTypesPackage));
  ASSERT_OK_AND_ASSIGN(Function function, package.FindFunction("Uint64"));
  ASSERT_OK_AND_ASSIGN(Value v0, Value::FromUint64(device, MAXUINT64));
  ASSERT_OK_AND_ASSIGN(FunctionResult result, function.Run({v0}));
  const auto& return_values = result.return_values();
  EXPECT_EQ(return_values[0].uint64_value(), -MAXUINT64);
}

TEST_F(cppgx, DTypes_ArrayUint64) {
  ASSERT_OK_AND_ASSIGN(Runtime runtime, TestRuntime());
  ASSERT_OK_AND_ASSIGN(Device device, runtime.GetDevice(0));
  ASSERT_OK_AND_ASSIGN(Package package, device.BuildPackage(kDTypesPackage));
  ASSERT_OK_AND_ASSIGN(Function function, package.FindFunction("ArrayUint64"));
  const uint64_t data[] = {1, MAXUINT64};
  ASSERT_OK_AND_ASSIGN(Shape shape, Shape::New(CGX_UINT64, {2}));
  ASSERT_OK_AND_ASSIGN(Value value,
                       Value::Send(device, shape, absl::MakeSpan(data)));
  ASSERT_OK_AND_ASSIGN(FunctionResult result, function.Run({value}));
  const uint64_t want[] = {MAXUINT64, 1};
  compare_array(result.return_values()[0], absl::MakeSpan(want));
}

}  // namespace
}  // namespace cppgx
}  // namespace gxlang
