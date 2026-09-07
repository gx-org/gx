// Copyright 2026 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package elements

import (
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/interp/engine"
)

type unknown struct{}

var _ engine.NumericalElement = &unknown{}

var unknownEl = &unknown{}

// Unknown returns an element for which the type is unknown.
// Used as a proxy when a field path is undetermined for example.
func Unknown() ir.Element {
	return unknownEl
}

func (unknown) UnaryOp(env *engine.Env, expr *ir.UnaryExpr) (engine.NumericalElement, error) {
	return unknownEl, nil
}

func (unknown) BinaryOp(env *engine.Env, expr *ir.BinaryExpr, y engine.NumericalElement) (engine.NumericalElement, error) {
	return unknownEl, nil
}

func (unknown) Cast(env *engine.Env, expr ir.Expr, target ir.Type) (engine.NumericalElement, error) {
	return unknownEl, nil
}

func (unknown) Reshape(env *engine.Env, expr ir.Expr, axisLengths []engine.NumericalElement) (engine.NumericalElement, error) {
	return unknownEl, nil
}

func (unknown) Type() ir.Type {
	return ir.UnknownType()
}
