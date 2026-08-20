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
	"github.com/gx-org/gx/build/fmterr"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/interp/constants"
	"github.com/gx-org/gx/internal/undef"
	"github.com/gx-org/gx/interp/engine"
)

// Error is an error produced when a require condition is not satisfied.
type Error struct {
	fmterr.ErrorWithPos
}

// ToError returns true if the error is a required error.
func ToError(err error) *Error {
	rErr, _ := err.(*Error)
	return rErr
}

// Impl is the implementation of the require builtin.
func Impl(env engine.Env, call *ir.FuncCallExpr, recv ir.Element, args []ir.Element) ([]ir.Element, error) {
	condValue, isCondDefined := constants.ConvertOk(constants.CBool, args[0])
	if !isCondDefined {
		return nil, undef.Err()
	}
	if condValue {
		return nil, nil
	}
	from := env.File()
	err := fmterr.Errorf(from.FileSet(), call.Src, "condition %s not satisfied", call.Args[0].SourceString(from))
	return nil, &Error{ErrorWithPos: err}
}
