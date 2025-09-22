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

package builtins

import (
	"go/ast"

	"github.com/pkg/errors"
	"github.com/gx-org/gx/build/builtins"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/interp/elements"
)

type setFunc struct{}

var _ ir.FuncImpl = (*setFunc)(nil)

// BuildFuncType builds the type of a function given how it is called.
func (f *setFunc) BuildFuncType(fetcher ir.Fetcher, call *ir.CallExpr) (*ir.FuncType, error) {
	ext := &ir.FuncType{
		BaseType: ir.BaseType[*ast.FuncType]{
			Src: &ast.FuncType{Func: call.Src.Pos()},
		},
	}
	params, err := builtins.BuildFuncParams(fetcher, call, "set", []ir.Type{
		builtins.GenericArrayType,
		builtins.GenericArrayType,
		builtins.PositionsType,
	})
	if err != nil {
		return ext, err
	}
	arrayParams, err := builtins.NarrowTypes[ir.ArrayType](fetcher, call, params)
	if err != nil {
		return ext, errors.Errorf("cannot fetch array type: %v", err)
	}
	ext.Results = builtins.Fields(call, arrayParams[0])
	sameDType, err := arrayParams[0].DataType().Equal(fetcher, arrayParams[1].DataType())
	if err != nil {
		return ext, errors.Errorf("cannot compare datatypes: %v", err)
	}
	if !sameDType {
		return ext, errors.Errorf("cannot set a slice of a [...]%s array with a [...]%s array", arrayParams[0].DataType().String(), arrayParams[1].DataType().String())
	}
	ext.Params = builtins.Fields(call, params...)
	xResolver := arrayParams[0].Rank()
	xRank := xResolver
	updateRank := arrayParams[1].Rank()
	posResolver := arrayParams[2].Rank()
	posRank := posResolver
	if xRank == nil || updateRank == nil || posRank == nil {
		return ext, nil
	}
	if len(posRank.Axes()) != 1 {
		return ext, errors.Errorf("position has an invalid number of axes: got %d but want 1", len(posRank.Axes()))
	}
	posSize, err := elements.EvalInt(fetcher, posRank.Axes()[0])
	if err != nil {
		return ext, err
	}
	if int(posSize) > len(xRank.Axes()) {
		return ext, errors.Errorf("position (length %d) exceeds operand rank (%d)", posSize, len(xRank.Axes()))
	}
	wantUpdate := ir.NewArrayType(&ast.ArrayType{}, arrayParams[0].DataType(), &ir.Rank{
		Ax: xRank.Axes()[posSize:],
	})
	ok, err := arrayParams[1].Equal(fetcher, wantUpdate)
	if err != nil {
		return ext, errors.Errorf("cannot compare rank: %v", err)
	}
	if !ok {
		return ext, errors.Errorf("cannot set array: update slice is %s but requires %s", arrayParams[1], wantUpdate)
	}
	return ext, nil
}

// BuildFuncType builds the type of a function given how it is called.
func (f *setFunc) Implementation() any {
	return nil
}
