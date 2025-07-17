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
	"github.com/gx-org/gx/interp"
)

type processCallResults func(ops.Node) ([]ir.Element, error)

// funcLit is a function represented as a subgraph.
type funcLit struct {
	eval *Evaluator
	lit  *ir.FuncLit
	litp *interp.FuncLitScope
}

var _ interp.Func = (*funcLit)(nil)

func (ev *Evaluator) newFuncLit(lit *ir.FuncLit, litp *interp.FuncLitScope) *funcLit {
	return &funcLit{
		eval: ev,
		lit:  lit,
		litp: litp,
	}
}

// Func returns the IR function represented by the graph.
func (sg *funcLit) Func() ir.Func {
	return sg.lit
}

// Recv returns the receiver of the function.
func (sg *funcLit) Recv() *interp.Receiver {
	return nil
}

// SubGraph represents the graph implementing the function.
func (sg *funcLit) SubGraph() (*ops.Subgraph, error) {
	subeval, err := sg.eval.subEval("literal")
	if err != nil {
		return nil, err
	}
	litp := sg.litp.FileScope().NewFuncLitScope(subeval)
	proxyArgs, err := buildProxyArguments(litp, sg.lit.FType.Params.Fields())
	if err != nil {
		return nil, err
	}
	fnInputs, err := subeval.FuncInputsToElements(litp.FileScope(), sg.lit.FType, nil, proxyArgs)
	if err != nil {
		return nil, err
	}
	outElts, err := litp.RunFuncLit(subeval, sg.lit, fnInputs.Args)
	if err != nil {
		return nil, err
	}
	_, outNode, err := subeval.outputNodesFromElements(litp.FileScope(), sg.lit.FType, outElts)
	if err != nil {
		return nil, err
	}
	return &ops.Subgraph{
		Graph:  subeval.ao.Graph(),
		Result: *outNode,
	}, nil
}

// Call the function literal given its set of arguments,
// effectively inlining the function in the parent graph.
func (sg *funcLit) Call(fitp *interp.FileScope, call *ir.CallExpr, args []ir.Element) ([]ir.Element, error) {
	return sg.litp.RunFuncLit(sg.eval, sg.lit, args)
}

// Unflatten creates a GX value from the next handles available in the parser.
func (sg *funcLit) Unflatten(handles *flatten.Parser) (values.Value, error) {
	return values.NewIRNode(sg.lit)
}

// Type of the function.
func (sg *funcLit) Type() ir.Type {
	return sg.lit.Type()
}
