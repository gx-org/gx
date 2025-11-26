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

#include "third_party/gxlang/gx/tests/bindings/pkgvars/pkgvars.h"

#include <cstdint>

#include <gmock/gmock.h>
#include <gtest/gtest.h>
#include <absl/types/span.h>
#include <golang/binder/ccgx/cppgx.h>
#include <golang/binder/ccgx/testing/testing.h>
#include <golang/binder/cgx/cgx.h>

using gxlang::cppgx::Array;
using gxlang::cppgx::Atomic;
using gxlang::cppgx::Device;
using gxlang::cppgx::Runtime;
using gxlang::cppgx::TestRuntime;

namespace third_party::gxlang::gx::tests::bindings::pkgvars::pkgvars {
namespace {

class pkgvarsgx : public testing::Test {
 protected:
  pkgvarsgx() : start_handle_count_(cgx_handle_count()) {}
  ~pkgvarsgx() override {
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

TEST_F(pkgvarsgx, ReturnVar1) {
  ASSERT_OK_AND_ASSIGN(Runtime runtime, TestRuntime());
  ASSERT_OK_AND_ASSIGN(Device device, runtime.GetDevice(0));
  auto var1Set = Var1::Set(42);
  ASSERT_OK_AND_ASSIGN(Pkgvars pkg, Pkgvars::BuildFor(device, &var1Set));
  ASSERT_OK_AND_ASSIGN(Atomic<int32_t> var1, pkg.ReturnVar1());
  ASSERT_OK_AND_ASSIGN(int32_t got, var1.get());
  EXPECT_EQ(got, 42);
}

TEST_F(pkgvarsgx, NewTwiceSize) {
  ASSERT_OK_AND_ASSIGN(Runtime runtime, TestRuntime());
  ASSERT_OK_AND_ASSIGN(Device device, runtime.GetDevice(0));
  auto sizeSet = Size::Set(3);
  ASSERT_OK_AND_ASSIGN(Pkgvars pkg, Pkgvars::BuildFor(device, &sizeSet));
  ASSERT_OK_AND_ASSIGN(Array<float> array, pkg.NewTwiceSize());
  ASSERT_OK_AND_ASSIGN(absl::Span<float> got, array.Acquire());
  EXPECT_THAT(got, ::testing::ElementsAre(3, 3, 3, 3, 3, 3));
}

}  // namespace
}  // namespace third_party::gxlang::gx::tests::bindings::pkgvars::pkgvars
