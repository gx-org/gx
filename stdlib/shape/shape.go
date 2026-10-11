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

// Package shape provides functions to manipulate the shape of tensors.
package shape

import (
	"fmt"
	"slices"

	"github.com/pkg/errors"
	"github.com/gx-org/gx/api/hostio"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/interp/coreiface"
	"github.com/gx-org/gx/internal/undef"
	"github.com/gx-org/gx/interp/elements"
	"github.com/gx-org/gx/interp/engine"
	"github.com/gx-org/gx/interp/materialise"
	"github.com/gx-org/gx/stdlib/builtin"
)

// Package description of the GX shape package.
var Package = builtin.PackageBuilder{
	FullPath: "shape",
	Builders: []builtin.Builder{
		builtin.ParseSource(),
		builtin.ImplementBuiltin("Gather", evalGather),
		builtin.ImplementBuiltin("Split", evalSplit),
		builtin.ImplementBuiltin("Broadcast", evalBroadcast),
		builtin.ImplementBuiltin("Transpose", evalTranspose),
		builtin.ImplementBuiltin("Concat", evalConcat),
		builtin.ImplementBuiltin("ConcatAxis", concatAxis),
		builtin.ImplementBuiltin("GatherAxis", gatherAxis),
		builtin.ImplementBuiltin("SameSlice", sameSlice),
		builtin.ImplementBuiltin("ReduceAxes", reduceAxes),
		builtin.ImplementBuiltin("ReverseAxes", reverseAxes),
		builtin.ImplementBuiltin("SplitAxis", splitAxis),
	},
}

func evalTranspose(env *engine.Env, call *ir.FuncCallExpr, recv ir.Element, genArgs, args []ir.Element) ([]ir.Element, error) {
	mat := builtin.Materialiser(env)
	argNode, argShape, err := materialise.Element(mat, args[0])
	if err != nil {
		return nil, err
	}
	if len(argShape.Dimensions) <= 1 {
		return []ir.Element{args[0]}, nil
	}
	wantAxes := make([]int, len(argShape.Dimensions))
	for i := range wantAxes {
		wantAxes[i] = len(wantAxes) - i - 1
	}
	op, err := env.Engine().Graph().Transpose(argNode, wantAxes...)
	if err != nil {
		return nil, err
	}
	return materialise.ElementFromNode(env.ExprEval(), mat, op, call.Type())
}

func outOfBoundAxis(env *engine.Env, call *ir.FuncCallExpr, axes *elements.Slice, idx int, name string) error {
	shapeS := ir.ExprString(env.ExprEval(), call.Expr(), axes)
	return ir.CompileErrorF("%s axis %d out of bound for array with %d axes (with axis lengths: %s)", name, idx, axes.Len(), shapeS)
}

func reverseAxes(env *engine.Env, call *ir.FuncCallExpr, recv ir.Element, genArgs, args []ir.Element) (_ []ir.Element, err error) {
	axes, err := elements.SliceFromElement(args[0])
	if err != nil {
		return nil, err
	}
	rev := append([]ir.Element{}, axes.Elements()...)
	slices.Reverse(rev)
	revAxes, err := elements.NewSlice(axes.Type(), rev)
	if err != nil {
		return nil, err
	}
	return []ir.Element{revAxes}, nil
}

func reduceAxes(env *engine.Env, call *ir.FuncCallExpr, recv ir.Element, genArgs, args []ir.Element) (_ []ir.Element, err error) {
	under, err := coreiface.Underlying(args[0])
	if err != nil {
		return nil, err
	}
	axes, ok := under.(*elements.Slice)
	if !ok {
		return nil, undef.Err()
	}
	indices, err := elements.Map(elements.IntFromElement, args[1])
	if undef.Is(err) {
		return nil, err
	}
	if err != nil {
		return nil, fmt.Errorf("cannot get axis index from argument 1: %w", err)
	}
	keep := make([]bool, axes.Len())
	for i := range keep {
		keep[i] = true
	}
	for _, axisIndex := range indices {
		if axisIndex >= len(keep) {
			err := outOfBoundAxis(env, call, axes, axisIndex, "reduce")
			return []ir.Element{args[0]}, err
		}
		// Reduced over the axis: do not keep.
		keep[axisIndex] = false
	}
	var eltsKeep []ir.Element
	for i, ax := range axes.Elements() {
		if !keep[i] {
			continue
		}
		eltsKeep = append(eltsKeep, ax)
	}
	out, err := elements.NewSlice(axes.Type(), eltsKeep)
	return []ir.Element{out}, err
}

func sameSlice(env *engine.Env, call *ir.FuncCallExpr, recv ir.Element, genArgs, args []ir.Element) (_ []ir.Element, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("cannot call fmt.SameSlice: %w", err)
		}
	}()
	if len(args) != 2 {
		return nil, errors.Errorf("got %d arguments but want 2", len(args))
	}
	x, err := elements.AxesFromElement(args[0])
	if err != nil {
		return nil, fmt.Errorf("cannot get axis from argument 1: %w", err)
	}
	y, err := elements.AxesFromElement(args[1])
	if err != nil {
		return nil, fmt.Errorf("cannot get axis from argument 2: %w", err)
	}
	ok := false
	if len(x) != len(y) {
		goto ret
	}
	for i, xi := range x {
		if xi != y[i] {
			goto ret
		}
	}
	ok = true
ret:
	boolValue, err := hostio.AtomBoolValue(ir.BoolType(), ok)
	if err != nil {
		return nil, errors.Errorf("cannot create bool value in fmt.SameSlice")
	}
	return []ir.Element{boolValue}, nil
}
