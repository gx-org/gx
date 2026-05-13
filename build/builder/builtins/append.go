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

	"github.com/gx-org/gx/build/builtins"
	"github.com/gx-org/gx/build/fmterr"
	"github.com/gx-org/gx/build/ir"
)

type appendFunc struct{}

var appendF = &appendFunc{}

// Append returns the append function builtin.
func Append() ir.FuncImpl {
	return appendF
}

// Name of the builtin function.
func (*appendFunc) Name() string {
	return "append"
}

// BuildFuncType builds the type of a function given how it is called.
func (f *appendFunc) BuildFuncType(fetcher ir.Fetcher, call *ir.FuncCallExpr) (*ir.FuncType, error) {
	ext := &ir.FuncType{
		BaseType: ir.BaseType[*ast.FuncType]{
			Src: &ast.FuncType{Func: call.Src.Pos()},
		},
	}
	if len(call.Args) == 0 {
		return ext, fmterr.Errorf(fetcher.File().FileSet(), call.Node(), "not enough arguments for append() (expected 1, found 0)")
	}
	if len(call.Args) == 1 {
		return ext, fmterr.Errorf(fetcher.File().FileSet(), call.Node(), "append with no values")
	}
	params, err := builtins.BuildFuncParams(fetcher, call, f.Name(), []ir.Type{
		builtins.GenericSliceType,
		builtins.VarArgsType,
	})
	if err != nil {
		return ext, err
	}
	container := ir.Underlying(params[0]).(*ir.SliceType)
	containerGroup := &ir.FieldGroup{
		Src:  &ast.Field{Type: call.Src},
		Type: ir.TypeExpr(call, container),
	}
	containerGroup.Fields = []*ir.Field{&ir.Field{Group: containerGroup}}
	ext.VarArgs = &ir.VarArgsType{
		Typ: &ir.SliceType{
			BaseType: ir.BaseType[ast.Expr]{Src: call.Expr()},
			Rank:     container.Rank,
			DType:    container.DType,
		},
	}
	elementsGroup := &ir.FieldGroup{
		Src:  &ast.Field{Type: call.Src},
		Type: ir.TypeExpr(call, ext.VarArgs),
	}
	srcFieldList := &ast.FieldList{Opening: call.Src.Lparen, Closing: call.Src.Rparen}
	ext.Params = &ir.FieldList{
		Src: srcFieldList,
		List: []*ir.FieldGroup{
			containerGroup,
			elementsGroup,
		}}
	ext.Results = &ir.FieldList{
		Src: srcFieldList,
		List: []*ir.FieldGroup{
			containerGroup,
		},
	}
	return ext, nil
}

// BuildFuncType builds the type of a function given how it is called.
func (f *appendFunc) Implementation() any {
	return nil
}
