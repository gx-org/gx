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

package cpevelements

import (
	"go/ast"

	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/interp/fun"
	"github.com/gx-org/gx/interp"
)

type proxyFunction struct {
	fn   ir.Func
	recv *fun.Receiver
}

var (
	_ ir.StorageElement = (*proxyFunction)(nil)
	_ ir.FuncElement    = (*proxyFunction)(nil)
)

// NewProxyFunc creates a new proxy function given its definition and a receiver.
// A proxy function does not evaluate its body and all returned values are proxy values.
func NewProxyFunc(fn ir.Func, recv *fun.Receiver) fun.Func {
	return &proxyFunction{fn: fn, recv: recv}
}

func (f *proxyFunction) Func() ir.Func {
	return f.fn
}

func (f *proxyFunction) Recv() *fun.Receiver {
	return f.recv
}

func (f *proxyFunction) Call(env *fun.CallEnv, call *ir.FuncCallExpr, args []ir.Element) ([]ir.Element, error) {
	res := call.Callee.FuncType().Results.Fields()
	els := make([]ir.Element, len(res))
	for i, ri := range res {
		var err error
		els[i], err = NewRuntimeValue(env.File(), &ir.LocalVarStorage{
			Src: &ast.Ident{},
			Typ: ri.Type(),
		})
		if err != nil {
			return nil, err
		}
	}
	return els, nil
}

func (f *proxyFunction) Store() ir.Storage {
	storage, ok := f.fn.(ir.Storage)
	if !ok {
		return nil
	}
	return storage
}

func (f *proxyFunction) Type() ir.Type {
	return f.fn.Type()
}

type mixFunction struct {
	proxyFunction
}

// NewMixFunc creates a new mix function given its definition and a receiver.
// A mix function evaluates other functions as proxy functions except for compeval functions and keywords which are executed.
func NewMixFunc(fn ir.Func, recv *fun.Receiver) fun.Func {
	return &mixFunction{
		proxyFunction: proxyFunction{fn: fn, recv: recv},
	}
}

func (f *mixFunction) run(env *fun.CallEnv, call *ir.FuncCallExpr, args []ir.Element) ([]ir.Element, error) {
	valArgs := make([]ir.Element, len(args))
	for i, arg := range args {
		valArgs[i] = ir.BareValue(arg)
	}
	fn := interp.NewRunFunc(f.fn, f.recv)
	return fn.Call(env.WithRunners(interp.Runners()), call, valArgs)
}

func (f *mixFunction) Call(env *fun.CallEnv, call *ir.FuncCallExpr, args []ir.Element) ([]ir.Element, error) {
	_, isKeyword := f.fn.(*ir.FuncKeyword)
	if isKeyword {
		return f.run(env, call, args)
	}
	fType := f.fn.FuncType()
	if fType != nil && fType.CompEval {
		return f.run(env, call, args)
	}
	return f.proxyFunction.Call(env, call, args)
}
