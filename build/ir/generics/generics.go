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

// Package generics provides functions to handle generics with the GX IR.
package generics

import (
	"github.com/gx-org/gx/build/ir"
)

// AssignTo checks if an expression can be assigned to a generic non-type parameter.
func AssignTo(fetcher ir.Fetcher, x ir.Expr, genAxis *ir.GenericNonTypeParam, tp ir.Type) bool {
	isAssignable, err := x.Type().AssignableTo(fetcher, tp)
	if err != nil {
		return fetcher.Err().Append(err)
	}
	if !isAssignable {
		from := fetcher.File()
		return fetcher.Err().Appendf(x.Node(), "cannot use %s (type: %s) as %s value in assignment for type parameter %s", x.SourceString(from), x.Type().ReferString(from), tp.ReferString(from), genAxis.NameDef().Name)
	}
	return true
}
