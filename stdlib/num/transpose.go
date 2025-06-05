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

package num

import (
	"go/ast"
	"slices"

	"github.com/gx-org/gx/build/builtins"
	"github.com/gx-org/gx/build/fmterr"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/stdlib/builtin"
	"github.com/gx-org/gx/stdlib/impl"
)

type transpose struct {
	builtin.Func
}

func (f transpose) BuildFuncIR(impl *impl.Stdlib, pkg *ir.Package) (*ir.FuncBuiltin, error) {
	return builtin.IRFuncBuiltin[transpose]("Transpose", impl.Num.Transpose, pkg), nil
}

func (f transpose) resultsType(fetcher ir.Fetcher, call *ir.CallExpr) (ir.Type, ir.Type, error) {
	if len(call.Args) != 1 {
		return nil, nil, fmterr.Errorf(fetcher.File().FileSet(), call.Source(), "wrong number of argument in call to %s: got %d but want 1", f.Name(), len(call.Args))
	}
	arg := call.Args[0]
	argType := arg.Type()
	arrayType, ok := argType.(ir.ArrayType)
	if !ok {
		return nil, nil, fmterr.Errorf(fetcher.File().FileSet(), call.Source(), "argument type %s not supported in call to %s", arg.Type().String(), f.Name())
	}
	rank := arrayType.Rank()
	inferredAxes := slices.Clone(rank.Axes())
	slices.Reverse(inferredAxes)
	return argType, ir.NewArrayType(
		nil,
		arrayType.DataType(),
		&ir.Rank{Ax: inferredAxes},
	), nil
}

func (f transpose) BuildFuncType(fetcher ir.Fetcher, call *ir.CallExpr) (*ir.FuncType, error) {
	param, result, err := f.resultsType(fetcher, call)
	if err != nil {
		return nil, err
	}
	return &ir.FuncType{
		BaseType: ir.BaseType[*ast.FuncType]{Src: &ast.FuncType{Func: call.Source().Pos()}},
		Params:   builtins.Fields(param),
		Results:  builtins.Fields(result),
	}, nil
}
