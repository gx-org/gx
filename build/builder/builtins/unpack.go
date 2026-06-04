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

package builtins

import (
	"github.com/pkg/errors"
	"github.com/gx-org/gx/build/ir"
)

type unpackFunc struct{}

var unpackF = unpackFunc{}

// Unpack returns the macro function.
func Unpack() BuiltinMacro {
	return unpackF
}

// Name of the builtin function.
func (unpackFunc) Name() string {
	return "unpack"
}

func (unpackFunc) Impl() ir.MacroKeywordImpl {
	return unpackImpl
}

func unpackImpl(from *ir.File, expr ir.Expr) (ir.Expr, error) {
	sliceTyp, isSliceType := ir.Underlying(expr.Type()).(*ir.SliceType)
	if !isSliceType {
		return expr, errors.Errorf("%s is not a slice (type %s)", expr.SourceString(from), expr.Type().ReferString(from))
	}
	eltTyp, ok := sliceTyp.ElementType()
	if !ok {
		return expr, errors.Errorf("cannot index type %s", expr.Type().ReferString(from))
	}
	return &ir.UnpackExpr{X: expr, EltTyp: eltTyp}, nil
}
