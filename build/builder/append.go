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

	"github.com/gx-org/gx/build/builtins"
	"github.com/gx-org/gx/build/ir"
)

type appendFunc struct {
	ext ir.FuncBuiltin
}

var (
	_ genericCallTypeNode = (*appendFunc)(nil)
	_ function            = (*appendFunc)(nil)
)

func (f *appendFunc) resolveGenericCallType(scope scoper, src ast.Node, fetcher ir.Fetcher, call *ir.CallExpr) (*funcType, bool) {
	params, err := builtins.BuildFuncParams(fetcher, call, f.String(), []ir.Type{
		builtins.GenericSliceType,
		nil,
	})
	if err != nil {
		scope.err().Append(err)
		return nil, false
	}
	container := params[0].(*ir.SliceType)
	dtype := container.DType
	for i, arg := range call.Args[1:] {
		argType := arg.Type()
		eq, err := argType.AssignableTo(fetcher, container.DType)
		if err != nil {
			scope.err().Appendf(src, "cannot evaluate the type of argument %d to append", i)
			return nil, false
		}
		if !eq {
			scope.err().Appendf(src, "cannot use %v as %v in argument to append", argType, dtype)
			return nil, false
		}
	}
	containerGroup := &ir.FieldGroup{Type: params[0]}
	containerGroup.Fields = []*ir.Field{&ir.Field{Group: containerGroup}}
	elementsGroup := &ir.FieldGroup{Type: container.DType}
	for range len(call.Args) - 1 {
		elementsGroup.Fields = append(elementsGroup.Fields, &ir.Field{
			Group: elementsGroup,
		})
	}
	srcFieldList := &ast.FieldList{Opening: call.Src.Lparen, Closing: call.Src.Rparen}
	typ := ir.FuncType{
		Src: &ast.FuncType{Func: call.Src.Pos()},
		Params: &ir.FieldList{
			Src: srcFieldList,
			List: []*ir.FieldGroup{
				containerGroup,
				elementsGroup,
			}},
		Results: &ir.FieldList{
			Src: srcFieldList,
			List: []*ir.FieldGroup{
				containerGroup,
			},
		},
	}
	return importFuncType(scope, &typ)
}

func (f *appendFunc) typeNode() typeNode {
	return f
}

func (f *appendFunc) name() *ast.Ident {
	return &ast.Ident{Name: f.ext.Name()}
}

func (f *appendFunc) irFunc() ir.Func {
	return &f.ext
}
func (f *appendFunc) isGeneric() bool {
	return false
}

func (f *appendFunc) kind() ir.Kind {
	return ir.FuncKind
}

func (f *appendFunc) buildType() ir.Type {
	return &ir.FuncType{}
}

func (f *appendFunc) String() string {
	return "append"
}
