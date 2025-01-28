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

package interp

import (
	"github.com/pkg/errors"
	"github.com/gx-org/gx/build/fmterr"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/golang/backend/kernels"
	"github.com/gx-org/gx/interp/elements"
	"github.com/gx-org/gx/interp/state"
)

type (
	stringValuer    struct{}
	stringEvaluator struct {
		errfs fmterr.FileSet
	}
)

var (
	_ valuer    = (*stringValuer)(nil)
	_ evaluator = (*stringEvaluator)(nil)
)

func (v *stringValuer) evaluator(errfs fmterr.FileSet) evaluator {
	return &stringEvaluator{errfs: errfs}
}

func (v *stringValuer) eval(ctx *context, expr ir.Expr) (state.Element, error) {
	return nil, errors.Errorf("expression %T not supported", expr)
}

func (v *stringValuer) toLiteral(ctx *context, expr ir.Expr, arr kernels.Array) (state.Element, error) {
	return nil, errors.Errorf("not supported")
}

func (e *stringEvaluator) scalar(ctx *context, expr ir.StaticExpr) (state.Element, error) {
	switch exprT := expr.(type) {
	case *ir.StringLiteral:
		return elements.NewString(exprT), nil
	default:
		return nil, errors.Errorf("string expression %T not supported", exprT)
	}
}

func (e *stringEvaluator) array(ctx *context, expr *ir.ArrayLitExpr) (state.Element, error) {
	return nil, errors.Errorf("not supported")
}
