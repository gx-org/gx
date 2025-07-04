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
	"github.com/gx-org/gx/build/fmterr"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/base/scope"
	"github.com/gx-org/gx/interp/elements"
	"github.com/gx-org/gx/interp/evaluator"
)

var builtinFile = &ir.File{Package: &ir.Package{Name: &ast.Ident{Name: "<interp>"}}}

func (ectx *EvalContext) defineBoolConstant(val ir.StorageWithValue) error {
	gxValue, err := values.AtomBoolValue(ir.BoolType(), val.Value(nil).(*ir.AtomicValueT[bool]).Val)
	if err != nil {
		return err
	}
	el, err := ectx.evaluator.ElementFromAtom(elements.NewExprAt(builtinFile, val.Value(nil)), gxValue)
	if err != nil {
		return err
	}
	ectx.builtin.scope.Define(val.NameDef().Name, el)
	return nil
}

func (ectx *EvalContext) buildBuiltinFrame() error {
	ectx.builtin = &baseFrame{scope: scope.NewScope[elements.Element](nil)}
	if err := ectx.defineBoolConstant(ir.FalseStorage()); err != nil {
		return err
	}
	if err := ectx.defineBoolConstant(ir.TrueStorage()); err != nil {
		return err
	}
	for name, impl := range map[string]ir.FuncImpl{
		"append":    appendFunc{},
		"axlengths": axlengthsFunc{},
		"set":       setFunc{},
		"trace":     traceFunc{},
	} {
		irFunc := &ir.FuncBuiltin{
			Src:  &ast.FuncDecl{Name: &ast.Ident{Name: name}},
			Impl: impl,
		}
		var err error
		irFunc.FType, err = impl.BuildFuncType(nil, nil)
		if err != nil {
			return err
		}
		elFunc := ectx.evaluator.NewFunc(irFunc, nil)
		ectx.builtin.scope.Define(name, elFunc)
	}
	return nil
}

type appendFunc struct{}

var _ ir.FuncImpl = (*appendFunc)(nil)

func (appendFunc) BuildFuncType(fetcher ir.Fetcher, call *ir.CallExpr) (*ir.FuncType, error) {
	return &ir.FuncType{CompEval: true}, nil
}

func (appendFunc) Implementation() any {
	return FuncBuiltin(appendImpl)
}

func appendImpl(ctx evaluator.Context, call elements.CallAt, fn elements.Func, irFunc *ir.FuncBuiltin, args []elements.Element) ([]elements.Element, error) {
	slice, ok := args[0].(*elements.Slice)
	if !ok {
		return nil, errors.Errorf("cannot cast %T to %s", args[0], reflect.TypeFor[*elements.Slice]())
	}
	els := append([]elements.Element{}, slice.Elements()...)
	els = append(els, args[1:]...)
	return []elements.Element{elements.NewSlice(slice.Type(), els)}, nil
}

type axlengthsFunc struct{}

var _ ir.FuncImpl = (*axlengthsFunc)(nil)

func (axlengthsFunc) BuildFuncType(fetcher ir.Fetcher, call *ir.CallExpr) (*ir.FuncType, error) {
	return &ir.FuncType{CompEval: true}, nil
}

func (axlengthsFunc) Implementation() any {
	return FuncBuiltin(axlengthsImpl)
}

func axlengthsImpl(ctx evaluator.Context, call elements.CallAt, fn elements.Func, irFunc *ir.FuncBuiltin, args []elements.Element) ([]elements.Element, error) {
	array, ok := args[0].(elements.WithAxes)
	if !ok {
		return nil, fmterr.Internalf(ctx.File().FileSet(), call.Node().Src, "cannot get the shape of %T: not supported", args[0])
	}
	shape, err := array.Axes(ctx)
	if err != nil {
		return nil, fmterr.Position(ctx.File().FileSet(), call.Node().Src, err)
	}
	return []elements.Element{shape}, nil
}

type setFunc struct{}

var _ ir.FuncImpl = (*setFunc)(nil)

func (setFunc) BuildFuncType(fetcher ir.Fetcher, call *ir.CallExpr) (*ir.FuncType, error) {
	return nil, nil
}

func (setFunc) Implementation() any {
	return FuncBuiltin(setImpl)
}

func setImpl(ctx evaluator.Context, call elements.CallAt, fn elements.Func, irFunc *ir.FuncBuiltin, args []elements.Element) ([]elements.Element, error) {
	out, err := ctx.Evaluation().Evaluator().ArrayOps().Set(call, args[0], args[1], args[2])
	if err != nil {
		return nil, err
	}
	return []elements.Element{out}, nil
}

type traceFunc struct{}

var _ ir.FuncImpl = (*traceFunc)(nil)

func (traceFunc) BuildFuncType(fetcher ir.Fetcher, call *ir.CallExpr) (*ir.FuncType, error) {
	return nil, nil
}

func (traceFunc) Implementation() any {
	return FuncBuiltin(traceImpl)
}

func traceImpl(ctx evaluator.Context, call elements.CallAt, fn elements.Func, irFunc *ir.FuncBuiltin, args []elements.Element) ([]elements.Element, error) {
	ctxT := ctx.(*context)
	return nil, ctxT.eval.evaluator.Trace(call, fn, irFunc, args, &ctxT.CallInputs().Values)
}
