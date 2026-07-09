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

type lenFunc struct{}

var lenF = &lenFunc{}

// Len returns the len function builtin.
func Len() ir.FuncImpl {
	return lenF
}

// Name of the builtin function.
func (*lenFunc) Name() string {
	return "len"
}

var srcField = &ast.Field{}
var lenRetGroup = &ir.FieldGroup{
	Src:  srcField,
	Type: ir.TypeExpr(nil, ir.IntType()),
}
var lenRetList = &ir.FieldList{
	Src: &ast.FieldList{List: []*ast.Field{
		lenRetGroup.Src,
	}},
	List: []*ir.FieldGroup{lenRetGroup},
}

func init() {
	lenRetGroup.Fields = []*ir.Field{&ir.Field{Group: lenRetGroup}}
}

// BuildFuncType builds the type of a function given how it is called.
func (f *lenFunc) BuildFuncType(fetcher ir.Fetcher, call *ir.FuncCallExpr) (*ir.FuncType, error) {
	ext := &ir.FuncType{
		BaseType: ir.BaseType[*ast.FuncType]{
			Src: &ast.FuncType{Func: call.Src.Pos()},
		},
		Results: lenRetList,
	}
	if len(call.Args) != 1 {
		return ext, errors.Errorf("incorrect number of arguments for %s (expected 1, found %d)", call.SourceString(fetcher.File()), len(call.Args))
	}
	arg := call.Args[0]
	knd := arg.Type().Kind()
	ok := irkind.IsConcrete(knd)
	switch knd {
	case irkind.Invalid:
		ok = true
	case irkind.Slice:
		ok = true
	case irkind.Array:
		ok = true
	case irkind.Rank:
		ok = true
	}
	if !ok {
		return ext, errors.Errorf("invalid argument: %s (%s) for built-in len", arg.SourceString(fetcher.File()), arg.Type().ReferString(fetcher.File()))
	}
	argGroup := &ir.FieldGroup{
		Src:  srcField,
		Type: ir.TypeExpr(nil, ir.AnyType()),
	}
	argGroup.Fields = []*ir.Field{&ir.Field{Group: argGroup}}
	ext.Params = &ir.FieldList{
		Src: &ast.FieldList{List: []*ast.Field{
			argGroup.Src,
		}},
		List: []*ir.FieldGroup{argGroup},
	}
	return ext, nil
}

// Implementation of the len builtin
func (f *lenFunc) Implementation() any {
	return nil
}
