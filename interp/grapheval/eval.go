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
	"strings"

	"github.com/pkg/errors"
	"github.com/gx-org/backend/ops"
	"github.com/gx-org/backend/shape"
	"github.com/gx-org/gx/api/values"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/interp/compeval"
	"github.com/gx-org/gx/internal/tracer/processor"
	"github.com/gx-org/gx/interp/context"
	"github.com/gx-org/gx/interp/elements"
	"github.com/gx-org/gx/interp/evaluator"
	"github.com/gx-org/gx/interp/proxies"
)

// Evaluator evaluates GX operations by adding the corresponding
// node in the backend graph.
type Evaluator struct {
	process *processor.Processor
	ao      *arrayOps
	newFunc elements.NewFunc

	hostEval evaluator.Evaluator
}

var _ context.Evaluator = (*Evaluator)(nil)

// New returns a new evaluator given a elements.
func New(importer ir.Importer, pr *processor.Processor, gr ops.Graph, newFunc elements.NewFunc) *Evaluator {
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

// Processor returns the processor where init and debug traces are registered.
func (ev *Evaluator) Processor() *processor.Processor {
	return ev.process
}

// Importer returns the importer used by the evaluator.
func (ev *Evaluator) Importer() ir.Importer {
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
func (ev *Evaluator) CallFuncLit(ctx *context.Context, ref *ir.FuncLit, args []ir.Element) ([]ir.Element, error) {
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
		graphSingleOutput = &ops.OutputNode{Node: tpl}
	}
	result, err := core.Call(ops.Subgraph{Graph: subgraph, Result: *graphSingleOutput})
	if err != nil {
		return nil, err
	}

	if resultTpl, ok := result.(ops.Tuple); ok {
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
		&ops.OutputNode{
			Node:  result,
			Shape: shapes[0],
		})
	if err != nil {
		return nil, err
	}
	return []ir.Element{el}, nil
}

// ElementsFromTupleNode converts the graph nodes of a tuple node into elements.
func ElementsFromTupleNode(g ops.Graph, file *ir.File, expr ir.Expr, tpl ops.Tuple, elExprs []ir.AssignableExpr, shps []*shape.Shape) ([]ir.Element, error) {
	elts := make([]ir.Element, tpl.Size())
	for i := range tpl.Size() {
		node, err := tpl.Element(i)
		if err != nil {
			return nil, err
		}
		elts[i], err = ElementFromNode(elements.NewExprAt(file, elExprs[i]), &ops.OutputNode{
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
func (ev *Evaluator) Trace(call elements.CallAt, args []ir.Element) error {
	return ev.process.RegisterTrace(call, args)
}

func evalFromContext(ctx ir.Evaluator) *Evaluator {
	return ctx.(evaluator.Context).Evaluator().(*Evaluator)
}

// FuncInputsToElements converts values to a function input.
func (ev *Evaluator) FuncInputsToElements(file *ir.File, fType *ir.FuncType, receiver values.Value, args []values.Value) (*elements.InputElements, error) {
	var recvEl ir.Element
	if receiver != nil {
		recvField := fType.ReceiverField()
		receiverProxy, err := proxies.ToProxy(receiver, recvField.Type())
		if err != nil {
			return nil, err
		}
		recvAt := elements.NewNodeAt(file, recvField)
		if recvEl, err = ev.Receiver(recvAt, receiverProxy); err != nil {
			return nil, err
		}
	}

	paramFields := fType.Params.Fields()
	proxyArgs, err := proxies.ToProxies(args, paramFields)
	if err != nil {
		return nil, err
	}
	argsEl := make([]ir.Element, len(args))
	for i, param := range paramFields {
		if i >= len(args) {
			missingParams := paramFields[len(args):]
			builder := strings.Builder{}
			for n, param := range missingParams {
				if n > 0 {
					builder.WriteString(", ")
				}
				builder.WriteString(param.Name.String())
			}
			return nil, errors.Errorf("missing parameter(s): %s", builder.String())
		}
		paramAt := elements.NewNodeAt(file, param)
		argNode, err := ev.ArgGX(paramAt, i, proxyArgs)
		if err != nil {
			return nil, err
		}
		argsEl[i] = argNode
	}
	return &elements.InputElements{
		Values: values.FuncInputs{
			Receiver: receiver,
			Args:     args,
		},
		Receiver: recvEl,
		Args:     argsEl,
	}, nil
}
