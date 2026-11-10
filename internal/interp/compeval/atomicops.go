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

package compeval

import (
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/interp/compeval/cpevops"
	"github.com/gx-org/gx/internal/interp/compeval/surrogates/storepath"
	"github.com/gx-org/gx/internal/interp/compeval/surrogates"
	"github.com/gx-org/gx/interp/engine"
)

// atomicOps implements non-constant compeval operations for a constant
// without implementing ConstantElement, breaking recursion when
// NumericalFromHostValue is called under the compeval engine.
//
// New instances of atomicOps are created when compeval evaluates an operator
// on a constant that csteager cannot fold eagerly, such as a binary operator
// between a constant and a symbolic (non-constant) operand (e.g. 2 + a),
// unary/binary/cast operators on boolean or array constants, or reshape,
// index, and slice operators.
type atomicOps struct {
	cst *constant
}

var (
	_ engine.NumericalElement = (*atomicOps)(nil)
	_ engine.Slicer           = (*atomicOps)(nil)
)

func (a *atomicOps) Type() ir.Type {
	return a.cst.Type()
}

func (a *atomicOps) UnaryOp(env *engine.Env, expr *ir.UnaryExpr) (engine.NumericalElement, error) {
	return cpevops.NewUnary(env, expr, a.cst)
}

func (a *atomicOps) BinaryOp(env *engine.Env, expr *ir.BinaryExpr, y engine.NumericalElement) (engine.NumericalElement, error) {
	return cpevops.NewBinaryFrom(env, expr, a.cst, y)
}

func (a *atomicOps) Cast(env *engine.Env, expr ir.Expr, target ir.Type) (engine.NumericalElement, error) {
	return cpevops.NewCast(env, expr, a.cst, target), nil
}

func (a *atomicOps) toArraySurrogate(expr ir.Expr) (engine.NumericalElement, error) {
	path, err := storepath.NewUnique(a.cst)
	if err != nil {
		return nil, err
	}
	return surrogates.NewArrayFrom(path, expr.Type())
}

func (a *atomicOps) Reshape(env *engine.Env, expr ir.Expr, axisLengths []engine.NumericalElement) (engine.NumericalElement, error) {
	return a.toArraySurrogate(expr)
}

func (a *atomicOps) SliceAt(env *engine.Env, expr *ir.IndexExpr, index engine.NumericalElement) (ir.Element, error) {
	return a.toArraySurrogate(expr)
}

func (a *atomicOps) Slice(env *engine.Env, expr *ir.SliceExpr, low, high engine.NumericalElement) (ir.Element, error) {
	return a.toArraySurrogate(expr)
}
