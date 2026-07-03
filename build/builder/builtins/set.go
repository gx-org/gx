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
	"github.com/gx-org/gx/build/ir/irkind"
)

type setFunc struct{}

var setF = &setFunc{}

// Set returns the set function builtin.
func Set() ir.FuncImpl {
	return setF
}

// Name of the builtin function.
func (*setFunc) Name() string {
	return "set"
}

// BuildFuncType builds the type of a function given how it is called.
func (f *setFunc) BuildFuncType(fetcher ir.Fetcher, call *ir.FuncCallExpr) (*ir.FuncType, error) {
	ext := &ir.FuncType{
		BaseType: ir.BaseType[*ast.FuncType]{
			Src: &ast.FuncType{Func: call.Src.Pos()},
		},
	}
	if len(call.Args) < 3 {
		return ext, errors.Errorf("not enough arguments to set: got %d but expect at least 3", len(call.Args))
	}
	args0 := call.Args[0].Type()
	xArray, isArray := ir.Underlying(args0).(ir.ArrayType)
	if !isArray {
		return ext, errors.Errorf("cannot use %s as array as value argument to set", args0.ReferString(fetcher.File()))
	}
	args1 := call.Args[1].Type()
	upArray, isArray := ir.Underlying(args1).(ir.ArrayType)
	if !isArray {
		return ext, errors.Errorf("cannot use %s as array as update argument to set", args1.ReferString(fetcher.File()))
	}
	posCallArgs := call.Args[2:]
	positions := make([]ir.Type, len(posCallArgs))
	for i, posI := range posCallArgs {
		positions[i] = posI.Type()
		if positions[i].Kind() == irkind.NumberInt {
			positions[i] = ir.IntType()
		}
		if !ir.IsIndexType(positions[i]) {
			return ext, errors.Errorf("cannot use %s as position argument to set", positions[i].ReferString(fetcher.File()))
		}
	}
	ext.Results = builtins.Fields(call, xArray)
	sameDType, err := xArray.DataType().Equal(fetcher, upArray.DataType())
	if err != nil {
		return ext, errors.Errorf("cannot compare datatypes: %v", err)
	}
	if !sameDType {
		return ext, errors.Errorf("cannot set a slice of a [...]%s array with a [...]%s array", xArray.DataType().ReferString(fetcher.File()), upArray.DataType().ReferString(fetcher.File()))
	}
	params := append([]ir.Type{xArray, upArray}, positions...)
	ext.Params = builtins.Fields(call, params...)
	xResolver := xArray.Rank()
	xRank := xResolver
	updateRank := upArray.Rank()
	if xRank == nil || updateRank == nil {
		return ext, nil
	}
	if len(positions) > len(xRank.Axes()) {
		return ext, errors.Errorf("position (length %d) exceeds operand rank (%d)", len(positions), len(xRank.Axes()))
	}
	wantUpdate := ir.NewArrayType(&ast.ArrayType{}, xArray.DataType(), &ir.Rank{
		Ax: xRank.Axes()[len(positions):],
	})
	ok, err := upArray.Equal(fetcher, wantUpdate)
	if err != nil {
		return ext, errors.Errorf("cannot compare rank: %v", err)
	}
	if !ok {
		from := fetcher.File()
		return ext, errors.Errorf("cannot set array: update slice is %s but requires %s", upArray.ReferString(from), wantUpdate.ReferString(from))
	}
	return ext, nil
}

// BuildFuncType builds the type of a function given how it is called.
func (f *setFunc) Implementation() any {
	return nil
}
