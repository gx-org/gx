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
	"github.com/gx-org/backend/shape"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/interp/compeval/srcstore"
	"github.com/gx-org/gx/internal/interp/compeval/surrogates/storepath"
	"github.com/gx-org/gx/internal/interp/compeval/surrogates"
	"github.com/gx-org/gx/interp/elements"
	"github.com/gx-org/gx/interp/engine"
)

type compArrayOps struct{}

var hostArrayOps engine.ArrayOps = &compArrayOps{}

// Graph returns the graph to which new nodes are being added.
func (compArrayOps) Graph() ops.Graph {
	return nil
}

// SubGraph returns a new graph builder.
func (compArrayOps) SubGraph(name string, args []*shape.Shape) (engine.ArrayOps, error) {
	return nil, errors.Errorf("not implemented")
}

// Einsum calls an einstein sum on x and y given the expression in ref.
func (compArrayOps) Einsum(ctx ir.Evaluator, expr *ir.EinsumExpr, x, y engine.NumericalElement) (engine.NumericalElement, error) {
	path := storepath.NewUniqueIR(expr)
	return surrogates.NewArrayFrom(path, expr.Typ)
}

// BroadcastInDim the data of an array across dimensions.
func (compArrayOps) BroadcastInDim(ctx ir.Evaluator, expr ir.Expr, x engine.NumericalElement, axisLengths []engine.NumericalElement) (engine.NumericalElement, error) {
	path := storepath.NewUniqueIR(expr)
	return surrogates.NewArrayFrom(path, expr.Type())
}

// Reshape an element into a given shape.
func (compArrayOps) Reshape(expr elements.ExprAt, x engine.NumericalElement, axisLengths []engine.NumericalElement) (engine.NumericalElement, error) {
	path := storepath.NewUniqueIR(expr.Node())
	return surrogates.NewArrayFrom(path, expr.Node().Type())
}

// Concat concatenates scalars elements into an array with one axis.
func (compArrayOps) Concat(ctx ir.Evaluator, expr ir.Expr, xs []engine.NumericalElement) (engine.NumericalElement, error) {
	path := storepath.NewUniqueIR(expr)
	return surrogates.NewArrayFrom(path, expr.Type())
}

// Set a slice in an array.
func (compArrayOps) Set(ctx ir.Evaluator, expr *ir.FuncCallExpr, x, updates ir.Element, positions []ir.Element) (ir.Element, error) {
	path := storepath.NewUniqueIR(expr)
	return surrogates.NewArrayFrom(path, expr.Type())
}

// ElementFromHostValue returns transforms an atomic literal element into an element specific to the ArrayOps implementation.
func (ao compArrayOps) ElementFromHostValue(ctx ir.Evaluator, el engine.Constant) (engine.ConstantElement, error) {
	return newConstant(ao, el), nil
}

// DefineGlobalConst defines a global constant for the interpreter.
func (ao compArrayOps) DefineGlobalConst(c ir.Storage, el ir.Element) (ir.Element, error) {
	return srcstore.Link(c, el)
}
