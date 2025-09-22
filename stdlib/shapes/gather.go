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
	"github.com/gx-org/gx/interp/elements"
	"github.com/gx-org/gx/stdlib/builtin"
	"github.com/gx-org/gx/stdlib/impl"
)

type gather struct {
	builtin.Func
}

func (f gather) BuildFuncIR(impl *impl.Stdlib, pkg *ir.Package) (*ir.FuncBuiltin, error) {
	return builtin.IRFuncBuiltin[gather]("Gather", impl.Shapes.Gather, pkg), nil
}

func (f gather) BuildFuncType(fetcher ir.Fetcher, call *ir.CallExpr) (*ir.FuncType, error) {
	params, err := builtins.BuildFuncParams(fetcher, call, f.Name(), []ir.Type{
		builtins.GenericArrayType,
		builtins.GenericArrayType,
	})
	if err != nil {
		return nil, err
	}
	sourceArray, err := builtins.NarrowType[ir.ArrayType](fetcher, call, call.Args[0].Type())
	if err != nil {
		return nil, err
	}
	indexArray, err := builtins.NarrowType[ir.ArrayType](fetcher, call, call.Args[1].Type())
	if err != nil {
		return nil, err
	}
	if !ir.IsIndexType(indexArray.DataType()) {
		return nil, fmterr.Errorf(fetcher.File().FileSet(), call.Source(), "Gather requires indices to have an integer type, got shape %q instead", indexArray)
	}

	sourceRank := sourceArray.Rank()
	indicesRank := indexArray.Rank()
	indexDims := indicesRank.Axes()[len(indicesRank.Axes())-1]
	indexRank, err := elements.EvalInt(fetcher, indexDims)
	if err != nil {
		return nil, err
	}

	resultRank := &ir.Rank{}
	resultRank.Ax = append(resultRank.Ax, indicesRank.Axes()[0:1]...)
	resultRank.Ax = append(resultRank.Ax, sourceRank.Axes()[indexRank:]...)
	return &ir.FuncType{
		BaseType: ir.BaseType[*ast.FuncType]{Src: &ast.FuncType{Func: call.Source().Pos()}},
		Params:   builtins.Fields(call, params...),
		Results:  builtins.Fields(call, ir.NewArrayType(&ast.ArrayType{}, sourceArray.DataType(), resultRank)),
	}, nil
}
