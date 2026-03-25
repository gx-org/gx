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

package builder

import "github.com/gx-org/gx/build/ir"

func castNil(scope resolveScope, expr ir.Expr, target ir.Type) (ir.Expr, bool) {
	if target != ir.ErrorType() {
		return expr, scope.Err().Appendf(expr.Node(), "cannot cast nil to %s", target.ReferString(scope.fileScope().irFile()))
	}
	return &ir.NilCastExpr{
		X:   expr,
		Typ: target,
	}, true
}
