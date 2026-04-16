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

package interp

import (
	"github.com/gx-org/gx/api/values"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/interp/flatten"
	"github.com/gx-org/gx/interp/context"
	"github.com/gx-org/gx/interp/fun"
)

// funcLit is a function represented as a subgraph.
type funcLit struct {
	ctx *context.Context
	lit *ir.FuncLit
}

var _ fun.Func = (*funcLit)(nil)

// NewFuncLit creates a new function literal.
func NewFuncLit(lit *ir.FuncLit, ctx *context.Context) fun.Func {
	return &funcLit{
		ctx: ctx.FuncLitFrame(lit),
		lit: lit,
	}
}

// Func returns the IR function represented by the graph.
func (sg *funcLit) Func() ir.Func {
	return sg.lit
}

// Recv returns the receiver of the function.
func (sg *funcLit) Recv() *fun.Receiver {
	return nil
}

// Call the function literal given its set of arguments,
// effectively inlining the function in the parent graph.
func (sg *funcLit) Call(env *fun.CallEnv, call *ir.FuncCallExpr, args []ir.Element) ([]ir.Element, error) {
	funcFrame := sg.ctx.PushBlockFrame()
	fType := sg.lit.FuncType()
	if err := assignArgumentValues(fType, funcFrame, args); err != nil {
		return nil, err
	}
	for _, resultName := range fieldNames(fType.Results.List) {
		funcFrame.Define(resultName, nil)
	}
	defer sg.ctx.PopFrame()
	fitp := toInterp(sg.ctx, env.Engine(), env.FuncEval())
	return evalFuncBody(fitp, sg.lit.Body)
}

// Unflatten creates a GX value from the next handles available in the parser.
func (sg *funcLit) Unflatten(handles *flatten.Parser) (values.Value, error) {
	return values.NewIRNode(sg.lit)
}

// Type of the function.
func (sg *funcLit) Type() ir.Type {
	return sg.lit.Type()
}
