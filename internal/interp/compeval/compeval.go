// Copyright 2025 Google LLC
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

// Package compeval runs GX code at compile time.
package compeval

import (
	"reflect"

	"github.com/pkg/errors"
	"github.com/gx-org/gx/api/options"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/interp/compeval/cpevops"
	"github.com/gx-org/gx/internal/interp/compeval/surrogates/storepath"
	"github.com/gx-org/gx/internal/interp/compeval/surrogates"
	"github.com/gx-org/gx/interp/elements"
	"github.com/gx-org/gx/interp/fun"
	"github.com/gx-org/gx/interp"
)

// EvalExpr evaluates a GX expression into an interpreter element.
func EvalExpr(eval ir.Evaluator, expr ir.Expr) (cpevops.Element, error) {
	val, err := eval.EvalExpr(expr)
	if err != nil {
		return nil, err
	}
	el, ok := val.(cpevops.Element)
	if !ok {
		return nil, errors.Errorf("cannot cast %T to %s", val, reflect.TypeFor[cpevops.Element]().String())
	}
	return el, nil
}

// NewOptionVariable creates a package option to set a static variable of a package with its corresponding symbolic element.
func NewOptionVariable(vr *ir.VarExpr) (options.PackageOption, error) {
	val, err := surrogates.New(storepath.NewVar(vr), vr.Type())
	return elements.PackageVarSetElement{
		Pkg:   vr.Decl.FFile.Package.Path(),
		Var:   vr.VName.Name,
		Value: val,
	}, err
}

type mixFunction struct {
	*surrogates.Function
}

func (f *mixFunction) run(env *fun.CallEnv, call *ir.FuncCallExpr, args []ir.Element) ([]ir.Element, error) {
	valArgs := make([]ir.Element, len(args))
	for i, arg := range args {
		valArgs[i] = ir.BareValue(arg)
	}
	fn := interp.NewRunFunc(f.IR(), f.Recv())
	return fn.Call(env.WithRunners(interp.Runners()), call, valArgs)
}

func (f *mixFunction) Call(env *fun.CallEnv, call *ir.FuncCallExpr, args []ir.Element) ([]ir.Element, error) {
	fn := f.IR()
	_, isKeyword := fn.(*ir.FuncKeyword)
	if isKeyword {
		return f.run(env, call, args)
	}
	fType := fn.FuncType()
	if fType != nil && fType.CompEval {
		return f.run(env, call, args)
	}
	return f.Function.Call(env, call, args)
}

// RunFunc creates functions such that compeval functions are evaluated
// while non-compeval functions are simulated.
func RunFunc(fn ir.Func, recv *fun.Receiver) fun.Func {
	return &mixFunction{
		Function: surrogates.NewSurFunc(fn, recv),
	}
}
