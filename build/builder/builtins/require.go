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

	"github.com/gx-org/gx/build/ir"
)

type requireFunc struct{}

var requireF = &requireFunc{}

// Require returns the require function builtin.
func Require() ir.FuncImpl {
	return requireF
}

// Name of the builtin function.
func (*requireFunc) Name() string {
	return "require"
}

var requireVarargsType = &ir.VarArgsType{
	Typ: &ir.SliceType{
		BaseType: ir.BaseType[ast.Expr]{Src: &ast.Ident{}},
		DType:    ir.TypeExpr(nil, ir.AnyType()),
		Rank:     1,
	},
}
var requireVarargsExpr = ir.TypeExpr(nil, requireVarargsType)

// BuildFuncType builds the type of a function given how it is called.
func (f *requireFunc) BuildFuncType(tpcmp ir.TypeCmp, call *ir.FuncCallExpr) (*ir.FuncType, error) {
	ext := &ir.FuncType{
		BaseType: ir.BaseType[*ast.FuncType]{
			Src: &ast.FuncType{Func: call.Src.Pos()},
		},
		Params: &ir.FieldList{
			Src: &ast.FieldList{},
		},

		Results: &ir.FieldList{
			Src: &ast.FieldList{},
		},
	}
	// Build the condition parameter.
	condGroup := &ir.FieldGroup{
		Src:  srcField,
		Type: ir.TypeExpr(nil, ir.BoolType()),
	}
	condGroup.Fields = []*ir.Field{&ir.Field{
		Name:  &ast.Ident{Name: "cond"},
		Group: condGroup,
	}}
	ext.Params.List = append(ext.Params.List, condGroup)
	if len(call.Args) <= 1 {
		return ext, nil
	}
	// Build the string parameter.
	formatGroup := &ir.FieldGroup{
		Src:  srcField,
		Type: ir.TypeExpr(nil, ir.StringType()),
	}
	formatGroup.Fields = []*ir.Field{&ir.Field{
		Name:  &ast.Ident{Name: "format"},
		Group: formatGroup,
	}}
	ext.Params.List = append(ext.Params.List, formatGroup)
	// Build the varargs
	anysGroup := &ir.FieldGroup{
		Src:  srcField,
		Type: requireVarargsExpr,
	}
	anysGroup.Fields = []*ir.Field{&ir.Field{
		Name:  &ast.Ident{Name: "a"},
		Group: anysGroup,
	}}
	ext.Params.List = append(ext.Params.List, anysGroup)
	ext.VarArgs = requireVarargsType
	return ext, nil
}

// Implementation of the require builtin
func (f *requireFunc) Implementation() any {
	return nil
}
