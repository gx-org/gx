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

type traceFunc struct {
	ext ir.FuncBuiltin
}

var (
	_ genericCallTypeNode = (*traceFunc)(nil)
	_ function            = (*traceFunc)(nil)
)

func (f *traceFunc) resolveGenericCallType(scope scoper, src ast.Node, fetcher ir.Fetcher, call *ir.CallExpr) (*funcType, bool) {
	params := make([]ir.Type, len(call.Args))
	for i, arg := range call.Args {
		params[i] = arg.Type()
	}
	return importFuncType(scope, &ir.FuncType{
		Src:     &ast.FuncType{Func: call.Src.Pos()},
		Params:  builtins.Fields(params...),
		Results: builtins.Fields(),
	})
}

func (f *traceFunc) typeNode() typeNode {
	return f
}

func (f *traceFunc) name() *ast.Ident {
	return &ast.Ident{Name: f.ext.Name()}
}

func (f *traceFunc) kind() ir.Kind {
	return ir.FuncKind
}

func (f *traceFunc) isGeneric() bool {
	return false
}

func (f *traceFunc) buildType() ir.Type {
	return &ir.FuncType{}
}

func (f *traceFunc) String() string {
	return "trace"
}

func (f *traceFunc) irFunc() ir.Func {
	return &f.ext
}
