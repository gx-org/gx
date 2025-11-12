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

// Package context maintains the namespaces and stack for the interpreter.
package context

import (
	"fmt"
	"go/ast"
	"strings"

	gxfmt "github.com/gx-org/gx/base/fmt"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/base/scope"
)

type (
	// Interpreter evaluates expressions and statements given the context.
	Interpreter interface {
		// InitPkgScope initialises the namespace of a package.
		InitPkgScope(pkg *ir.Package, scope *scope.RWScope[ir.Element]) (ir.PackageElement, error)

		// InitBuiltins initialises a namespace with GX builtins implementation.
		InitBuiltins(ctx *Context, scope *scope.RWScope[ir.Element]) error

		// PackageToImport encapsulates a package into its import declaration.
		PackageToImport(imp *ir.ImportDecl, pkg ir.PackageElement) ir.Element
	}

	// Core contains everything in the context independent of code location.
	Core struct {
		interp Interpreter

		builtin        *baseFrame
		packageToFrame map[*ir.Package]*packageFrame
		importer       ir.Importer
	}
)

// New returns a new interpreter context.
func New(interp Interpreter, importer ir.Importer) (*Core, error) {
	core := &Core{
		interp:         interp,
		packageToFrame: make(map[*ir.Package]*packageFrame),
		importer:       importer,
	}
	core.builtin = &baseFrame{scope: scope.NewScope[ir.Element](nil)}
	ctx, err := core.NewFileContext(&ir.File{
		Package: &ir.Package{
			Name:  &ast.Ident{Name: "<interp>"},
			Decls: &ir.Declarations{},
		},
	})
	if err != nil {
		return nil, err
	}
	return core, interp.InitBuiltins(ctx, core.builtin.scope)
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
	return &Frame{file: ctx.File(), scope: ctx.currentFrame().scope}
}

// Core context used by this context.
func (ctx *Context) Core() *Core {
	return ctx.core
}

func (ctx *Context) currentFrame() *blockFrame {
	return ctx.stack[len(ctx.stack)-1]
}

// CurrentFunc returns the current function being run.
func (ctx *Context) CurrentFunc() ir.Func {
	return ctx.currentFrame().owner.function
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

// Scope returns the scope for the current context.
func (ctx *Context) Scope() *scope.RWScope[ir.Element] {
	return ctx.currentFrame().scope
}

// File returns the current file the interpreter is running code from.
func (ctx *Context) File() *ir.File {
	return ctx.currentFrame().owner.parent.file
}

func (ctx *Context) String() string {
	s := strings.Builder{}
	for i, fr := range ctx.stack {
		fmt.Fprintf(&s, "Stack %d:\n%s", i, gxfmt.Indent(fr.String()))
	}
	return s.String()
}
