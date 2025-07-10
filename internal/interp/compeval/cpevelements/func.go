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
	"reflect"

	"github.com/pkg/errors"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/interp/context"
	"github.com/gx-org/gx/interp/elements"
)

type fun struct {
	fn   ir.Func
	recv *elements.Receiver
}

// NewFunc creates a new function given its definition and a receiver.
func NewFunc(fn ir.Func, recv *elements.Receiver) elements.Func {
	return &fun{fn: fn, recv: recv}
}

func (f *fun) Func() ir.Func {
	return f.fn
}

func (f *fun) Recv() *elements.Receiver {
	return f.recv
}

func (f *fun) callAtCompEval(fctx ir.Evaluator, call *ir.CallExpr, args []ir.Element) ([]ir.Element, error) {
	fnContext, ok := fctx.(elements.FuncEvaluator)
	if !ok {
		return nil, errors.Errorf("cannot evaluate function %s: context %T does not implement %s", f.fn.Name(), fctx, reflect.TypeFor[elements.FuncEvaluator]().String())
	}
	return fnContext.EvalFunc(f.fn, call, args)
}

func (f *fun) Call(fctx ir.Evaluator, call *ir.CallExpr, args []ir.Element) ([]ir.Element, error) {
	fType := f.fn.FuncType() // Some builtin functions have no type at the moment.
	if fType != nil && fType.CompEval {
		return f.callAtCompEval(fctx, call, args)
	}
	ctx := fctx.(*context.Context)
	res := call.Callee.T.Results.Fields()
	els := make([]ir.Element, len(res))
	for i, ri := range res {
		var err error
		els[i], err = NewRuntimeValue(ctx.File(), ctx.NewFunc, &ir.LocalVarStorage{
			Src: &ast.Ident{},
			Typ: ri.Type(),
		})
		if err != nil {
			return nil, err
		}
	}
	return els, nil
}

func (f *fun) Type() ir.Type {
	return f.fn.Type()
}
