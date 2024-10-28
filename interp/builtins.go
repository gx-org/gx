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

package interp

import (
	"reflect"

	"github.com/pkg/errors"
	"github.com/gx-org/backend/dtype"
	"github.com/gx-org/backend/shape"
	"github.com/gx-org/gx/api/values"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/golang/backend/kernels"
	"github.com/gx-org/gx/interp/state"
)

var (
	trueValue = values.NewHostArray(
		ir.ToAtomic(ir.BoolKind),
		kernels.ToBuffer(
			[]bool{true},
			&shape.Shape{DType: dtype.Bool},
		),
	)
	falseValue = values.NewHostArray(
		ir.ToAtomic(ir.BoolKind),
		kernels.ToBuffer(
			[]bool{false},
			&shape.Shape{DType: dtype.Bool},
		),
	)
)

func (ctx *context) newBuiltinFrame(fn *ir.FuncDecl) *frame {
	builtins := map[string]state.Element{
		"true":  ctx.state.Numerical(trueValue),
		"false": ctx.state.Numerical(falseValue),
	}
	ctx.appendBuiltinFunc(builtins, "append", appendFunc{})
	ctx.appendBuiltinFunc(builtins, "axlengths", axlengthsFunc{})
	ctx.appendBuiltinFunc(builtins, "set", setFunc{})
	ctx.appendBuiltinFunc(builtins, "trace", traceFunc{})
	return &frame{
		ctx:      ctx,
		vars:     builtins,
		function: fn,
	}
}

func (ctx *context) appendBuiltinFunc(vals map[string]state.Element, name string, impl ir.FuncImpl) {
	vals[name] = ctx.state.Func(&ir.FuncBuiltin{
		FName: name,
		Impl:  impl,
	}, nil)
}

type appendFunc struct{}

var _ ir.FuncImpl = (*appendFunc)(nil)

func (appendFunc) BuildFuncType(fetcher ir.Fetcher, call *ir.CallExpr) (*ir.FuncType, error) {
	return nil, errors.Errorf("not implemented")
}

func (appendFunc) Implementation() any {
	return FuncBuiltin(appendImpl)
}

func appendImpl(ctx Context, call state.CallAt, fn *state.Func, irFunc *ir.FuncBuiltin, args []state.Element) (output state.Element, err error) {
	slice, ok := args[0].(*state.Slice)
	if !ok {
		return nil, errors.Errorf("cannot cast %T to %s", args[0], reflect.TypeFor[*state.Slice]())
	}
	elements := append([]state.Element{}, slice.Elements()...)
	elements = append(elements, args[1:]...)
	return ctx.State().ToSlice(call.ToExprAt(), elements), nil
}

type axlengthsFunc struct{}

var _ ir.FuncImpl = (*axlengthsFunc)(nil)

func (axlengthsFunc) BuildFuncType(fetcher ir.Fetcher, call *ir.CallExpr) (*ir.FuncType, error) {
	return nil, errors.Errorf("not implemented")
}

func (axlengthsFunc) Implementation() any {
	return FuncBuiltin(axlengthsImpl)
}

func axlengthsImpl(ctx Context, call state.CallAt, fn *state.Func, irFunc *ir.FuncBuiltin, args []state.Element) (output state.Element, err error) {
	shape, err := state.ShapeFromElement(args[0])
	if err != nil {
		return nil, err
	}
	axes := make([]state.Element, len(shape.AxisLengths))
	for i, axisSize := range shape.AxisLengths {
		axes[i], err = state.NewAtomicLiteral[ir.Int](
			ctx.State(),
			ir.AxisLengthType(),
			ir.Int(axisSize),
		)
		if err != nil {
			return nil, err
		}
	}
	return ctx.State().ToSlice(call.ToExprAt(), axes), nil
}

type setFunc struct{}

var _ ir.FuncImpl = (*setFunc)(nil)

func (setFunc) BuildFuncType(fetcher ir.Fetcher, call *ir.CallExpr) (*ir.FuncType, error) {
	return nil, errors.Errorf("not implemented")
}

func (setFunc) Implementation() any {
	return FuncBuiltin(setImpl)
}

func setImpl(ctx Context, call state.CallAt, fn *state.Func, irFunc *ir.FuncBuiltin, args []state.Element) (output state.Element, err error) {
	nodes, err := state.OutputsFromElements(args)
	if err != nil {
		return nil, err
	}
	setNode, err := ctx.State().BackendGraph().Core().NewSet(nodes[0].Node, nodes[1].Node, nodes[2].Node)
	if err != nil {
		return nil, err
	}
	return ctx.State().ElementFromNode(call.ToExprAt(), setNode, nodes[0].Shape)
}

type traceFunc struct{}

var _ ir.FuncImpl = (*traceFunc)(nil)

func (traceFunc) BuildFuncType(fetcher ir.Fetcher, call *ir.CallExpr) (*ir.FuncType, error) {
	return nil, errors.Errorf("not implemented")
}

func (traceFunc) Implementation() any {
	return FuncBuiltin(traceImpl)
}

func traceImpl(ctx Context, call state.CallAt, fn *state.Func, irFunc *ir.FuncBuiltin, args []state.Element) (output state.Element, err error) {
	return nil, ctx.State().Trace(call, fn, irFunc, args, ctx)
}
