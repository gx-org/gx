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

package compeval

import (
	"github.com/pkg/errors"
	"github.com/gx-org/backend/ops"
	"github.com/gx-org/gx/api/values"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/interp/compeval/cpevelements"
	"github.com/gx-org/gx/interp/elements"
	"github.com/gx-org/gx/interp/evaluator"
)

type compArrayOps struct{}

var hostArrayOps evaluator.ArrayOps = &compArrayOps{}

// Graph returns the graph to which new nodes are being added.
func (compArrayOps) Graph() ops.Graph {
	return nil
}

// SubGraph returns a new graph builder.
func (compArrayOps) SubGraph(name string) (evaluator.ArrayOps, error) {
	return nil, errors.Errorf("not implemented")
}

// Einsum calls an einstein sum on x and y given the expression in ref.
func (compArrayOps) Einsum(ctx ir.Evaluator, expr *ir.EinsumExpr, x, y evaluator.NumericalElement) (evaluator.NumericalElement, error) {
	return cpevelements.NewArray(expr.Type().(ir.ArrayType)), nil
}

// BroadcastInDim the data of an array across dimensions.
func (compArrayOps) BroadcastInDim(ctx ir.Evaluator, expr ir.AssignableExpr, x evaluator.NumericalElement, axisLengths []evaluator.NumericalElement) (evaluator.NumericalElement, error) {
	return cpevelements.NewArray(expr.Type().(ir.ArrayType)), nil
}

// Reshape an element into a given shape.
func (compArrayOps) Reshape(expr elements.ExprAt, x evaluator.NumericalElement, axisLengths []evaluator.NumericalElement) (evaluator.NumericalElement, error) {
	return cpevelements.NewArray(expr.Node().Type().(ir.ArrayType)), nil
}

// Concat concatenates scalars elements into an array with one axis.
func (compArrayOps) Concat(ctx ir.Evaluator, expr ir.AssignableExpr, xs []evaluator.NumericalElement) (evaluator.NumericalElement, error) {
	return cpevelements.NewArray(expr.Type().(ir.ArrayType)), nil
}

// Set a slice in an array.
func (compArrayOps) Set(ctx ir.Evaluator, expr *ir.CallExpr, x, updates, index ir.Element) (ir.Element, error) {
	return cpevelements.NewArray(expr.Type().(ir.ArrayType)), nil
}

// ElementFromArray returns an element from an array GX value.
func (compArrayOps) ElementFromArray(ctx ir.Evaluator, expr ir.AssignableExpr, val values.Array) (evaluator.NumericalElement, error) {
	return cpevelements.NewArray(val.Type().(ir.ArrayType)), nil
}
