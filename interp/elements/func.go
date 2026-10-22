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

package elements

import (
	"github.com/gx-org/gx/api/hostio"
	"github.com/gx-org/gx/api/values"
	"github.com/gx-org/gx/build/fmterr"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/build/ir/irkind"
	"github.com/gx-org/gx/internal/interp/flatten"
	"github.com/gx-org/gx/interp/engine"
)

// FuncBuiltin defines a builtin function provided by a backend.
type FuncBuiltin func(env *engine.Env, call *ir.FuncCallExpr, recv ir.Element, genArgs, args []ir.Element) ([]ir.Element, error)

type caller func(env *engine.Env, call *ir.FuncCallExpr, recv engine.Copier, genArgs, args []ir.Element) ([]ir.Element, error)

func funcDeclCaller(f *ir.FuncDecl) caller {
	return func(env *engine.Env, call *ir.FuncCallExpr, recv engine.Copier, genArgs, args []ir.Element) (outs []ir.Element, err error) {
		return env.Runners().FuncDecl(f, env.For(f.FType.Nature), call, recv, genArgs, args)
	}
}

func funcBuiltinCaller(f *ir.FuncBuiltin) caller {
	return func(env *engine.Env, call *ir.FuncCallExpr, recv engine.Copier, genArgs, args []ir.Element) (outs []ir.Element, err error) {
		return env.Runners().Builtin(f, env.For(call.Callee.FuncType().Nature), call, recv, genArgs, args)
	}
}

func funcKeywordCaller(f *ir.FuncKeyword) caller {
	return func(env *engine.Env, call *ir.FuncCallExpr, recv engine.Copier, genArgs, args []ir.Element) (outs []ir.Element, err error) {
		return env.Runners().Keyword(f, env.For(call.Callee.FuncType().Nature), call, recv, genArgs, args)
	}
}

type elFunc struct {
	fn      ir.Func
	recv    *engine.Receiver
	storage ir.Storage

	call caller
}

var (
	_ engine.Func  = (*elFunc)(nil)
	_ ir.WithStore = (*elFunc)(nil)
)

// NewFunc creates a function given an IR and a receiver.
// The function is run when being called.
func NewFunc(fn ir.Func, recv *engine.Receiver) engine.Func {
	elfn := elFunc{fn: fn, recv: recv}
	switch fnT := fn.(type) {
	case *ir.AnnotatorField:
		return newFieldAnnotator(fnT, recv)
	case *ir.AnnotatorFunc:
		return newFuncAnnotator(fnT, recv)
	case *ir.FuncDecl:
		elfn.storage = fnT
		elfn.call = funcDeclCaller(fnT)
	case *ir.FuncBuiltin:
		elfn.storage = fnT
		elfn.call = funcBuiltinCaller(fnT)
	case *ir.FuncKeyword:
		elfn.storage = fnT
		elfn.call = funcKeywordCaller(fnT)
	case *ir.Macro:
		return NewMacro(fnT)
	}
	return &elfn
}

// Type of the function.
func (f *elFunc) Type() ir.Type {
	return f.fn.FuncType()
}

// FuncIR returns the function represented by the node.
func (f *elFunc) FuncIR() ir.Func {
	return f.fn
}

// Recv returns the receiver of the function or nil if the function has no receiver.
func (f *elFunc) Recv() *engine.Receiver {
	return f.recv
}

// Unflatten creates a GX value from the next handles available in the parser.
func (f *elFunc) Unflatten(handles *flatten.Parser) (hostio.Value, error) {
	return values.NewIRNode(f.fn), nil
}

// Kind of the element.
func (*elFunc) Kind() irkind.Kind {
	return irkind.Func
}

// Storage of the function.
func (f *elFunc) Store() ir.Storage {
	return f.storage
}

// Call the function.
func (f *elFunc) Call(env *engine.Env, call *ir.FuncCallExpr, genArgs, args []ir.Element) ([]ir.Element, error) {
	if f.call == nil {
		return nil, fmterr.InternalAt(env.File().FileSet(), f.fn.Node(), "function type %T not supported", f.fn)
	}
	var recv engine.NamedType
	if f.Recv() != nil {
		recv = f.Recv().Element
	}
	return f.call(env, call, recv, genArgs, args)
}

// String representation of the node.
func (f *elFunc) String() string {
	return f.fn.DefineString(nil)
}
