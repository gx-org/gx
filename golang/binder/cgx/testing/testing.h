#ifndef THIRD_PARTY_GXLANG_GX_GOLANG_BINDER_CGX_TESTING_TESTING_H_
#define THIRD_PARTY_GXLANG_GX_GOLANG_BINDER_CGX_TESTING_TESTING_H_

#include "third_party/absl/status/statusor.h"
#include "third_party/gxlang/gx/golang/binder/cgx/cppgx.h"
#include "third_party/gxlang/gx/golang/binder/cgx/testing/testing.cgo.h"

namespace gxlang {
namespace cppgx {

absl::StatusOr<Runtime> TestRuntime();

}  // namespace cppgx
}  // namespace gxlang

#endif  // THIRD_PARTY_GXLANG_GX_GOLANG_BINDER_CGX_TESTING_TESTING_H_
