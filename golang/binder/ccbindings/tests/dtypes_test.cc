// Copyright 2025 Google LLC
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

#include "third_party/gxlang/gx/tests/bindings/dtypes/dtypes.h"

#include <cstdint>
#include <limits>

#include <gmock/gmock.h>
#include <gtest/gtest.h>
#include <absl/types/span.h>
#include <gxdeps/github.com/gx-org/gx/golang/binder/ccgx/cppgx.h>
#include <gxdeps/github.com/gx-org/gx/golang/binder/ccgx/testing/testing.h>
#include <gxdeps/github.com/gx-org/gx/golang/binder/cgx/cgx.h>

using gxlang::cppgx::Array;
using gxlang::cppgx::Atomic;
using gxlang::cppgx::Device;
using gxlang::cppgx::DeviceArray;
using gxlang::cppgx::DeviceAtomic;
using gxlang::cppgx::Runtime;
using gxlang::cppgx::Shape;
using gxlang::cppgx::TestRuntime;

namespace third_party::gxlang::gx::tests::bindings::dtypes::dtypes {
namespace {

class dtypesgx : public testing::Test {
 protected:
  dtypesgx() : start_handle_count_(cgx_handle_count()) {}
  ~dtypesgx() override {
    // Make sure the test didn't leak any cgx handles; assert the number of
    // outstanding cgx handles is unchanged before and after each test runs:
    AssertHandleCountUnchanged();
  }

  void AssertHandleCountUnchanged() {
    ASSERT_EQ(start_handle_count_, cgx_handle_count());
  }

 private:
  const int64_t start_handle_count_;
};

TEST_F(dtypesgx, AtomicBool) {
  ASSERT_OK_AND_ASSIGN(Runtime runtime, TestRuntime());
  ASSERT_OK_AND_ASSIGN(Device device, runtime.GetDevice(0));
  ASSERT_OK_AND_ASSIGN(Dtypes pkg, Dtypes::BuildFor(device));
  ASSERT_OK_AND_ASSIGN(DeviceAtomic<bool> in, device.Send<bool>(true));
  ASSERT_OK_AND_ASSIGN(Atomic<bool> out, pkg.Bool(in));
  ASSERT_OK_AND_ASSIGN(bool got, out.get());
  EXPECT_EQ(got, false);
}

TEST_F(dtypesgx, AtomicInt32) {
  const int32_t max = std::numeric_limits<int32_t>::max();
  ASSERT_OK_AND_ASSIGN(Runtime runtime, TestRuntime());
  ASSERT_OK_AND_ASSIGN(Device device, runtime.GetDevice(0));
  ASSERT_OK_AND_ASSIGN(Dtypes pkg, Dtypes::BuildFor(device));
  ASSERT_OK_AND_ASSIGN(DeviceAtomic<int32_t> in, device.Send<int32_t>(max));
  ASSERT_OK_AND_ASSIGN(Atomic<int32_t> out, pkg.Int32(in));
  ASSERT_OK_AND_ASSIGN(int32_t got, out.get());
  EXPECT_EQ(got, -max);
}

TEST_F(dtypesgx, AtomicInt64) {
  const int64_t max = std::numeric_limits<int64_t>::max();
  ASSERT_OK_AND_ASSIGN(Runtime runtime, TestRuntime());
  ASSERT_OK_AND_ASSIGN(Device device, runtime.GetDevice(0));
  ASSERT_OK_AND_ASSIGN(Dtypes pkg, Dtypes::BuildFor(device));
  ASSERT_OK_AND_ASSIGN(DeviceAtomic<int64_t> in, device.Send<int64_t>(max));
  ASSERT_OK_AND_ASSIGN(Atomic<int64_t> out, pkg.Int64(in));
  ASSERT_OK_AND_ASSIGN(int64_t got, out.get());
  EXPECT_EQ(got, -max);
}

TEST_F(dtypesgx, AtomicFloat32) {
  const float max = std::numeric_limits<float>::max();
  ASSERT_OK_AND_ASSIGN(Runtime runtime, TestRuntime());
  ASSERT_OK_AND_ASSIGN(Device device, runtime.GetDevice(0));
  ASSERT_OK_AND_ASSIGN(Dtypes pkg, Dtypes::BuildFor(device));
  ASSERT_OK_AND_ASSIGN(DeviceAtomic<float> in, device.Send<float>(max));
  ASSERT_OK_AND_ASSIGN(Atomic<float> out, pkg.Float32(in));
  ASSERT_OK_AND_ASSIGN(float got, out.get());
  EXPECT_EQ(got, -max);
}

TEST_F(dtypesgx, AtomicFloat64) {
  const double max = std::numeric_limits<double>::max();
  ASSERT_OK_AND_ASSIGN(Runtime runtime, TestRuntime());
  ASSERT_OK_AND_ASSIGN(Device device, runtime.GetDevice(0));
  ASSERT_OK_AND_ASSIGN(Dtypes pkg, Dtypes::BuildFor(device));
  ASSERT_OK_AND_ASSIGN(DeviceAtomic<double> in, device.Send<double>(max));
  ASSERT_OK_AND_ASSIGN(Atomic<double> out, pkg.Float64(in));
  ASSERT_OK_AND_ASSIGN(double got, out.get());
  EXPECT_EQ(got, -max);
}

TEST_F(dtypesgx, ArrayBool) {
  bool data[] = {true, false, true};
  ASSERT_OK_AND_ASSIGN(Runtime runtime, TestRuntime());
  ASSERT_OK_AND_ASSIGN(Device device, runtime.GetDevice(0));
  ASSERT_OK_AND_ASSIGN(Dtypes pkg, Dtypes::BuildFor(device));
  ASSERT_OK_AND_ASSIGN(Shape shape, Shape::New(CGX_BOOL, {3}));
  ASSERT_OK_AND_ASSIGN(DeviceArray<bool> in,
                       device.Send<bool>(absl::MakeSpan(data), shape));
  ASSERT_OK_AND_ASSIGN(Array<bool> out, pkg.ArrayBool(in));
  ASSERT_OK_AND_ASSIGN(absl::Span<bool> got, out.Acquire());
  EXPECT_THAT(got, ::testing::ElementsAre(false, true, false));
}

}  // namespace
}  // namespace third_party::gxlang::gx::tests::bindings::dtypes::dtypes
