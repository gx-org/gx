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
