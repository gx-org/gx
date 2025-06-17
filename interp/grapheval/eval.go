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

// Package grapheval implements the evaluation of core GX functions.
package grapheval

import (
	"github.com/gx-org/backend/graph"
	"github.com/gx-org/backend/shape"
	"github.com/gx-org/gx/api/values"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/interp/compeval"
	"github.com/gx-org/gx/internal/tracer/processor"
	"github.com/gx-org/gx/interp/elements"
	"github.com/gx-org/gx/interp/evaluator"
)

// Evaluator evaluates GX operations by adding the corresponding
// node in the backend graph.
type Evaluator struct {
	process *processor.Processor
	ao      *arrayOps
	newFunc elements.NewFunc

	hostEval evaluator.Evaluator
}

var _ evaluator.Evaluator = (*Evaluator)(nil)

// New returns a new evaluator given a elements.
func New(importer evaluator.Importer, pr *processor.Processor, gr graph.Graph, newFunc elements.NewFunc) *Evaluator {
	ev := &Evaluator{
		process:  pr,
		hostEval: compeval.NewHostEvaluator(importer),
		newFunc:  newFunc,
	}
	ev.ao = &arrayOps{graph: gr, ev: ev}
	return ev
}

// NewFunc creates a new function given its definition and a receiver.
func (ev *Evaluator) NewFunc(fn ir.Func, recv *elements.Receiver) elements.Func {
	return ev.newFunc(fn, recv)
}

// NewSub returns a new evaluator given a new array operator implementations.
func (ev *Evaluator) NewSub(ao elements.ArrayOps) evaluator.Evaluator {
	sub := &Evaluator{process: ev.process, ao: ao.(*arrayOps)}
	sub.ao.ev = sub
	return sub
}

// Processor returns the processor where init and debug traces are registered.
func (ev *Evaluator) Processor() *processor.Processor {
	return ev.process
}

// Importer returns the importer used by the evaluator.
func (ev *Evaluator) Importer() evaluator.Importer {
	return ev.hostEval.Importer()
}

// ArrayOps returns the array operators implementation.
func (ev *Evaluator) ArrayOps() elements.ArrayOps {
	return ev.ao
}

// ElementFromAtom returns an element from a GX value.
func (ev *Evaluator) ElementFromAtom(src elements.ExprAt, val values.Array) (elements.NumericalElement, error) {
	return ev.hostEval.ElementFromAtom(src, val)
}

// CallFuncLit calls a function literal.
func (ev *Evaluator) CallFuncLit(ctx evaluator.Context, ref *ir.FuncLit, args []elements.Element) ([]elements.Element, error) {
	core := ev.ao.graph.Core()
	name := ref.Name()
	if name == "" {
		name = "<lambda>"
	}
	subgraph, err := core.Subgraph(name)
	if err != nil {
		return nil, err
	}
	subeval := New(ev.Importer(), ev.process, subgraph, ev.newFunc)
	outs, err := ctx.EvalFunctionToElement(subeval, ref, args)
	if err != nil {
		return nil, err
	}
	outputExprs, outputGraphNodes, err := extractGraphNodes(outs)
	if err != nil {
		return nil, err
	}
	if len(outputGraphNodes) == 0 {
		// The sub-function does not need the graph.
		// Returns its output element.
		return outs, nil
	}
	subresultNodes, shapes := unpackOutputs(outputGraphNodes)
	// If the function has multiple return values, we alter the graph so that it returns a tuple of
	// values, and we also transparently unpack the tuple below.
	graphSingleOutput := outputGraphNodes[0]
	if len(outputGraphNodes) > 1 {
		tpl, err := subgraph.Core().Tuple(subresultNodes)
		if err != nil {
			return nil, err
		}
		graphSingleOutput = &graph.OutputNode{Node: tpl}
	}
	result, err := core.Call(graph.Subgraph{Graph: subgraph, Result: *graphSingleOutput})
	if err != nil {
		return nil, err
	}

	if resultTpl, ok := result.(graph.Tuple); ok {
		return ElementsFromTupleNode(
			ev.ao.graph,
			ctx.File(),
			ref,
			resultTpl,
			outputExprs,
			shapes)
	}
	el, err := ElementFromNode(
		elements.NewExprAt(ctx.File(), ref),
		&graph.OutputNode{
			Node:  result,
			Shape: shapes[0],
		})
	if err != nil {
		return nil, err
	}
	return []elements.Element{el}, nil
}

// ElementsFromTupleNode converts the graph nodes of a tuple node into elements.
func ElementsFromTupleNode(g graph.Graph, file *ir.File, expr ir.Expr, tpl graph.Tuple, elExprs []ir.AssignableExpr, shps []*shape.Shape) ([]elements.Element, error) {
	elts := make([]elements.Element, tpl.Size())
	for i := range tpl.Size() {
		node, err := tpl.Element(i)
		if err != nil {
			return nil, err
		}
		elts[i], err = ElementFromNode(elements.NewExprAt(file, elExprs[i]), &graph.OutputNode{
			Node:  node,
			Shape: shps[i],
		})
		if err != nil {
			return nil, err
		}
	}
	return elts, nil
}

// Trace a set of elements.
func (ev *Evaluator) Trace(call elements.CallAt, fn elements.Func, irFunc *ir.FuncBuiltin, args []elements.Element, fc *elements.InputValues) error {
	return ev.process.RegisterTrace(call, fn, irFunc, args, fc)
}

func evalFromContext(ctx elements.FileContext) *Evaluator {
	return ctx.(evaluator.Context).Evaluation().Evaluator().(*Evaluator)
}
