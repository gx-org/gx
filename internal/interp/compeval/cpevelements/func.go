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

type function struct {
	fn   ir.Func
	recv *fun.Receiver
}

var (
	_ ir.StorageElement = (*function)(nil)
	_ ir.FuncElement    = (*function)(nil)
)

// NewFunc creates a new function given its definition and a receiver.
func NewFunc(fn ir.Func, recv *fun.Receiver) fun.Func {
	return &function{fn: fn, recv: recv}
}

func (f *function) Func() ir.Func {
	return f.fn
}

func (f *function) Recv() *fun.Receiver {
	return f.recv
}

func (f *function) Call(env *fun.CallEnv, call *ir.CallExpr, args []ir.Element) ([]ir.Element, error) {
	valArgs := make([]ir.Element, len(args))
	for i, arg := range args {
		valArgs[i] = StoredValueOf(arg)
	}
	_, isKeyword := f.fn.(*ir.FuncKeyword)
	fType := f.fn.FuncType()
	if isKeyword || fType != nil && fType.CompEval { // Some builtin functions have no type at the moment.
		return interp.NewRunFunc(f.fn, f.recv).Call(env, call, valArgs)
	}
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

func (f *function) Store() ir.Storage {
	storage, ok := f.fn.(ir.Storage)
	if !ok {
		return nil
	}
	return storage
}

func (f *function) Type() ir.Type {
	return f.fn.Type()
}
