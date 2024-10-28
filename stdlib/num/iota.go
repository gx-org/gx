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
	"github.com/gx-org/gx/build/builtins"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/stdlib/builtin"
	"github.com/gx-org/gx/stdlib/impl"
)

type iotaWithAxis struct {
	builtin.Func
}

func (f iotaWithAxis) BuildFuncIR(impl *impl.Stdlib, pkg *ir.Package) (*ir.FuncBuiltin, error) {
	return builtin.IRFuncBuiltin[iotaWithAxis]("Iota", impl.Num.Iota, pkg), nil
}

func (f iotaWithAxis) BuildFuncType(fetcher ir.Fetcher, call *ir.CallExpr) (*ir.FuncType, error) {
	params, err := builtins.BuildFuncParams(fetcher, call, f.Name(), []ir.Type{
		ir.AxisLengthsType(),
		ir.AxisIndexType(),
	})
	if err != nil {
		return nil, err
	}
	rank := builtins.RankFromExpr(call.Src, call.Args[0])
	return &ir.FuncType{
		Params: builtins.Fields(params...),
		Results: builtins.Fields(
			&ir.ArrayType{
				DType: ir.DefaultIntType,
				RankF: rank,
			},
		),
	}, nil
}

type iotaFull struct {
	builtin.Func
}

func (f iotaFull) BuildFuncIR(impl *impl.Stdlib, pkg *ir.Package) (*ir.FuncBuiltin, error) {
	return builtin.IRFuncBuiltin[iotaFull]("IotaFull", impl.Num.IotaFull, pkg), nil
}

func (f iotaFull) BuildFuncType(fetcher ir.Fetcher, call *ir.CallExpr) (*ir.FuncType, error) {
	params, err := builtins.BuildFuncParams(fetcher, call, f.Name(), []ir.Type{
		ir.AxisLengthsType(),
	})
	if err != nil {
		return nil, err
	}
	rank := builtins.RankFromExpr(call.Src, call.Args[0])
	return &ir.FuncType{
		Params: builtins.Fields(params...),
		Results: builtins.Fields(
			&ir.ArrayType{
				DType: ir.DefaultIntType,
				RankF: rank,
			},
		),
	}, nil
}
