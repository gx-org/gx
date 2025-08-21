// Copyright 2025 Google LLC
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

// Package control provides runtime control flow via the GX `control` standard library.
package control

import (
	"go/ast"

	"github.com/gx-org/gx/build/builtins"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/stdlib/builtin"
	"github.com/gx-org/gx/stdlib/impl"
)

// Package description of the GX control package.
var Package = builtin.PackageBuilder{
	FullPath: "control",
	Builders: []builtin.Builder{builtin.BuildFunc(while{})},
}

type while struct {
	builtin.Func
}

func (f while) BuildFuncIR(impl *impl.Stdlib, pkg *ir.Package) (*ir.FuncBuiltin, error) {
	return builtin.IRFuncBuiltin[while]("While", impl.Control.While, pkg), nil
}

// Described in Go syntax, control.While has the signature:
//
//	func While[T any](state T, cond func(state T) bool, body func(state T) T) T
//
// The `cond` and `body` functions must accept the iteration state through identical parameters, and
// the body function must return a new iteration state. Last, the overall While() function takes an
// initial iteration state and returns the final state.
func (f while) BuildFuncType(fetcher ir.Fetcher, call *ir.CallExpr) (*ir.FuncType, error) {
	stateType := call.Args[0].Type()
	condType := &ir.FuncType{ // `func(state T) bool`
		BaseType: ir.BaseType[*ast.FuncType]{Src: &ast.FuncType{Func: call.Source().Pos()}},
		Params:   builtins.Fields(call, stateType),
		Results:  builtins.Fields(call, ir.BoolType()),
	}
	bodyType := &ir.FuncType{ // `func(state T) T`
		BaseType: ir.BaseType[*ast.FuncType]{Src: &ast.FuncType{Func: call.Source().Pos()}},
		Params:   builtins.Fields(call, stateType),
		Results:  builtins.Fields(call, stateType),
	}
	return &ir.FuncType{
		BaseType: ir.BaseType[*ast.FuncType]{Src: &ast.FuncType{Func: call.Source().Pos()}},
		Params:   builtins.Fields(call, stateType, condType, bodyType),
		Results:  builtins.Fields(call, stateType),
	}, nil
}
