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
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/build/ir/irkind"
)

type axlengthsFunc struct{}

var axLengthsF = &axlengthsFunc{}

// AxLengths returns the append function builtin.
func AxLengths() ir.FuncImpl {
	return axLengthsF
}

// Name of the builtin function.
func (*axlengthsFunc) Name() string {
	return "axlengths"
}

// BuildFuncType builds the type of a function given how it is called.
func (f *axlengthsFunc) BuildFuncType(fetcher ir.Fetcher, call *ir.FuncCallExpr) (*ir.FuncType, error) {
	ext := &ir.FuncType{
		BaseType: ir.BaseType[*ast.FuncType]{
			Src: &ast.FuncType{Func: call.Src.Pos()},
		},
	}
	if len(call.Args) != 1 {
		return ext, errors.Errorf("wrong number of arguments to axlengths: expected 1, found %d", len(call.Args))
	}
	arg := call.Args[0]
	if arg.Type().Kind() != irkind.Array {
		return ext, errors.Errorf("axlengths(%s) not supported", arg.Type().Kind())
	}

	srcFieldList := &ast.FieldList{Opening: call.Src.Lparen, Closing: call.Src.Rparen}

	paramsGroup := &ir.FieldGroup{
		Src:  &ast.Field{Type: call.Src},
		Type: ir.TypeExpr(nil, arg.Type()),
	}
	paramsGroup.Fields = []*ir.Field{{Group: paramsGroup}}

	resultsGroup := &ir.FieldGroup{
		Src:  &ast.Field{Type: call.Src},
		Type: ir.TypeExpr(nil, ir.IntLenSliceType()),
	}
	resultsGroup.Fields = []*ir.Field{{Group: resultsGroup}}

	ext.Params = &ir.FieldList{
		Src:  srcFieldList,
		List: []*ir.FieldGroup{paramsGroup},
	}
	ext.Results = &ir.FieldList{
		Src:  srcFieldList,
		List: []*ir.FieldGroup{resultsGroup},
	}
	return ext, nil
}

// BuildFuncType builds the type of a function given how it is called.
func (f *axlengthsFunc) Implementation() any {
	return nil
}
