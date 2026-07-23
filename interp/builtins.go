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
	"github.com/gx-org/gx/build/builder/builtins"
	"github.com/gx-org/gx/build/fmterr"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/base/scope"
	"github.com/gx-org/gx/internal/interp/numbers"
	"github.com/gx-org/gx/interp/elements"
	"github.com/gx-org/gx/interp/engine"
)

// InitBuiltins initializes the builtins.
func (itp *Base) InitBuiltins(scope *scope.RWScope[ir.Element]) error {
	nilStorage := elements.NilStorage()
	scope.Define(nilStorage.NameDef().Name, nilStorage)
	if err := itp.defineBoolConstant(scope, ir.FalseStorage()); err != nil {
		return err
	}
	if err := itp.defineBoolConstant(scope, ir.TrueStorage()); err != nil {
		return err
	}
	for _, impl := range []ir.FuncImpl{
		appendF,
		axlengthsF,
		lenF,
		setF,
		traceF,
	} {
		irFunc := &ir.FuncKeyword{
			ID:   &ast.Ident{Name: impl.Name()},
			Impl: impl,
		}
		elFunc := itp.funFact.NewFunc(irFunc, nil)
		scope.Define(impl.Name(), elFunc)
	}
	for _, tp := range []ir.Type{
		ir.AnyType(),
		ir.ErrorType(),
		ir.BoolType(),
		ir.Bfloat16Type(),
		ir.Float32Type(),
		ir.Float64Type(),
		ir.IntType(),
		ir.Int32Type(),
		ir.Int64Type(),
		ir.StringType(),
		ir.Uint32Type(),
		ir.Uint64Type(),
	} {
		scope.Define(tp.ReferString(nil), tp)
	}
	for _, mc := range []builtins.BuiltinMacro{
		builtins.VarArgsIndex(),
		builtins.Unpack(),
	} {
		scope.Define(mc.Name(), &ir.MacroKeyword{
			MetaCore: ir.MetaCore{
				Src: &ast.FuncDecl{Name: &ast.Ident{Name: mc.Name()}},
			},
			BuildSynthetic: mc.Impl(),
		})
	}
	return nil
}

var builtinFile = &ir.File{
	Src: &ast.File{Name: &ast.Ident{Name: "<builtin file>"}},
	Package: &ir.Package{
		Name:  &ast.Ident{Name: "<builtin package>"},
		Decls: &ir.Declarations{},
	},
}

func (itp *Base) defineBoolConstant(scope *scope.RWScope[ir.Element], val ir.StorageWithValue) error {
	el := numbers.NewBoolFromStorage(val)
	bckEl, err := itp.Engine().ArrayOps().ElementFromAtomLit(builtinFile, el)
	if err != nil {
		return err
	}
	scope.Define(val.NameDef().Name, bckEl)
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
	lenF = &builtinFunc{
		FuncImpl: builtins.Len(),
		impl:     lenImpl,
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

func appendImpl(env engine.Env, call *ir.FuncCallExpr, recv ir.Element, args []ir.Element) ([]ir.Element, error) {
	slice, ok := elements.Underlying(args[0]).(engine.Slice)
	if !ok {
		return nil, errors.Errorf("cannot cast %T to %s", args[0], reflect.TypeFor[engine.Slice]())
	}
	withElts, err := elements.ToWithElements(args[1])
	if err != nil {
		return nil, err
	}
	return []ir.Element{slice.Append(call, withElts.Elements())}, nil
}

func axlengthsImpl(env engine.Env, call *ir.FuncCallExpr, recv ir.Element, args []ir.Element) ([]ir.Element, error) {
	file := env.ExprEval().File()
	array, ok := args[0].(elements.WithAxes)
	if !ok {
		return nil, fmterr.InternalAt(file.FileSet(), call.Src, "cannot get the shape of %T: not supported", args[0])
	}
	shape, err := array.Axes(env.ExprEval())
	if err != nil {
		return nil, fmterr.Error(file.FileSet(), call.Src, err)
	}
	return []ir.Element{shape}, nil
}

func lenImpl(env engine.Env, call *ir.FuncCallExpr, recv ir.Element, args []ir.Element) ([]ir.Element, error) {
	withLen, ok := args[0].(ir.WithLength)
	if !ok {
		return nil, errors.Errorf("cannot cast %T to %s", args[0], reflect.TypeFor[ir.WithLength]())
	}
	l, err := withLen.Length(env.ExprEval())
	if err != nil {
		return nil, err
	}
	return []ir.Element{numbers.NewIntFrom(int64(l), ir.IntType())}, err
}

func setImpl(env engine.Env, call *ir.FuncCallExpr, recv ir.Element, args []ir.Element) ([]ir.Element, error) {
	out, err := env.Engine().ArrayOps().Set(env.ExprEval(), call, args[0], args[1], args[2:])
	if err != nil {
		return nil, err
	}
	return []ir.Element{out}, nil
}

func traceImpl(env engine.Env, call *ir.FuncCallExpr, recv ir.Element, args []ir.Element) ([]ir.Element, error) {
	return nil, env.Engine().Trace(env.ExprEval(), call, args)
}
