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

package context

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

// FuncBuiltin defines a builtin function provided by a backend.
type FuncBuiltin func(ctx evaluator.Context, call elements.CallAt, fn elements.Func, irFunc *ir.FuncBuiltin, args []ir.Element) ([]ir.Element, error)

func (ctx *Context) defineBoolConstant(val ir.StorageWithValue) error {
	gxValue, err := values.AtomBoolValue(ir.BoolType(), val.Value(nil).(*ir.AtomicValueT[bool]).Val)
	if err != nil {
		return err
	}
	el, err := ctx.evaluator.ElementFromAtom(elements.NewExprAt(builtinFile, val.Value(nil)), gxValue)
	if err != nil {
		return err
	}
	ctx.builtin.scope.Define(val.NameDef().Name, el)
	return nil
}

func (ctx *Context) buildBuiltinFrame() error {
	ctx.builtin = &baseFrame{scope: scope.NewScope[ir.Element](nil)}
	if err := ctx.defineBoolConstant(ir.FalseStorage()); err != nil {
		return err
	}
	if err := ctx.defineBoolConstant(ir.TrueStorage()); err != nil {
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
		elFunc := ctx.evaluator.NewFunc(irFunc, nil)
		ctx.builtin.scope.Define(name, elFunc)
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

func appendImpl(ctx evaluator.Context, call elements.CallAt, fn elements.Func, irFunc *ir.FuncBuiltin, args []ir.Element) ([]ir.Element, error) {
	slice, ok := args[0].(*elements.Slice)
	if !ok {
		return nil, errors.Errorf("cannot cast %T to %s", args[0], reflect.TypeFor[*elements.Slice]())
	}
	els := append([]ir.Element{}, slice.Elements()...)
	els = append(els, args[1:]...)
	return []ir.Element{elements.NewSlice(slice.Type(), els)}, nil
}

type axlengthsFunc struct{}

var _ ir.FuncImpl = (*axlengthsFunc)(nil)

func (axlengthsFunc) BuildFuncType(fetcher ir.Fetcher, call *ir.CallExpr) (*ir.FuncType, error) {
	return &ir.FuncType{CompEval: true}, nil
}

func (axlengthsFunc) Implementation() any {
	return FuncBuiltin(axlengthsImpl)
}

func axlengthsImpl(ctx evaluator.Context, call elements.CallAt, fn elements.Func, irFunc *ir.FuncBuiltin, args []ir.Element) ([]ir.Element, error) {
	array, ok := args[0].(elements.WithAxes)
	if !ok {
		return nil, fmterr.Internalf(ctx.File().FileSet(), call.Node().Src, "cannot get the shape of %T: not supported", args[0])
	}
	shape, err := array.Axes(ctx)
	if err != nil {
		return nil, fmterr.Position(ctx.File().FileSet(), call.Node().Src, err)
	}
	return []ir.Element{shape}, nil
}

type setFunc struct{}

var _ ir.FuncImpl = (*setFunc)(nil)

func (setFunc) BuildFuncType(fetcher ir.Fetcher, call *ir.CallExpr) (*ir.FuncType, error) {
	return nil, nil
}

func (setFunc) Implementation() any {
	return FuncBuiltin(setImpl)
}

func setImpl(ctx evaluator.Context, call elements.CallAt, fn elements.Func, irFunc *ir.FuncBuiltin, args []ir.Element) ([]ir.Element, error) {
	out, err := ctx.Evaluator().ArrayOps().Set(call, args[0], args[1], args[2])
	if err != nil {
		return nil, err
	}
	return []ir.Element{out}, nil
}

type traceFunc struct{}

var _ ir.FuncImpl = (*traceFunc)(nil)

func (traceFunc) BuildFuncType(fetcher ir.Fetcher, call *ir.CallExpr) (*ir.FuncType, error) {
	return nil, nil
}

func (traceFunc) Implementation() any {
	return FuncBuiltin(traceImpl)
}

func traceImpl(ctx evaluator.Context, call elements.CallAt, fn elements.Func, irFunc *ir.FuncBuiltin, args []ir.Element) ([]ir.Element, error) {
	ctxT := ctx.(*Context)
	return nil, ctxT.evaluator.Trace(call, fn, irFunc, args, &ctxT.CallInputs().Values)
}
