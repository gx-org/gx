/*
 * Copyright 2025 Google LLC
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

#ifndef THIRD_PARTY_GXLANG_GX_GOLANG_BINDER_PYGX_PYGX_H_
#define THIRD_PARTY_GXLANG_GX_GOLANG_BINDER_PYGX_PYGX_H_

#include <functional>
#include <stdexcept>
#include <string>
#include <type_traits>
#include <utility>

#include <absl/status/statusor.h>
#include <golang/binder/ccgx/cppgx.h>
#include <golang/binder/cgx/cgx.h>

namespace gxlang {
namespace pygx {

// Type returned by successfully invoking the given function with `Args...`.
template <auto Func, typename... Args>
using unwrap_result_t =
    typename std::invoke_result_t<decltype(Func), Args...>::value_type;

// Handles automatically unwrapping absl::StatusOr after invoking Func.
// If the resulting status indicates an error, throw an exception; else we
// return the wrapped value.
template <auto Func, typename... Args>
auto unwrap_statusor(Args... args) -> unwrap_result_t<Func, Args...> {
  absl::StatusOr<unwrap_result_t<Func, Args...>> result =
      std::invoke(Func, std::forward<Args>(args)...);
  if (!result.ok()) {
    throw std::runtime_error(std::string(result.status().message()));
  }
  return std::move(result).value();
}

// Handles automatically unwrapping absl::StatusOr after invoking T::Method.
// If the resulting status indicates an error, throw an exception; else we
// return the wrapped value.
template <typename T, auto Method, typename... Args>
auto unwrap_statusor(T& obj, Args... args)
    -> unwrap_result_t<Method, T&, Args...> {
  absl::StatusOr<unwrap_result_t<Method, T&, Args...>> result =
      std::invoke(Method, obj, std::forward<Args>(args)...);
  if (!result.ok()) {
    throw std::runtime_error(std::string(result.status().message()));
  }
  return std::move(result).value();
}

// Handles automatically handling absl::Status after invoking T::Method.
// If the resulting status indicates an error, throw an exception.
template <typename T, auto Method, typename... Args>
void unwrap_status(T& obj, Args... args) {
  const absl::Status result =
      std::invoke(Method, obj, std::forward<Args>(args)...);
  if (!result.ok()) {
    throw std::runtime_error(std::string(result.message()));
  }
}

// Throws an exception corresponding to the input cgx_error, if non-empty.
template <typename E>
inline void maybe_throw_error(cgx_error error) {
  if (error != cgx_error{}) {
    const std::string msg = cppgx::FromHeapCString(cgx_error_message(error));
    cgx_release_reference(error);
    throw E(msg);
  }
}

}  // namespace pygx
}  // namespace gxlang

#endif  // THIRD_PARTY_GXLANG_GX_GOLANG_BINDER_PYGX_PYGX_H_
