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
	"github.com/gx-org/gx/build/builder/builtins"
	"github.com/gx-org/gx/build/fmterr"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/base/scope"
	"github.com/gx-org/gx/interp/context"
	"github.com/gx-org/gx/interp/elements"
	"github.com/gx-org/gx/interp/evaluator"
	"github.com/gx-org/gx/interp/fun"
)

// InitBuiltins initializes the builtins.
func (itp *Interpreter) InitBuiltins(ctx *context.Context, scope *scope.RWScope[ir.Element]) error {
	if err := itp.defineBoolConstant(scope, ctx, ir.FalseStorage()); err != nil {
		return err
	}
	if err := itp.defineBoolConstant(scope, ctx, ir.TrueStorage()); err != nil {
		return err
	}
	for _, impl := range []ir.FuncImpl{
		appendF,
		axlengthsF,
		setF,
		traceF,
	} {
		irFunc := &ir.FuncKeyword{
			ID:   &ast.Ident{Name: impl.Name()},
			Impl: impl,
		}
		elFunc := itp.eval.NewFunc(irFunc, nil)
		scope.Define(impl.Name(), elFunc)
	}
	for _, tp := range []ir.Type{
		ir.AnyType(),
		ir.BoolType(),
		ir.Bfloat16Type(),
		ir.Float32Type(),
		ir.Float64Type(),
		ir.Int32Type(),
		ir.Int64Type(),
		ir.StringType(),
		ir.Uint32Type(),
		ir.Uint64Type(),
		ir.IntLenType(),
		ir.IntIndexType(),
	} {
		scope.Define(tp.String(), tp)
	}
	return nil
}

func (itp *Interpreter) defineBoolConstant(scope *scope.RWScope[ir.Element], ctx *context.Context, val ir.StorageWithValue) error {
	gxValue, err := values.AtomBoolValue(ir.BoolType(), val.Value(nil).(*ir.AtomicValueT[bool]).Val)
	if err != nil {
		return err
	}
	var el ir.Element
	el, err = itp.eval.ElementFromAtom(ctx.File(), val.Value(nil), gxValue)
	if err != nil {
		return err
	}
	el = itp.eval.ElementFromStorage(ctx.File(), val, el)
	scope.Define(val.NameDef().Name, el)
	return nil
}

type builtinFunc struct {
	ir.FuncImpl
	impl FuncBuiltin
}

func (bf builtinFunc) Implementation() any {
	return bf.impl
}

var (
	appendF = &builtinFunc{
		FuncImpl: builtins.Append(),
		impl:     appendImpl,
	}
	axlengthsF = &builtinFunc{
		FuncImpl: builtins.AxLengths(),
		impl:     axlengthsImpl,
	}
	setF = &builtinFunc{
		FuncImpl: builtins.Set(),
		impl:     setImpl,
	}
	traceF = &builtinFunc{
		FuncImpl: builtins.Trace(),
		impl:     traceImpl,
	}
)

func appendImpl(env evaluator.Env, call elements.CallAt, fn fun.Func, irFunc *ir.FuncBuiltin, args []ir.Element) ([]ir.Element, error) {
	slice, ok := args[0].(*elements.Slice)
	if !ok {
		return nil, errors.Errorf("cannot cast %T to %s", args[0], reflect.TypeFor[*elements.Slice]())
	}
	els := append([]ir.Element{}, slice.Elements()...)
	els = append(els, args[1:]...)
	return []ir.Element{elements.NewSlice(slice.Type(), els)}, nil
}

func axlengthsImpl(env evaluator.Env, call elements.CallAt, fn fun.Func, irFunc *ir.FuncBuiltin, args []ir.Element) ([]ir.Element, error) {
	file := env.ExprEval().File()
	array, ok := args[0].(elements.WithAxes)
	if !ok {
		return nil, fmterr.Internalf(file.FileSet(), call.Node().Src, "cannot get the shape of %T: not supported", args[0])
	}
	shape, err := array.Axes(env.ExprEval())
	if err != nil {
		return nil, fmterr.AtNode(file.FileSet(), call.Node().Src, err)
	}
	return []ir.Element{shape}, nil
}

func setImpl(env evaluator.Env, call elements.CallAt, fn fun.Func, irFunc *ir.FuncBuiltin, args []ir.Element) ([]ir.Element, error) {
	out, err := env.Evaluator().ArrayOps().Set(env.ExprEval(), call.Node(), args[0], args[1], args[2])
	if err != nil {
		return nil, err
	}
	return []ir.Element{out}, nil
}

func traceImpl(env evaluator.Env, call elements.CallAt, fn fun.Func, irFunc *ir.FuncBuiltin, args []ir.Element) ([]ir.Element, error) {
	return nil, env.Evaluator().Trace(env.ExprEval(), call.Node(), args)
}
