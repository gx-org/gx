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

// Package testalgexpr provides testers for algebraic expressions.
package testalgexpr

import (
	"github.com/gx-org/gx/build/builder/testbuild"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/build/ir/irkind"
)

func buildExpr(ev *testbuild.Evaluator, src string) (ir.Expr, bool) {
	x, err := ev.BuildExpr(src)
	if err != nil {
		return nil, ev.Err().Append(err)
	}
	if !irkind.IsNumber(x.Type().Kind()) {
		return x, true
	}
	return ir.CastNumber(ev, x, ir.IntType())
}

func evalExpr(ev *testbuild.Evaluator, src string) (ir.Element, error) {
	x, ok := buildExpr(ev, src)
	if !ok {
		return nil, ev.Err().Errors().ToError()
	}
	return ev.EvalExpr(x)
}
