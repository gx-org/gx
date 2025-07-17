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
	"github.com/gx-org/gx/internal/interp/compeval/cpevelements"
	"github.com/gx-org/gx/internal/tracer/processor"
	"github.com/gx-org/gx/interp/elements"
	"github.com/gx-org/gx/interp/evaluator"
	"github.com/gx-org/gx/interp"
)

// SubGrapher is an element (e.g. a function) that can be represented by a sub-graph.
type SubGrapher interface {
	SubGraph() (*ops.Subgraph, error)
}

// Evaluator evaluates GX operations by adding the corresponding
// node in the backend graph.
type Evaluator struct {
	process *processor.Processor
	ao      *arrayOps

	hostEval evaluator.Evaluator
}

var _ interp.Evaluator = (*Evaluator)(nil)

// New returns a new evaluator given a elements.
func New(importer ir.Importer, pr *processor.Processor, gr ops.Graph) *Evaluator {
	ev := &Evaluator{
		process:  pr,
		hostEval: compeval.NewHostEvaluator(importer),
	}
	ev.ao = &arrayOps{graph: gr, ev: ev}
	return ev
}

// NewFunc creates a new function given its definition and a receiver.
func (ev *Evaluator) NewFunc(itp *interp.Interpreter, fn ir.PkgFunc, recv *interp.Receiver) interp.Func {
	return interp.NewRunFunc(fn, recv)
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
func (ev *Evaluator) ArrayOps() evaluator.ArrayOps {
	return ev.ao
}

// Materialiser returns an array materialiser.
func (ev *Evaluator) Materialiser() elements.ArrayMaterialiser {
	return ev.ao
}

// ElementFromAtom returns an element from a GX value.
func (ev *Evaluator) ElementFromAtom(ctx ir.Evaluator, src ir.AssignableExpr, val values.Array) (evaluator.NumericalElement, error) {
	return ev.hostEval.ElementFromAtom(ctx, src, val)
}

func buildProxyArguments(litp *interp.FuncLitScope, args []*ir.Field) ([]ir.Element, error) {
	els := make([]ir.Element, len(args))
	file := litp.FileScope().File()
	newFunc := litp.FileScope().NewFunc
	for i, arg := range args {
		var err error
		els[i], err = cpevelements.NewRuntimeValue(file, newFunc, &ir.FieldStorage{
			Field: arg,
		})
		if err != nil {
			return nil, err
		}
	}
	return els, nil
}

func (ev *Evaluator) outputNodesFromElements(fileScope *interp.FileScope, fType *ir.FuncType, out []ir.Element) (processCallResults, *ops.OutputNode, error) {
	outNodes, err := MaterialiseAll(fileScope, out)
	if err != nil {
		return nil, nil, err
	}
	if len(out) == 0 {
		return nil, nil, errors.Errorf("literal has no output")
	}
	results := fType.Results
	if len(out) != results.Len() {
		return nil, nil, errors.Errorf("got %d out elements but want %d from function type", len(out), results.Len())
	}
	if results.Len() == 1 {
		return func(outputNode ops.Node) ([]ir.Element, error) {
			expr := elements.NewExprAt(fileScope.File(), &ir.ValueRef{
				Stor: &ir.FieldStorage{Field: results.Fields()[0]},
			})
			return ElementsFromNode(expr, outNodes[0])
		}, outNodes[0], nil
	}
	nodes := make([]ops.Node, len(outNodes))
	shapes := make([]*shape.Shape, len(outNodes))
	exprs := make([]ir.AssignableExpr, len(outNodes))
	for i, outNode := range outNodes {
		nodes[i] = outNode.Node
		shapes[i] = outNode.Shape
		exprs[i] = &ir.ValueRef{
			Stor: &ir.FieldStorage{Field: results.Fields()[0]},
		}
	}
	tupleNode, err := ev.ao.Graph().Core().Tuple(nodes)
	if err != nil {
		return nil, nil, err
	}
	return func(outputNode ops.Node) ([]ir.Element, error) {
		return ElementsFromTupleNode(fileScope.File(), tupleNode, exprs, shapes)
	}, &ops.OutputNode{Node: tupleNode}, nil
}

// NewFuncLit creates a new function literal.
func (ev *Evaluator) NewFuncLit(fitp *interp.FileScope, lit *ir.FuncLit) (interp.Func, error) {
	return ev.newFuncLit(lit, fitp.NewFuncLitScope(ev)), nil
}

// ElementsFromTupleNode converts the graph nodes of a tuple node into elements.
func ElementsFromTupleNode(file *ir.File, tpl ops.Tuple, elExprs []ir.AssignableExpr, shps []*shape.Shape) ([]ir.Element, error) {
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

func (ev *Evaluator) subEval(name string) (*Evaluator, error) {
	subGraph, err := ev.ao.SubGraph(name)
	if err != nil {
		return nil, err
	}
	return New(ev.Importer(), ev.process, subGraph.Graph()), nil
}

// Trace a set of elements.
func (ev *Evaluator) Trace(ctx ir.Evaluator, call *ir.CallExpr, args []ir.Element) error {
	return ev.process.RegisterTrace(ctx, call, args)
}

func opsFromContext(ctx *interp.FileScope) *arrayOps {
	return ctx.Evaluator().(*Evaluator).ao
}

// FuncInputsToElements converts values to a function input.
func (ev *Evaluator) FuncInputsToElements(fitp *interp.FileScope, fType *ir.FuncType, receiver ir.Element, args []ir.Element) (*elements.InputElements, error) {
	var recvEl ir.Element
	if receiver != nil {
		recvField := fType.ReceiverField()
		var err error
		if recvEl, err = ev.Receiver(fitp, recvField, receiver); err != nil {
			return nil, err
		}
	}

	paramFields := fType.Params.Fields()
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
		argNode, err := ev.ArgGX(fitp, param, i, args[i])
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

// GraphFromElement returns a graph given an element.
func GraphFromElement(el ir.Element) (*ops.Subgraph, error) {
	grapher, ok := el.(SubGrapher)
	if !ok {
		return nil, errors.Errorf("cannot get a graph from %T", el)
	}
	return grapher.SubGraph()
}
