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
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/stdlib/builtin"
	"github.com/gx-org/gx/stdlib/impl"
)

type lenFunc struct {
	builtin.Func
}

func (f lenFunc) BuildFuncIR(impl *impl.Stdlib, pkg *ir.Package) (*ir.FuncBuiltin, error) {
	return builtin.IRFuncBuiltin[lenFunc]("Len", impl.Shapes.Len, pkg), nil
}

func (f lenFunc) BuildFuncType(fetcher ir.Fetcher, call *ir.CallExpr) (*ir.FuncType, error) {
	params, err := builtins.BuildFuncParams(fetcher, call, f.Func.Name(), []ir.Type{
		builtins.GenericArrayType,
	})
	if err != nil {
		return nil, err
	}
	return &ir.FuncType{
		Src:     &ast.FuncType{Func: call.Source().Pos()},
		Params:  builtins.Fields(params...),
		Results: builtins.Fields(ir.DefaultIntType),
	}, nil
}
