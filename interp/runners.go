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
	"github.com/gx-org/gx/interp/engine"
	"github.com/gx-org/gx/interp/fun"
)

// Runners provide runners to the environment to execute functions.
type Runners struct{}

var _ fun.Runners = Runners{}

// FuncDecl runs a function implemented in GX.
func (Runners) FuncDecl(fn *ir.FuncDecl, env *fun.CallEnv, call *ir.FuncCallExpr, recv engine.Copier, args []ir.Element) ([]ir.Element, error) {
	if fn.Body == nil {
		return nil, fmterr.Errorf(fn.File().FileSet(), fn.Node(), "missing function body")
	}
	callerFrame := env.Context().CurrentFrame()
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
	if err := assignTypeParameters(env.Context(), call.Callee, callerFrame, funcFrame); err != nil {
		return nil, err
	}
	assignAxisLengths(call.Callee, funcFrame)
	if err := assignArgumentValues(ftype, funcFrame, args); err != nil {
		return nil, err
	}
	// Evaluate the function within the frame.
	fitp := toInterp(env.Context(), env.Engine(), env.FuncEval())
	return evalFuncBody(fitp, fn.Body)
}

// Builtin runs a function builtin in GX or provided by a backend.
func (Runners) Builtin(fn ir.Func, impl ir.FuncImpl, env *fun.CallEnv, call *ir.FuncCallExpr, recv engine.Copier, args []ir.Element) (_ []ir.Element, err error) {
	defer func() {
		if err != nil {
			err = fmterr.Error(env.File().FileSet(), call.Expr(), err)
		}
	}()
	if impl == nil {
		return nil, errors.Errorf("function %s has no implementation", fn.ShortString())
	}
	builtin, isBuiltin := impl.Implementation().(FuncBuiltin)
	if !isBuiltin {
		return nil, errors.Errorf("type %T is not a function builtin implementation", impl)
	}
	if builtin == nil {
		return nil, errors.Errorf("function %s has no implementation", fn.ShortString())
	}
	return builtin(env, call, recv, args)
}
