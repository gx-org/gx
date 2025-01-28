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

package shapes

import (
	"go/ast"
	"go/token"

	"github.com/pkg/errors"

	"github.com/gx-org/gx/build/builtins"
	"github.com/gx-org/gx/build/fmterr"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/stdlib/builtin"
	"github.com/gx-org/gx/stdlib/impl"
)

type split struct {
	builtin.Func
}

func (f split) BuildFuncIR(impl *impl.Stdlib, pkg *ir.Package) (*ir.FuncBuiltin, error) {
	return builtin.IRFuncBuiltin[split]("Split", impl.Shapes.Split, pkg), nil
}

func evaluateIntegerExpression(fetcher ir.Fetcher, expr ir.Expr) (int, error) {
	num, unknowns, err := ir.Eval[ir.Int](fetcher, expr)
	if err != nil {
		return 0, err
	}
	if len(unknowns) != 0 {
		return 0, errors.Errorf("unable to evaluate integer expression due to unknowns %s", unknowns)
	}
	return int(num), nil
}

func (f split) checkSplitDimensionSize(fetcher ir.Fetcher, call *ir.CallExpr, axis int, rank *ir.Rank) error {
	numSplits, err := evaluateIntegerExpression(fetcher, call.Args[2])
	if err != nil {
		return fmterr.Errorf(fetcher.FileSet(), call.Source(), "unable to evaluate splits argument to %s: %s", f.Func.Name(), err)
	}

	// Check that the split count divides the dimension.
	splitAxisSize, err := evaluateIntegerExpression(fetcher, rank.Axes[axis])
	if err != nil {
		// We cannot evaluate the axis length. So, the test is skipped.
		return nil
	}
	if splitAxisSize%numSplits != 0 {
		return fmterr.Errorf(fetcher.FileSet(), call.Source(), "dimension size (%d) of axis %d in call to %s must be divisible by the number of splits (%d)", splitAxisSize, axis, f.Func.Name(), numSplits)
	}
	return nil
}

func (f split) BuildFuncType(fetcher ir.Fetcher, call *ir.CallExpr) (*ir.FuncType, error) {
	params, err := builtins.BuildFuncParams(fetcher, call, f.Name(), []ir.Type{
		ir.IntIndexType(),
		builtins.GenericArrayType,
		ir.IntLenType(),
	})
	if err != nil {
		return nil, err
	}

	// Check, if possible, if we can divide the length of the selected axis.
	axis, err := evaluateIntegerExpression(fetcher, call.Args[0])
	if err != nil {
		return nil, fmterr.Errorf(fetcher.FileSet(), call.Source(), "unable to evaluate axis argument to %s: %s", f.Func.Name(), err)
	}
	arrayType, ok := params[1].(ir.ArrayType)
	if !ok {
		return nil, fmterr.Errorf(fetcher.FileSet(), call.Source(), "expected second argument to be an array in call to %s, but got %s", f.Func.Name(), call.Args[0].Type().String())
	}
	rank, err := builtins.RankOf(fetcher, call, arrayType)
	if err != nil {
		return nil, err
	}
	if err := f.checkSplitDimensionSize(fetcher, call, axis, rank); err != nil {
		return nil, err
	}

	// Determine dimensions after split.
	dims := rank.Axes
	outputDims := append([]ir.AxisLength{&ir.AxisExpr{Src: call.Expr(), X: call.Args[2]}}, dims...)
	outputDims[axis+1] = &ir.AxisExpr{
		Src: call.Expr(),
		X:   builtins.ToBinaryExpr(token.QUO, dims[axis], call.Args[2]),
	}
	out := ir.NewArrayType(nil, arrayType.DataType(), &ir.Rank{Axes: outputDims})
	return &ir.FuncType{
		Src:     &ast.FuncType{Func: call.Source().Pos()},
		Params:  builtins.Fields(params...),
		Results: builtins.Fields(out),
	}, nil
}
