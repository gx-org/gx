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
#include <stdlib.h>

#include <stdexcept>

#include <golang/binder/ccgx/cppgx.h>
#include <golang/binder/cgx/cgx.h>
#include <golang/binder/cgx/testing/testing.cgo.h>
#include "third_party/gxlang/gx/golang/binder/pygx/pygx-inl.h"
#include "third_party/pybind11/include/pybind11/pybind11.h"

namespace {

static gxlang::cppgx::Runtime new_runtime() {
  auto result = cgx_testing_runtime();
  gxlang::pygx::maybe_throw_error<std::runtime_error>(result.error);
  return gxlang::cppgx::Runtime(result.runtime);
}

}  // namespace

PYBIND11_MODULE(pygxtesting, m) { m.def("new_runtime", &new_runtime); }
