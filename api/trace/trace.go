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

// Package trace is the API to use the builtin `trace` in GX.
package trace

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/gx-org/gx/api/hostio"
	gxfmt "github.com/gx-org/gx/base/fmt"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/golang/backend/kernels"
)

// Callback is called when the trace builtin is invoked
// in GX source code.
type Callback interface {
	Trace(file *ir.File, call *ir.FuncCallExpr, values []hostio.Value) error
}

// Call is a call to the trace builtin. Its values have been transferred to the host.
type Call struct {
	File   *ir.File
	Expr   *ir.FuncCallExpr
	Values []hostio.Value
}

// Recorder is a Callback recording the calls to the trace builtin.
type Recorder struct {
	Calls []Call
}

var _ Callback = (*Recorder)(nil)

// Trace transfers the values to the host and records the call.
func (r *Recorder) Trace(file *ir.File, call *ir.FuncCallExpr, vals []hostio.Value) error {
	hostVals, err := hostio.ToHost(kernels.DefaultAllocator(), vals)
	if err != nil {
		return err
	}
	r.Calls = append(r.Calls, Call{
		File:   file,
		Expr:   call,
		Values: hostVals,
	})
	return nil
}

// Format returns a string representation of the calls to the trace builtin,
// or an empty string if calls is empty.
func Format(calls []Call) string {
	if len(calls) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("Trace:\n")
	for i, call := range calls {
		pos := call.File.FileSet().Position(call.Expr.Src.Pos())
		fmt.Fprintf(&b, "%s:%d\n", filepath.Base(pos.Filename), i)
		for _, val := range call.Values {
			valS := ir.DefineString(call.File, val)
			valS = gxfmt.Indent(valS)
			b.WriteString(valS)
			b.WriteString("\n")
		}
	}
	return b.String()
}
