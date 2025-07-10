// Copyright 2024 Google LLC
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

package context

import (
	"fmt"
	"strings"

	"github.com/gx-org/gx/api/options"
	gxfmt "github.com/gx-org/gx/base/fmt"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/interp/elements"
	"github.com/gx-org/gx/interp/evaluator"
)

type (
	// Interpreter evaluates expressions and statements given the context.
	Interpreter interface {
		// EvalExpr runs an expression through the evaluator.
		EvalExpr(*Context, ir.Expr) (ir.Element, error)
		// EvalStmt runs a block of statements through the evaluator.
		EvalStmt(*Context, *ir.BlockStmt) ([]ir.Element, bool, error)
	}

	// Evaluator provides core primitives for the interpreter.
	Evaluator interface {
		evaluator.Evaluator

		// NewFunc creates a new function given its definition and a receiver.
		NewFunc(*Core, ir.Func, *elements.Receiver) elements.Func

		// CallFuncLit calls a function literal.
		CallFuncLit(ctx *Context, ref *ir.FuncLit, args []ir.Element) ([]ir.Element, error)
	}

	// Core contains everything in the context independent of code location.
	Core struct {
		interp Interpreter

		options        []options.PackageOption
		packageOptions map[string][]packageOption
		evaluator      Evaluator
		builtin        *baseFrame
		packageToFrame map[*ir.Package]*packageFrame
	}
)

var _ evaluator.Context = (*Context)(nil)

// New returns a new interpreter context.
func New(interp Interpreter, eval Evaluator, options []options.PackageOption) (*Core, error) {
	ctx := &Core{
		interp:         interp,
		options:        options,
		packageToFrame: make(map[*ir.Package]*packageFrame),
		packageOptions: make(map[string][]packageOption),
		evaluator:      eval,
	}
	if err := ctx.buildBuiltinFrame(); err != nil {
		return nil, err
	}
	if err := ctx.processOptions(options); err != nil {
		return nil, err
	}
	return ctx, nil
}

// NewFunc creates a new function given its definition and a receiver.
func (core *Core) NewFunc(fn ir.Func, recv *elements.Receiver) elements.Func {
	return core.evaluator.NewFunc(core, fn, recv)
}

// Context of an evaluation while running the interpreter.
// It contains the current frame stack, values of variables,
// and the evaluator to execute operations.
type Context struct {
	core  *Core
	stack []*blockFrame
}

// NewFileContext returns a context for a given file.
func (core *Core) NewFileContext(file *ir.File) (*Context, error) {
	n := &Context{core: core}
	flFrame, err := n.core.fileFrame(file)
	if err != nil {
		return nil, err
	}
	flFrame.pushFuncFrame(n, nil)
	return n, nil
}

// Evaluator returns the evaluator used by in evaluations.
func (ctx *Context) Evaluator() evaluator.Evaluator {
	return ctx.core.evaluator
}

// EvalFunc evaluates a function.
func (ctx *Context) EvalFunc(f ir.Func, call *ir.CallExpr, args []ir.Element) ([]ir.Element, error) {
	fnEl := NewRunFunc(f, nil)
	return fnEl.Call(ctx, call, args)
}

// EvalExpr evaluates an expression in this context.
func (ctx *Context) EvalExpr(expr ir.Expr) (ir.Element, error) {
	return ctx.core.interp.EvalExpr(ctx, expr)
}

func (ctx *Context) pushFrame(fr *blockFrame) *blockFrame {
	ctx.stack = append(ctx.stack, fr)
	return fr
}

// PopFrame pops the current frame from the stack.
func (ctx *Context) PopFrame() {
	ctx.stack = ctx.stack[:len(ctx.stack)-1]
}

// CurrentFrame returns the current frame.
func (ctx *Context) CurrentFrame() *Frame {
	return &Frame{file: ctx.File(), current: ctx.currentFrame()}
}

func (ctx *Context) currentFrame() *blockFrame {
	return ctx.stack[len(ctx.stack)-1]
}

// CurrentFunc returns the current function being run.
func (ctx *Context) CurrentFunc() ir.Func {
	return ctx.currentFrame().owner.function
}

// NewFunc creates a new function given its definition and a receiver.
func (ctx *Context) NewFunc(fn ir.Func, recv *elements.Receiver) elements.Func {
	return ctx.core.NewFunc(fn, recv)
}

// Sub returns a child context given a set of elements.
func (ctx *Context) Sub(elts map[string]ir.Element) *Context {
	sub := &Context{core: ctx.core}
	sub.stack = append([]*blockFrame{}, ctx.stack...)
	bFrame := sub.PushBlockFrame()
	for n, elt := range elts {
		bFrame.Define(n, elt)
	}
	return sub
}

// File returns the current file the interpreter is running code from.
func (ctx *Context) File() *ir.File {
	return ctx.currentFrame().owner.parent.file
}

// EvalFunctionToElement evaluates a function such as it becomes an element.
func (ctx *Context) EvalFunctionToElement(eval evaluator.Evaluator, fn ir.Func, args []ir.Element) ([]ir.Element, error) {
	subctx, err := ctx.core.NewFileContext(fn.File())
	if err != nil {
		return nil, err
	}
	funcFrame, err := subctx.pushFuncFrame(fn)
	if err != nil {
		return nil, err
	}

	assignArgumentValues(fn.FuncType(), funcFrame, args)
	for _, resultName := range fieldNames(fn.FuncType().Results.List) {
		funcFrame.Define(resultName.Name, nil)
	}
	defer subctx.PopFrame()

	var body *ir.BlockStmt
	switch fn := fn.(type) {
	case *ir.FuncDecl:
		body = fn.Body
	case *ir.FuncLit:
		body = fn.Body
	}
	return evalFuncBody(subctx, body)
}

func (ctx *Context) String() string {
	s := strings.Builder{}
	for i, fr := range ctx.stack {
		s.WriteString(fmt.Sprintf("Stack %d:\n%s", i, gxfmt.Indent(fr.String())))
	}
	return s.String()
}
