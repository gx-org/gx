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

// Package require provides the implementation of the require builtin.
package require

import (
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/interp/constants"
	"github.com/gx-org/gx/internal/interp/coreiface"
	"github.com/gx-org/gx/internal/togo"
	"github.com/gx-org/gx/interp/elements"
	"github.com/gx-org/gx/interp/engine"
)

// Impl is the implementation of the require builtin.
func Impl(env *engine.Env, call *ir.FuncCallExpr, recv ir.Element, args []ir.Element) ([]ir.Element, error) {
	condValue, isCondDefined := constants.ConvertOk(constants.CBool, args[0])
	if condValue || !isCondDefined {
		return nil, nil
	}
	if len(args) == 1 {
		from := env.File()
		return nil, ir.CompileErrorF("condition %s not satisfied", call.Args[0].SourceString(from))
	}
	fString, err := elements.StringFromElement(args[1])
	if err != nil {
		return nil, err
	}
	goArgs, err := coreiface.Map(togo.Value, args[2])
	if err != nil {
		return nil, err
	}
	return nil, ir.CompileErrorF(fString, goArgs...)
}
