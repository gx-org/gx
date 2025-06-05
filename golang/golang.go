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

// Package golang provides everything required to run GX from Go.
//
// This includes a bindings generator and GX api in Go.
package golang

import (
	"fmt"
	"go/token"
	"strings"

	"github.com/gx-org/gx/api/trace"
	"github.com/gx-org/gx/api/values"
	"github.com/gx-org/gx/build/ir"
)

// Tracer is a default tracer for GX code.
type Tracer struct {
}

var _ trace.Callback = (*Tracer)(nil)

// Trace prints traced values on the standard output.
func (Tracer) Trace(fset *token.FileSet, call *ir.CallExpr, values []values.Value) error {
	pos := fset.Position(call.Src.Pos())
	fmt.Println(pos.String())
	for _, val := range values {
		valS := strings.ReplaceAll(fmt.Sprint(val), "\n", "\n  ")
		fmt.Println(valS)
	}
	fmt.Println()
	return nil
}
