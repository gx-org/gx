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
	"github.com/pkg/errors"
	"github.com/gomlx/compute"
	"github.com/gx-org/gx/build/fmterr"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/interp/compeval"
	"github.com/gx-org/gx/internal/tracer/processor"
	"github.com/gx-org/gx/interp/engine"
	"github.com/gx-org/gx/interp"
	"github.com/gx-org/gx/interp/materialise"
)

// New returns a new evaluator given a elements.
func New(importer ir.Importer, pr *processor.Processor, gr compute.Function) (*engine.Registry, *Ops) {
	ao := &Ops{
		graph:   gr,
		process: pr,
	}
	return &engine.Registry{
		Importer:       importer,
		CompEvalEngine: compeval.Engine(),
		HostEngine:     ao,
		Processor:      pr,
		Runners:        interp.Runners(),
		Trace:          pr.RegisterTrace,
	}, ao
}

func unpackTypes(file *ir.File, src ir.Node, tp ir.Type) (*ir.StructType, []*ir.NamedType, error) {
	switch tpT := tp.(type) {
	case *ir.NamedType:
		sType, nTypes, err := unpackTypes(file, src, tpT.Underlying.Val())
		if err != nil {
			return nil, nil, err
		}
		nTypes = append(nTypes, tpT)
		return sType, nTypes, nil
	case *ir.StructType:
		return tpT, nil, nil
	default:
		return nil, nil, fmterr.InternalAt(file.FileSet(), src.Node(), "cannot unpack a tuple to type %T: not supported", tp)
	}
}

// GraphFromElement returns a graph given an element.
func GraphFromElement(env *engine.Env, call *ir.FuncCallExpr, el ir.Element, args []ir.Element) (compute.Function, error) {
	fn, ok := el.(engine.Func)
	if !ok {
		return nil, errors.Errorf("cannot get a graph from %T", el)
	}
	subGraph, err := env.Engine().Graph().Closure()
	if err != nil {
		return nil, err
	}
	proc := processor.New()
	subReg, subOps := New(env.Registry().Importer, proc, subGraph)
	fnIR := fn.FuncIR()
	vis := newInputVisitor(subOps, fnIR.File())
	paramFields := fnIR.FuncType().Params.Fields()
	subArgs := make([]ir.Element, len(paramFields))
	for i, param := range paramFields {
		var err error
		subArgs[i], err = vis.visitArg(param, i, args[i])
		if err != nil {
			return nil, err
		}
	}
	if _, err := materialise.All(subOps, proc.ElementArgs()); err != nil {
		return nil, err
	}
	subEnv := env.Sub(subReg, subOps)
	outElts, err := fn.Call(subEnv, call, nil, subArgs)
	if err != nil {
		return nil, err
	}
	outNodes, _, err := materialise.Flatten(subOps, outElts...)
	if err != nil {
		return nil, err
	}
	if err := subGraph.Return(outNodes, nil); err != nil {
		return nil, err
	}
	return subGraph, nil
}
