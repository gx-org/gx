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
	"github.com/gx-org/gx/build/fmterr"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/interp/compeval"
	"github.com/gx-org/gx/internal/interp/compeval/cpevelements"
	"github.com/gx-org/gx/internal/tracer/processor"
	"github.com/gx-org/gx/interp/elements"
	"github.com/gx-org/gx/interp/evaluator"
	"github.com/gx-org/gx/interp"
	"github.com/gx-org/gx/interp/materialise"
)

// SubGrapher is an element (e.g. a function) that can be represented by a sub-graph.
type SubGrapher interface {
	SubGraph(name string) (*ops.Subgraph, error)
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
func (ev *Evaluator) NewFunc(itp *interp.Interpreter, fn ir.Func, recv *interp.Receiver) interp.Func {
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
func (ev *Evaluator) Materialiser() materialise.Materialiser {
	return ev.ao
}

// ElementFromAtom returns an element from a GX value.
func (ev *Evaluator) ElementFromAtom(ctx ir.Evaluator, src ir.AssignableExpr, val values.Array) (evaluator.NumericalElement, error) {
	return ev.hostEval.ElementFromAtom(ctx, src, val)
}

func buildProxyArguments(file *ir.File, args []*ir.Field) ([]ir.Element, error) {
	els := make([]ir.Element, len(args))
	for i, arg := range args {
		var err error
		els[i], err = cpevelements.NewRuntimeValue(file, interp.NewRunFunc, &ir.FieldStorage{
			Field: arg,
		})
		if err != nil {
			return nil, err
		}
	}
	return els, nil
}

func (ev *Evaluator) outputNodesFromElements(fileScope *interp.FileScope, fType *ir.FuncType, out []ir.Element) (processCallResults, *ops.OutputNode, error) {
	if len(out) == 0 {
		return nil, nil, errors.Errorf("literal has no output")
	}
	nodes, shapes, err := materialise.Flatten(ev.ao, out...)
	if err != nil {
		return nil, nil, err
	}
	results := fType.Results
	if len(out) != results.Len() {
		return nil, nil, errors.Errorf("got %d out elements but want %d from function type", len(out), results.Len())
	}
	if len(nodes) == 1 {
		out := &ops.OutputNode{Node: nodes[0], Shape: shapes[0]}
		return func(outputNode ops.Node) ([]ir.Element, error) {
			expr := &ir.ValueRef{
				Stor: &ir.FieldStorage{Field: results.Fields()[0]},
			}
			return ev.ao.ElementsFromNodes(fileScope.File(), expr, out)
		}, out, nil
	}
	exprs := make([]ir.AssignableExpr, len(nodes))
	for i := range nodes {
		exprs[i] = &ir.ValueRef{
			Stor: &ir.FieldStorage{Field: results.Fields()[0]},
		}
	}
	tupleNode, err := ev.ao.Graph().Core().Tuple(nodes)
	if err != nil {
		return nil, nil, err
	}
	return func(outputNode ops.Node) ([]ir.Element, error) {
		return ev.elementsFromTupleNode(fileScope.File(), tupleNode, exprs, shapes)
	}, &ops.OutputNode{Node: tupleNode}, nil
}

// NewFuncLit creates a new function literal.
func (ev *Evaluator) NewFuncLit(fitp *interp.FileScope, lit *ir.FuncLit) (interp.Func, error) {
	litScope := fitp.NewFuncLitScope(ev, lit)
	return ev.newFuncLit(lit, litScope), nil
}

func (ev *Evaluator) elementsFromTupleNode(file *ir.File, tpl ops.Tuple, elExprs []ir.AssignableExpr, shps []*shape.Shape) ([]ir.Element, error) {
	elts := make([]ir.Element, tpl.Size())
	for i := range tpl.Size() {
		node, err := tpl.Element(i)
		if err != nil {
			return nil, err
		}
		elts[i], err = NewBackendNode(ev, elements.NewExprAt(file, elExprs[i]), &ops.OutputNode{
			Node:  node,
			Shape: shps[i],
		})
		if err != nil {
			return nil, err
		}
	}
	return elts, nil
}

func unpackTypes(file *ir.File, src ir.SourceNode, tp ir.Type) (*ir.StructType, []*ir.NamedType, error) {
	switch tpT := tp.(type) {
	case *ir.NamedType:
		sType, nTypes, err := unpackTypes(file, src, tpT.Underlying.Typ)
		if err != nil {
			return nil, nil, err
		}
		nTypes = append(nTypes, tpT)
		return sType, nTypes, nil
	case *ir.StructType:
		return tpT, nil, nil
	default:
		return nil, nil, fmterr.Internalf(file.FileSet(), src.Source(), "cannot unpack a tuple to type %T: not supported", tp)
	}
}

// ElementFromTuple creates an interpreter element of a given type from a graph tuple.
func (ev *Evaluator) ElementFromTuple(file *ir.File, expr ir.AssignableExpr, tpl ops.Tuple, shapes []*shape.Shape, targetType ir.Type) (ir.Element, error) {
	structTyp, namedTypes, err := unpackTypes(file, expr, targetType)
	if err != nil {
		return nil, err
	}
	// Construct dummy expressions for all the fields of the structure to keep track of the value types.
	fieldExprs := make([]ir.AssignableExpr, structTyp.NumFields())
	for i, field := range structTyp.Fields.Fields() {
		fieldExprs[i] = &ir.ValueRef{
			Src:  field.Name,
			Stor: field.Storage(),
		}
	}
	els, err := ev.elementsFromTupleNode(file, tpl, fieldExprs, shapes)
	if err != nil {
		return nil, err
	}
	var el interp.Copier
	el = interp.NewStructFromElements(structTyp, els)
	for _, nType := range namedTypes {
		el = interp.NewNamedType(interp.NewRunFunc, nType, el)
	}
	return el, nil
}

func (ev *Evaluator) subEval(proc *processor.Processor, name string) (*Evaluator, error) {
	args := proc.Args()
	shapes := make([]*shape.Shape, len(args))
	for i, arg := range args {
		shapes[i] = arg.Shape()
	}
	subGraph, err := ev.ao.SubGraph(name, shapes)
	if err != nil {
		return nil, err
	}
	return New(ev.Importer(), proc, subGraph.Graph()), nil
}

// Trace a set of elements.
func (ev *Evaluator) Trace(ctx ir.Evaluator, call *ir.CallExpr, args []ir.Element) error {
	return ev.process.RegisterTrace(ctx, call, args)
}

func opsFromContext(ctx *interp.FileScope) *arrayOps {
	return ctx.Evaluator().(*Evaluator).ao
}

// FuncInputsToElements converts values to a function input.
func FuncInputsToElements(file *ir.File, processor *processor.Processor, fType *ir.FuncType, receiver ir.Element, args []ir.Element) (*elements.InputElements, error) {
	vis := newInputVisitor(file, processor)
	var recvEl ir.Element
	if receiver != nil {
		recvField := fType.ReceiverField()
		var err error
		if recvEl, err = vis.visitReceiver(recvField, receiver); err != nil {
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
		argNode, err := vis.visitArg(param, i, args[i])
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
func GraphFromElement(name string, el ir.Element) (*ops.Subgraph, error) {
	grapher, ok := el.(SubGrapher)
	if !ok {
		return nil, errors.Errorf("cannot get a graph from %T", el)
	}
	return grapher.SubGraph(name)
}

func axesFromShape(ev ir.Evaluator, shape *shape.Shape) (*interp.Slice, error) {
	ctx := ev.(evaluator.Context)
	axes := make([]ir.Element, len(shape.AxisLengths))
	for i, axisSize := range shape.AxisLengths {
		iExpr := &ir.AtomicValueT[ir.Int]{
			Val: ir.Int(i),
			Typ: ir.IntLenType(),
		}
		iValue, err := values.AtomIntegerValue[ir.Int](ir.IntLenType(), ir.Int(axisSize))
		if err != nil {
			return nil, err
		}
		axes[i], err = ctx.Evaluator().ElementFromAtom(ctx, iExpr, iValue)
		if err != nil {
			return nil, err
		}
	}
	return interp.NewSlice(ir.IntLenSliceType(), axes), nil
}
