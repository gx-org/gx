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

// Package csteager provides utility to evaluate constants eagerly.
package csteager

import (
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/interp/coreiface"
	"github.com/gx-org/gx/interp/engine"
)

type (
	// Eval is a constant able to evaluate expressions.
	Eval interface {
		engine.Constant
		// EvalUnary greedily evaluates an unary expression.
		EvalUnary(env engine.Env, expr *ir.UnaryExpr) (engine.Constant, error)
		// EvalBinary greedily evaluates a binary expression.
		// A nil constant needs to be returned if the expression cannot be evaluated.
		EvalBinary(env engine.Env, expr *ir.BinaryExpr, y engine.Constant) (engine.Constant, error)
	}

	// Caster is a constant able to cast to another constant.
	Caster interface {
		Eval
		// EvalCast casts a constant.
		EvalCast(env engine.Env, expr ir.Expr, tp ir.Type) (engine.Constant, error)
	}
)

// Cast evaluates a cast on a given constant.
// Returns a nil element (and no error) if the expression could not be evaluated eagerly.
func Cast(env engine.Env, expr ir.Expr, target ir.Type, x engine.Constant) (engine.Constant, error) {
	xE, xIsEager := x.(Caster)
	if !xIsEager {
		return nil, nil
	}
	return xE.EvalCast(env, expr, target)
}

// Unary evaluates an eager unary expression on a given constant.
// Returns a nil element (and no error) if the expression could not be evaluated eagerly.
func Unary(env engine.Env, expr *ir.UnaryExpr, x engine.Constant) (engine.Constant, error) {
	xE, xIsEager := x.(Eval)
	if !xIsEager {
		return nil, nil
	}
	return xE.EvalUnary(env, expr)
}

// Binary evaluates an eager binary expression on given constants.
// Returns a nil element (and no error) if the expression could not be evaluated eagerly.
func Binary(env engine.Env, expr *ir.BinaryExpr, x engine.Constant, y engine.NumericalElement) (engine.Constant, error) {
	yCst, yIsConstant, err := coreiface.CastOk[engine.ConstantElement](y)
	if !yIsConstant || err != nil {
		return nil, err
	}
	xE, xIsEager := x.(Eval)
	if !xIsEager {
		return nil, nil
	}
	return xE.EvalBinary(env, expr, yCst.Constant())
}
