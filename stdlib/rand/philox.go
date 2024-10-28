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

package rand

import (
	"github.com/gx-org/gx/build/builder"
	"github.com/gx-org/gx/build/builtins"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/stdlib/builtin"
	"github.com/gx-org/gx/stdlib/impl"
)

type philoxUint64 struct {
	builtin.Func
}

func (f philoxUint64) BuildMethodIR(impl *impl.Stdlib, pkg builder.Package, tp *ir.NamedType) (*ir.FuncBuiltin, error) {
	rankType := &ir.SliceType{
		DType: ir.AxisLengthType(),
		Rank:  1,
	}
	philoxType := tp
	valueType := &ir.ArrayType{
		DType: &ir.AtomicType{Knd: ir.Uint64Kind},
		RankF: &ir.GenericRank{},
	}
	fn := &ir.FuncBuiltin{
		Package: pkg.IR(),
		FName:   "Uint64",
		FType: &ir.FuncType{
			Receiver: tp,
			Params: builtins.Fields(
				rankType,
			),
			Results: builtins.Fields(
				philoxType,
				valueType,
			),
		},
	}
	fn.Impl = bootstrapGeneratorNext{Func: builtin.Func{
		Func: fn,
		Impl: impl.Rand.PhiloxUint64,
	}}
	return fn, nil
}

func (f philoxUint64) BuildFuncType(fetcher ir.Fetcher, call *ir.CallExpr) (*ir.FuncType, error) {
	return f.Func.Func.FType, nil
}
