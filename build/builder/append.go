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

package builder

import (
	"go/ast"

	"github.com/pkg/errors"
	"github.com/gx-org/gx/build/builtins"
	"github.com/gx-org/gx/build/ir"
)

type appendFunc struct{}

const appendName = "append"

var _ ir.FuncImpl = (*appendFunc)(nil)

// BuildFuncType builds the type of a function given how it is called.
func (f *appendFunc) BuildFuncType(fetcher ir.Fetcher, call *ir.CallExpr) (*ir.FuncType, error) {
	ext := &ir.FuncType{
		BaseType: baseType(&ast.FuncType{Func: call.Src.Pos()}),
	}
	params, err := builtins.BuildFuncParams(fetcher, call, appendName, []ir.Type{
		builtins.GenericSliceType,
		nil,
	})
	if err != nil {
		return ext, err
	}
	container := params[0].(*ir.SliceType)
	dtype := container.DType
	for i, arg := range call.Args[1:] {
		argType := arg.Type()
		eq, err := argType.AssignableTo(fetcher, container.DType.Typ)
		if err != nil {
			return ext, errors.Errorf("cannot evaluate the type of argument %d to append", i)
		}
		if !eq {
			return ext, errors.Errorf("cannot use %v as %v in argument %d to append", argType, dtype, i)
		}
	}
	containerGroup := &ir.FieldGroup{
		Type: &ir.TypeValExpr{Typ: params[0]},
	}
	containerGroup.Fields = []*ir.Field{&ir.Field{Group: containerGroup}}
	elementsGroup := &ir.FieldGroup{
		Type: &ir.TypeValExpr{Typ: container.DType.Typ},
	}
	for range len(call.Args) - 1 {
		elementsGroup.Fields = append(elementsGroup.Fields, &ir.Field{
			Group: elementsGroup,
		})
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
