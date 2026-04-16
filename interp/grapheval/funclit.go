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

package grapheval

import (
	"github.com/gx-org/backend/ops"
	"github.com/gx-org/gx/api/values"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/interp/flatten"
	"github.com/gx-org/gx/interp/context"
	"github.com/gx-org/gx/interp/fun"
	"github.com/gx-org/gx/interp"
)

type processCallResults func(ops.Node) ([]ir.Element, error)

// funcLit is a function represented as a subgraph.
type funcLit struct {
	lit  *ir.FuncLit
	litp *interp.FuncLitScope
	ctx  *context.Context
}

var _ fun.Func = (*funcLit)(nil)

// NewFuncLit creates a new function literal.
func (ev *Evaluator) NewFuncLit(lit *ir.FuncLit, ctx *context.Context) (fun.Func, error) {
	return &funcLit{
		lit:  lit,
		litp: interp.NewFuncLitScope(ctx, lit),
	}, nil
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
	return sg.litp.RunFuncLit(env, args)
}

// Unflatten creates a GX value from the next handles available in the parser.
func (sg *funcLit) Unflatten(handles *flatten.Parser) (values.Value, error) {
	return values.NewIRNode(sg.lit)
}

// Type of the function.
func (sg *funcLit) Type() ir.Type {
	return sg.lit.Type()
}
