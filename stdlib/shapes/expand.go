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

	"github.com/gx-org/gx/build/builtins"
	"github.com/gx-org/gx/build/fmterr"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/stdlib/builtin"
	"github.com/gx-org/gx/stdlib/impl"
)

type expand struct {
	builtin.Func
}

func (f expand) BuildFuncIR(impl *impl.Stdlib, pkg *ir.Package) (*ir.FuncBuiltin, error) {
	return builtin.IRFuncBuiltin[expand]("Expand", impl.Shapes.Expand, pkg), nil
}

func checkExpandRanks(fetcher ir.Fetcher, call *ir.CallExpr, src *ir.Rank, target ir.ArrayRank) error {
	if src == nil || target == nil {
		return nil
	}
	srcAxes := src.Axes
	targetRank, ok := target.(*ir.Rank)
	if !ok {
		return nil
	}
	targetAxes := targetRank.Axes
	if len(srcAxes) != len(targetRank.Axes) {
		return fmterr.Errorf(fetcher.FileSet(), call.Source(), "cannot expand array with %d axes to %d axes: the same number of axes is required", len(srcAxes), len(targetAxes))
	}

	for i, targetExpr := range targetAxes {
		valTarget, unknownsTarget, err := ir.Eval[ir.Int](fetcher, targetExpr)
		if err != nil {
			return err
		}
		valArray, unknownsArray, err := ir.Eval[ir.Int](fetcher, srcAxes[i])
		if err != nil {
			return err
		}
		if unknownsArray != nil || unknownsTarget != nil {
			continue
		}
		if valArray != valTarget && valArray != 1 {
			return fmterr.Errorf(fetcher.FileSet(), srcAxes[i].Source(), "cannot expand array with axis %d of size %d: size of 1 or %d required", i, valArray, valTarget)
		}
	}
	return nil
}

func (f expand) BuildFuncType(fetcher ir.Fetcher, call *ir.CallExpr) (*ir.FuncType, error) {
	params, err := builtins.BuildFuncParams(fetcher, call, f.Name(), []ir.Type{
		builtins.GenericArrayType,
		ir.IntLenSliceType(),
	})
	if err != nil {
		return nil, err
	}
	arrayType, err := builtins.NarrowType[ir.ArrayType](fetcher, call, call.Args[0].Type())
	if err != nil {
		return nil, err
	}
	targetRank := builtins.RankFromExpr(call.Src, call.Args[1])
	srcRank, err := builtins.RankOf(fetcher, call, arrayType)
	if err != nil {
		return nil, err
	}
	if err := checkExpandRanks(fetcher, call, srcRank, targetRank); err != nil {
		return nil, err
	}
	return &ir.FuncType{
		Src:     &ast.FuncType{Func: call.Source().Pos()},
		Params:  builtins.Fields(params...),
		Results: builtins.Fields(ir.NewArrayType(nil, arrayType.DataType(), targetRank)),
	}, nil
}
