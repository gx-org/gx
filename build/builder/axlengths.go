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

	"github.com/gx-org/gx/build/ir"
)

type axlengthsFunc struct {
	ext ir.FuncBuiltin
}

var (
	_ genericCallTypeNode = (*axlengthsFunc)(nil)
	_ staticValueNode     = (*axlengthsFunc)(nil)
	_ function            = (*axlengthsFunc)(nil)
)

func (f *axlengthsFunc) resolveGenericCallType(scope scoper, fetcher ir.Fetcher, call *callExpr) (*funcType, bool) {
	src := call.source()
	irCall := call.buildExpr().(*ir.CallExpr)
	if len(irCall.Args) != 1 {
		scope.err().Appendf(src, "wrong number of arguments to axlengths: expected 1, found %d", len(irCall.Args))
		return nil, false
	}
	arg := irCall.Args[0]
	if arg.Type().Kind() != ir.ArrayKind {
		scope.err().Appendf(src, "axlengths(%s) not supported", arg.Type().Kind())
		return nil, false
	}
	argType, ok := toTypeNode(scope, arg.Type())
	if !ok {
		return nil, false
	}
	argArrayType, ok := argType.(*arrayType)
	if !ok {
		scope.err().Appendf(src, "axlengths(%T) not supported", argType)
		return nil, false
	}
	axes, ok := argArrayType.lengths(scope, irCall)
	if !ok {
		return nil, false
	}
	axesType, ok := axes.resolveType(scope)
	if !ok {
		return nil, false
	}

	srcFieldList := &ast.FieldList{Opening: irCall.Src.Lparen, Closing: irCall.Src.Rparen}

	paramsGroup := &ir.FieldGroup{Type: argType.irType()}
	paramsGroup.Fields = []*ir.Field{{Group: paramsGroup}}

	resultsGroup := &ir.FieldGroup{Type: axesType.irType()}
	resultsGroup.Fields = []*ir.Field{{Group: resultsGroup}}

	typ := ir.FuncType{
		Src: &ast.FuncType{Func: irCall.Src.Pos()},
		Params: &ir.FieldList{
			Src:  srcFieldList,
			List: []*ir.FieldGroup{paramsGroup},
		},
		Results: &ir.FieldList{
			Src:  srcFieldList,
			List: []*ir.FieldGroup{resultsGroup},
		},
	}
	return importFuncType(scope, &typ)
}

func (f *axlengthsFunc) resolveType(scoper) (typeNode, bool) {
	return f, true
}

func (f *axlengthsFunc) receiver() *fieldList {
	return nil
}

func (f *axlengthsFunc) name() *ast.Ident {
	return &ast.Ident{Name: f.ext.Name()}
}

func (f *axlengthsFunc) kind() ir.Kind {
	return ir.FuncKind
}

func (f *axlengthsFunc) isGeneric() bool {
	return false
}

func (f *axlengthsFunc) staticValue() ir.StaticValue {
	return &f.ext
}

func (f *axlengthsFunc) irType() ir.Type {
	return &ir.FuncType{}
}

func (f *axlengthsFunc) String() string {
	return "axlengths"
}

func (f *axlengthsFunc) irFunc() ir.Func {
	return &f.ext
}
