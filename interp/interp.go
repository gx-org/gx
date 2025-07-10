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

// Package interp evaluates GX code given an evaluator.
//
// All values in the interpreter are represented in elements.
// The GX Context evaluates GX code represented as an
// intermediate representation (IR) tree
// (see [github.com/gx-org/gx/build/ir]),
// evaluates a function given a receiver and arguments passed as interpreter elements.
package interp

import (
	"github.com/gx-org/gx/api/options"
	"github.com/gx-org/gx/api/values"
	"github.com/gx-org/gx/build/fmterr"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/interp/context"
	"github.com/gx-org/gx/interp/elements"
	"github.com/gx-org/gx/interp/evaluator"
)

// FuncBuiltin defines a builtin function provided by a backend.
type FuncBuiltin = context.FuncBuiltin

type interp struct{}

func (interp) EvalExpr(ctx *context.Context, expr ir.Expr) (ir.Element, error) {
	return evalExpr(ctx, expr)
}

func (interp) EvalStmt(ctx *context.Context, block *ir.BlockStmt) ([]ir.Element, bool, error) {
	return evalBlockStmt(ctx, block)
}

// New returns a new interpreter.
func New(eval context.Evaluator, options []options.PackageOption) (*context.Core, error) {
	return context.New(interp{}, eval, options)
}

// EvalFunc evaluates a function.
func EvalFunc(core *context.Core, fn *ir.FuncDecl, in *elements.InputElements) (outs []ir.Element, err error) {
	return context.EvalFunc(core, fn, in)
}

func dimsAsElements(ctx *context.Context, expr ir.AssignableExpr, dims []int) ([]evaluator.NumericalElement, error) {
	els := make([]evaluator.NumericalElement, len(dims))
	for i, di := range dims {
		val, err := values.AtomIntegerValue[int64](ir.IntLenType(), int64(di))
		if err != nil {
			return nil, err
		}
		els[i], err = ctx.Evaluator().ElementFromAtom(ctx, expr, val)
		if err != nil {
			return nil, err
		}
	}
	return els, nil
}

func rankOf(ctx evaluator.Context, src ir.SourceNode, typ ir.ArrayType) (ir.ArrayRank, error) {
	switch rank := typ.Rank().(type) {
	case *ir.Rank:
		return rank, nil
	case *ir.RankInfer:
		if rank.Rnk == nil {
			return nil, fmterr.Errorf(ctx.File().FileSet(), src.Source(), "array rank has not been resolved")
		}
		return rank.Rnk, nil
	default:
		return nil, fmterr.Errorf(ctx.File().FileSet(), src.Source(), "rank %T not supported", rank)
	}
}
