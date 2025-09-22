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

package control

import (
	"go/ast"

	"github.com/gx-org/backend/ops"
	"github.com/gx-org/gx/build/builtins"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/interp/elements"
	"github.com/gx-org/gx/interp/evaluator"
	"github.com/gx-org/gx/interp/grapheval"
	"github.com/gx-org/gx/interp"
	"github.com/gx-org/gx/interp/materialise"
	"github.com/gx-org/gx/stdlib/builtin"
	"github.com/gx-org/gx/stdlib/impl"
)

type while struct {
	builtin.Func
}

func (f while) BuildFuncIR(impl *impl.Stdlib, pkg *ir.Package) (*ir.FuncBuiltin, error) {
	return builtin.IRFuncBuiltin[while]("While", evalWhile, pkg), nil
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

func evalWhile(ctx evaluator.Context, call elements.CallAt, fn interp.Func, irFunc *ir.FuncBuiltin, args []ir.Element) ([]ir.Element, error) {
	g := ctx.Evaluator().ArrayOps().Graph().Core()

	cond, err := grapheval.GraphFromElement("while.cond", args[1])
	if err != nil {
		return nil, err
	}
	body, err := grapheval.GraphFromElement("while.body", args[2])
	if err != nil {
		return nil, err
	}

	stateNodes, stateShapes, err := materialise.Flatten(ctx.Materialiser(), args[0])
	if err != nil {
		return nil, err
	}
	stateTpl, err := g.Tuple(stateNodes)
	if err != nil {
		return nil, err
	}

	fnState, err := g.While(cond, body, stateTpl)
	if err != nil {
		return nil, err
	}
	ev := ctx.Evaluator().(*grapheval.Evaluator)
	out, err := ev.ElementFromTuple(ctx.File(), call.Node(), fnState.(ops.Tuple), stateShapes, args[0].Type())
	if err != nil {
		return nil, err
	}
	return []ir.Element{out}, nil
}
