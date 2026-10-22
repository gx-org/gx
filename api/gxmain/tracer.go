// Copyright 2026 Google LLC
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

package gxmain

import (
	"github.com/gx-org/gx/api/hostio"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/golang/backend/kernels"
)

// tracer collects the values passed to the trace builtin.
type tracer struct {
	tracedOut []hostio.Value
}

func (t *tracer) Trace(file *ir.File, call *ir.FuncCallExpr, vals []hostio.Value) error {
	vals, err := hostio.ToHost(kernels.DefaultAllocator(), vals)
	if err != nil {
		return err
	}
	t.tracedOut = append(t.tracedOut, vals...)
	return nil
}
