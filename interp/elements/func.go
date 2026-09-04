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
type FuncBuiltin func(env *engine.Env, call *ir.FuncCallExpr, recv ir.Element, args []ir.Element) ([]ir.Element, error)

type caller func(env *engine.Env, call *ir.FuncCallExpr, recv engine.Copier, args []ir.Element) ([]ir.Element, error)

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
	base := elFunc{fn: fn, recv: recv}
	switch fnT := fn.(type) {
	case *ir.AnnotatorField:
		return newFieldAnnotator(fnT, recv)
	case *ir.AnnotatorFunc:
		return newFuncAnnotator(fnT, recv)
	case *ir.FuncDecl:
		base.storage = fnT
		base.call = funcDecl{fnT: fnT}.callDecl
	case *ir.FuncBuiltin:
		base.storage = fnT
		base.call = funcBuiltin{fun: fnT, impl: fnT.Impl}.callBuiltin
	case *ir.FuncKeyword:
		base.storage = fnT
		base.call = funcBuiltin{fun: fnT, impl: fnT.Impl}.callBuiltin
	case *ir.Macro:
		return NewMacro(fnT, nil)
	}
	return &base
}

// Type of the function.
func (f *elFunc) Type() ir.Type {
	return f.fn.FuncType()
}

// IR returns the function represented by the node.
func (f *elFunc) IR() ir.Func {
	return f.fn
}

// Recv returns the receiver of the function or nil if the function has no receiver.
func (f *elFunc) Recv() *engine.Receiver {
	return f.recv
}

// Unflatten creates a GX value from the next handles available in the parser.
func (f *elFunc) Unflatten(handles *flatten.Parser) (hostio.Value, error) {
	return values.NewIRNode(f.fn)
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
func (f *elFunc) Call(env *engine.Env, call *ir.FuncCallExpr, args []ir.Element) ([]ir.Element, error) {
	if f.call == nil {
		return nil, fmterr.InternalAt(env.File().FileSet(), f.fn.Node(), "function type %T not supported", f.fn)
	}
	var recv engine.NamedType
	if f.Recv() != nil {
		recv = f.Recv().Element
	}
	return f.call(env, call, recv, args)
}

// String representation of the node.
func (f *elFunc) String() string {
	return f.fn.DefineString(nil)
}

type funcDecl struct {
	fnT *ir.FuncDecl
}

func (f funcDecl) callDecl(env *engine.Env, call *ir.FuncCallExpr, recv engine.Copier, args []ir.Element) (outs []ir.Element, err error) {
	return env.Runners().FuncDecl(f.fnT, env, call, recv, args)
}

type funcBuiltin struct {
	fun  ir.Func
	impl ir.FuncImpl
}

func (f funcBuiltin) callBuiltin(env *engine.Env, call *ir.FuncCallExpr, recv engine.Copier, args []ir.Element) (outs []ir.Element, err error) {
	return env.Runners().Builtin(f.fun, f.impl, env, call, recv, args)
}
