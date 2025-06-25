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

package math

import (
	"go/ast"

	"github.com/gx-org/gx/build/builtins"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/stdlib/builtin"
	"github.com/gx-org/gx/stdlib/impl"
)

type minFunc struct {
	builtin.Func
}

func (f minFunc) BuildFuncIR(impl *impl.Stdlib, pkg *ir.Package) (*ir.FuncBuiltin, error) {
	return builtin.IRFuncBuiltin[minFunc]("Min", impl.Math.Min, pkg), nil
}

func (f minFunc) BuildFuncType(fetcher ir.Fetcher, call *ir.CallExpr) (*ir.FuncType, error) {
	mainParam, auxParam, result, err := mainAuxArgsToTypes(f.Name(), fetcher, call)
	if err != nil {
		return nil, err
	}
	return &ir.FuncType{
		BaseType: ir.BaseType[*ast.FuncType]{Src: &ast.FuncType{Func: call.Source().Pos()}},
		Params:   builtins.Fields(call, mainParam, auxParam),
		Results:  builtins.Fields(call, result),
	}, nil
}

type maxFunc struct {
	builtin.Func
}

func (f maxFunc) BuildFuncIR(impl *impl.Stdlib, pkg *ir.Package) (*ir.FuncBuiltin, error) {
	return builtin.IRFuncBuiltin[maxFunc]("Max", impl.Math.Max, pkg), nil
}

func (f maxFunc) BuildFuncType(fetcher ir.Fetcher, call *ir.CallExpr) (*ir.FuncType, error) {
	mainParam, auxParam, result, err := mainAuxArgsToTypes(f.Name(), fetcher, call)
	if err != nil {
		return nil, err
	}
	return &ir.FuncType{
		BaseType: ir.BaseType[*ast.FuncType]{Src: &ast.FuncType{Func: call.Source().Pos()}},
		Params:   builtins.Fields(call, mainParam, auxParam),
		Results:  builtins.Fields(call, result),
	}, nil
}
