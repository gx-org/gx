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

#include "third_party/gxlang/gx/golang/binder/cgx/testing/testing.h"

#include "third_party/absl/status/statusor.h"
#include "third_party/gxlang/gx/golang/binder/cgx/cppgx.h"

namespace gxlang {
namespace cppgx {

absl::StatusOr<Runtime> TestRuntime() {
  const auto result = cgx_testing_runtime();
  if (result.error != cgx_error{}) {
    return ToErrorStatus(result.error);
  }
  return Runtime(result.runtime);
}

}  // namespace cppgx
}  // namespace gxlang
