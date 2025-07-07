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
	"github.com/gx-org/gx/build/ir"
)

type traceFunc struct {
}

var _ ir.FuncImpl = (*traceFunc)(nil)

// BuildFuncType builds the type of a function given how it is called.
func (f *traceFunc) BuildFuncType(fetcher ir.Fetcher, call *ir.CallExpr) (*ir.FuncType, error) {
	params := make([]ir.Type, len(call.Args))
	for i, arg := range call.Args {
		params[i] = arg.Type()
	}
	return &ir.FuncType{
		BaseType: ir.BaseType[*ast.FuncType]{
			Src: &ast.FuncType{Func: call.Src.Pos()},
		},
		Params:  builtins.Fields(call, params...),
		Results: builtins.Fields(call),
	}, nil
}

// BuildFuncType builds the type of a function given how it is called.
func (f *traceFunc) Implementation() any {
	return nil
}
