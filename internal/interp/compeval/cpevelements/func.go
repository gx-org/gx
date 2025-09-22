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

	"github.com/pkg/errors"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/interp"
)

type function struct {
	fn   ir.Func
	recv *interp.Receiver
}

// NewFunc creates a new function given its definition and a receiver.
func NewFunc(fn ir.Func, recv *interp.Receiver) interp.Func {
	return &function{fn: fn, recv: recv}
}

func (f *function) Func() ir.Func {
	return f.fn
}

func (f *function) Recv() *interp.Receiver {
	return f.recv
}

func (f *function) callAtCompEval(fitp *interp.FileScope, call *ir.CallExpr, args []ir.Element) ([]ir.Element, error) {
	pkgFunc, ok := f.fn.(ir.PkgFunc)
	if !ok {
		return nil, errors.Errorf("cannot evaluate a non compeval function (e.g. a function literal) at compile time")
	}
	return fitp.EvalFunc(pkgFunc, call, args)
}

func (f *function) Call(fitp *interp.FileScope, call *ir.CallExpr, args []ir.Element) ([]ir.Element, error) {
	fType := f.fn.FuncType()
	if fType != nil && fType.CompEval { // Some builtin functions have no type at the moment.
		return f.callAtCompEval(fitp, call, args)
	}
	res := call.Callee.T.Results.Fields()
	els := make([]ir.Element, len(res))
	for i, ri := range res {
		var err error
		els[i], err = NewRuntimeValue(fitp.File(), fitp.NewFunc, &ir.LocalVarStorage{
			Src: &ast.Ident{},
			Typ: ri.Type(),
		})
		if err != nil {
			return nil, err
		}
	}
	return els, nil
}

func (f *function) Type() ir.Type {
	return f.fn.Type()
}
