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
	"go/ast"
	"reflect"

	"github.com/pkg/errors"
	"github.com/gx-org/gx/api/values"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/base/scope"
	"github.com/gx-org/gx/interp/elements"
	"github.com/gx-org/gx/interp/state"
)

var (
	builtinFile = &ir.File{Package: &ir.Package{Name: &ast.Ident{Name: "<interp>"}}}
	boolType    = ir.TypeFromKind(ir.BoolKind)
	trueExpr    = elements.NewExprAt[ir.Expr](builtinFile, &ir.ValueRef{
		Src: &ast.Ident{Name: "true"},
		Typ: boolType,
	})
	falseExpr = elements.NewExprAt[ir.Expr](builtinFile, &ir.ValueRef{
		Src: &ast.Ident{Name: "false"},
		Typ: boolType,
	})
)

func (ctx *context) buildBuiltinFrame() error {
	ctx.builtin = scope.NewScope[frameState, state.Element](frameState{Context: ctx}, nil)
	trueElement, err := ctx.Evaluator().ElementFromValue(ctx, trueExpr.Node(), values.AtomBoolValue(boolType, true))
	if err != nil {
		return err
	}
	ctx.builtin.Define("true", trueElement)
	falseElement, err := ctx.Evaluator().ElementFromValue(ctx, falseExpr.Node(), values.AtomBoolValue(boolType, false))
	if err != nil {
		return err
	}
	ctx.builtin.Define("false", falseElement)
	assignBuiltinFunc(ctx, ctx.builtin, "append", appendFunc{})
	assignBuiltinFunc(ctx, ctx.builtin, "axlengths", axlengthsFunc{})
	assignBuiltinFunc(ctx, ctx.builtin, "set", setFunc{})
	assignBuiltinFunc(ctx, ctx.builtin, "trace", traceFunc{})
	return nil
}

func assignBuiltinFunc(ctx Context, f *frame, name string, impl ir.FuncImpl) {
	f.Define(name, ctx.State().Func(&ir.FuncBuiltin{
		FName: name,
		Impl:  impl,
	}, nil))
}

type appendFunc struct{}

var _ ir.FuncImpl = (*appendFunc)(nil)

func (appendFunc) BuildFuncType(fetcher ir.Fetcher, call *ir.CallExpr) (*ir.FuncType, error) {
	return nil, errors.Errorf("not implemented")
}

func (appendFunc) Implementation() any {
	return FuncBuiltin(appendImpl)
}

func appendImpl(ctx Context, call elements.CallAt, fn *state.Func, irFunc *ir.FuncBuiltin, args []state.Element) (output state.Element, err error) {
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

func axlengthsImpl(ctx Context, call elements.CallAt, fn *state.Func, irFunc *ir.FuncBuiltin, args []state.Element) (output state.Element, err error) {
	shape, err := state.ShapeFromElement(args[0])
	if err != nil {
		return nil, err
	}
	axes := make([]state.Element, len(shape.AxisLengths))
	for i, axisSize := range shape.AxisLengths {
		iExpr := &ir.AtomicValueT[ir.Int]{
			Src: call.Node().Src,
			Val: ir.Int(i),
			Typ: ir.IntLenType(),
		}
		iValue := values.AtomIntegerValue[ir.Int](ir.IntLenType(), ir.Int(axisSize))
		axes[i], err = ctx.Evaluator().ElementFromValue(ctx, iExpr, iValue)
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

func setImpl(ctx Context, call elements.CallAt, fn *state.Func, irFunc *ir.FuncBuiltin, args []state.Element) (output state.Element, err error) {
	return ctx.Evaluator().Set(ctx, call.Node(), args[0], args[1], args[2])
}

type traceFunc struct{}

var _ ir.FuncImpl = (*traceFunc)(nil)

func (traceFunc) BuildFuncType(fetcher ir.Fetcher, call *ir.CallExpr) (*ir.FuncType, error) {
	return nil, errors.Errorf("not implemented")
}

func (traceFunc) Implementation() any {
	return FuncBuiltin(traceImpl)
}

func traceImpl(ctx Context, call elements.CallAt, fn *state.Func, irFunc *ir.FuncBuiltin, args []state.Element) (output state.Element, err error) {
	return nil, ctx.State().Trace(call, fn, irFunc, args, ctx)
}
