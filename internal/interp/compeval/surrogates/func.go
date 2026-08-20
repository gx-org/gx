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

package surrogates

import (
	"go/ast"

	"github.com/pkg/errors"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/interp/compeval/surrogates/storepath"
	"github.com/gx-org/gx/interp/fun"
)

// Function is a surrogate function.
// When called, it is not executed and returns surrogate values instead.
type Function struct {
	store ir.Storage
	fn    ir.Func
	recv  *fun.Receiver
}

// NewSurFunc returns a new surrogate function.
func NewSurFunc(fn ir.Func, recv *fun.Receiver) *Function {
	store, _ := fn.(ir.Storage)
	return &Function{
		fn:    fn,
		recv:  recv,
		store: store,
	}
}

// NewFunc returns a new surrogate function.
func NewFunc(fn ir.Func, recv *fun.Receiver) fun.Func {
	return NewSurFunc(fn, recv)
}

type surrogateFunction interface {
	Element
	fun.Func
}

func newSurrogateFunc(path storepath.Path, fType *ir.FuncType) surrogateFunction {
	fn := &ir.FuncLit{
		Src:   &ast.FuncLit{Type: fType.Src},
		FType: fType,
	}
	return &Function{store: path.Store(), fn: fn}
}

// IR definition of the function.
func (f *Function) IR() ir.Func {
	return f.fn
}

// Recv returns the receiver for the function.
func (f *Function) Recv() *fun.Receiver {
	return f.recv
}

// Call returns surrogate values for all results of a function simulating a function call.
func Call(call *ir.FuncCallExpr) ([]ir.Element, error) {
	res := call.Callee.FuncType().Results.Fields()
	els := make([]ir.Element, len(res))
	for i, ri := range res {
		var err error
		els[i], err = New(storepath.NewUniqueIR(call), ri.Type())
		if err != nil {
			return nil, err
		}
	}
	return els, nil
}

// Call the surrogate function.
func (f *Function) Call(env *fun.CallEnv, call *ir.FuncCallExpr, args []ir.Element) ([]ir.Element, error) {
	return Call(call)
}

// Type of the function.
func (f *Function) Type() ir.Type {
	return f.fn.Type()
}

// Store returns the storage referencing the function.
func (f *Function) Store() ir.Storage {
	return f.store
}

// Expr return the function as an expression.
func (f *Function) Expr(ir.Evaluator, ast.Expr) ([]ir.Expr, error) {
	if f.store == nil {
		return nil, errors.Errorf("anonymous function has no expression")
	}
	return []ir.Expr{ir.NewIdent(f.store)}, nil
}
