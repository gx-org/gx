// Copyright 2026 Google LLC
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
	"github.com/pkg/errors"
	"github.com/gx-org/gx/build/fmterr"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/concrete"
	"github.com/gx-org/gx/internal/undef"
	"github.com/gx-org/gx/interp/elements"
	"github.com/gx-org/gx/interp/engine"
)

// runners provide runners to the environment to execute functions.
type runners struct{}

var run = runners{}

// Runners returns a set of runners to run all functions.
func Runners() engine.Runners {
	return run
}

func (runners) FuncDecl(fn *ir.FuncDecl, env *engine.Env, call *ir.FuncCallExpr, recv engine.Copier, genArgs, args []ir.Element) ([]ir.Element, error) {
	if fn.Body == nil {
		return nil, fmterr.Errorf(fn.File().FileSet(), fn.Node(), "missing function body")
	}
	// Create a new function frame.
	funcFrame, err := env.Context().PushFuncFrame(fn)
	if err != nil {
		return nil, err
	}
	defer env.Context().PopFrame()
	ftype := fn.FuncType()
	results := ftype.Results
	if results != nil {
		for _, resultName := range fieldNames(results.List) {
			funcFrame.Define(resultName, nil)
		}
	}
	// Add the receiver name to the function frame if present.
	if recv != nil {
		recvField := fn.FuncType().ReceiverField()
		if recvField != nil && ir.ValidIdent(recvField.Name) {
			recv = recv.Copy()
			funcFrame.Define(recvField.Name, recv)
		}
	}
	assignTypeParameters(env.Context(), call.Callee, funcFrame, genArgs)
	if err := assignArgumentValues(ftype, funcFrame, args); err != nil {
		return nil, err
	}
	// Evaluate the function within the frame.
	fitp := newInterp(env.Registry(), env.Context(), env.Engine())
	return evalFuncBody(fitp, fn.Body)
}

// Builtin runs a function provided by a backend.
func (r runners) Builtin(fn *ir.FuncBuiltin, env *engine.Env, call *ir.FuncCallExpr, recv engine.Copier, genArgs, args []ir.Element) ([]ir.Element, error) {
	return r.call(fn, fn.Impl, env, call, recv, genArgs, args)
}

// Keyword runs a keyword in GX.
func (r runners) Keyword(fn *ir.FuncKeyword, env *engine.Env, call *ir.FuncCallExpr, recv engine.Copier, genArgs, args []ir.Element) ([]ir.Element, error) {
	return r.call(fn, fn.Impl, env, call, recv, genArgs, args)
}

func (runners) call(fn ir.Func, impl ir.FuncImpl, env *engine.Env, call *ir.FuncCallExpr, recv engine.Copier, genArgs, args []ir.Element) (_ []ir.Element, err error) {
	defer func() {
		if err == nil || undef.Is(err) {
			return
		}
		if _, isCompErr := err.(*ir.CompileError); isCompErr {
			return
		}
		err = fmterr.Error(env.File().FileSet(), call.Expr(), err)
	}()
	if impl == nil {
		return nil, errors.Errorf("function %s has no implementation", fn.ReferString(env.File()))
	}
	builtin, isBuiltin := impl.Implementation().(elements.FuncBuiltin)
	if !isBuiltin {
		return nil, errors.Errorf("type %T is not a function builtin implementation", impl)
	}
	if builtin == nil {
		return nil, errors.Errorf("function %s has no implementation", fn.ReferString(env.File()))
	}
	cCall, err := concreteCall(env, call)
	if err != nil {
		return nil, err
	}
	return builtin(env, cCall, recv, genArgs, args)
}

// concreteCall returns a copy of a function call where the type arguments of
// the callee are concrete, that is the generic type parameters they reference
// are replaced by their values in the current frame.
func concreteCall(env *engine.Env, call *ir.FuncCallExpr) (*ir.FuncCallExpr, error) {
	callee, isFuncVal := call.Callee.(*ir.FuncValExpr)
	if !isFuncVal {
		return call, nil
	}
	ftype := callee.FuncType()
	if len(ftype.GenParams.Values) == 0 {
		return call, nil
	}
	ev := env.For(compEvalNature).ExprEval()
	values := make([]ir.GenericValue, len(ftype.GenParams.Values))
	for i, value := range ftype.GenParams.Values {
		typeValue, isTypeValue := value.(*ir.TypeGenericValue)
		if !isTypeValue {
			values[i] = value
			continue
		}
		cType, err := concrete.Concrete(ev, typeValue.DefinedType())
		if err != nil {
			return nil, err
		}
		values[i] = ir.NewTypeExprGenericValue(typeValue.Generic().(*ir.GenericTypeParam), ir.TypeExpr(typeValue.Value().Expr(), cType))
	}
	cFType := *ftype
	cFType.GenParams.Values = values
	cCall := *call
	cCall.Callee = callee.NewFType(&cFType)
	return &cCall, nil
}
