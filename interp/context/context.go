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
	"go/ast"
	"go/token"
	"strings"

	"github.com/gx-org/gx/api/options"
	gxfmt "github.com/gx-org/gx/base/fmt"
	"github.com/gx-org/gx/build/fmterr"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/interp/elements"
	"github.com/gx-org/gx/interp/evaluator"
)

type (
	// Interpreter evaluates expressions and statements given the context.
	Interpreter interface {
		EvalExpr(*Context, ir.Expr) (ir.Element, error)
		EvalStmt(*Context, *ir.BlockStmt) ([]ir.Element, bool, error)
	}

	// Context of an evaluation while running the interpreter.
	// It contains the current frame stack, values of variables,
	// and the evaluator to execute operations.
	Context struct {
		interp Interpreter

		options        []options.PackageOption
		packageOptions map[string][]packageOption
		evaluator      evaluator.Evaluator
		builtin        *baseFrame
		packageToFrame map[*ir.Package]*packageFrame

		callInputs *elements.InputElements
		stack      []*blockFrame
	}
)

var _ evaluator.Context = (*Context)(nil)

// New returns a new interpreter context.
func New(interp Interpreter, eval evaluator.Evaluator, options []options.PackageOption) (*Context, error) {
	ctx := &Context{
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

// Evaluator returns the evaluator used by in evaluations.
func (ctx *Context) Evaluator() evaluator.Evaluator {
	return ctx.evaluator
}

// NewFileContext returns a context for a given file.
func (ctx *Context) NewFileContext(file *ir.File) (evaluator.Context, error) {
	return ctx.newFileContext(file)
}

func (ctx *Context) branch() *Context {
	return &Context{
		interp:         ctx.interp,
		options:        ctx.options,
		packageOptions: ctx.packageOptions,
		evaluator:      ctx.evaluator,
		builtin:        ctx.builtin,
		packageToFrame: ctx.packageToFrame,
	}
}

func (ctx *Context) newFileContext(file *ir.File) (*Context, error) {
	n := ctx.branch()
	flFrame, err := n.fileFrame(file)
	if err != nil {
		return nil, err
	}
	flFrame.pushFuncFrame(n, nil)
	return n, nil
}

// EvalFunc evaluates a function.
func (ctx *Context) EvalFunc(f ir.Func, call *ir.CallExpr, args []ir.Element) ([]ir.Element, error) {
	fnEl := NewRunFunc(f, nil)
	return fnEl.Call(ctx, call, args)
}

// EvalExpr evaluates an expression in this context.
func (ctx *Context) EvalExpr(expr ir.Expr) (ir.Element, error) {
	return ctx.interp.EvalExpr(ctx, expr)
}

// CallInputs returns the value with which a function has been called.
func (ctx *Context) CallInputs() *elements.InputElements {
	return ctx.callInputs
}

func (ctx *Context) pushFrame(fr *blockFrame) *blockFrame {
	ctx.stack = append(ctx.stack, fr)
	return fr
}

// PopFrame pops the current frame from the stack.
func (ctx *Context) PopFrame() {
	ctx.stack = ctx.stack[:len(ctx.stack)-1]
}

func (ctx *Context) currentFrame() *blockFrame {
	return ctx.stack[len(ctx.stack)-1]
}

// Set the value of a given storage.
func (ctx *Context) Set(tok token.Token, dest ir.Storage, value ir.Element) error {
	fr := ctx.currentFrame()
	switch destT := dest.(type) {
	case *ir.LocalVarStorage:
		if !ir.ValidIdent(destT.Src) {
			return nil
		}
		if tok == token.ILLEGAL {
			return nil
		}
		if tok == token.DEFINE {
			fr.Define(destT.Src.Name, value)
			return nil
		}
		return fr.Assign(destT.Src.Name, value)
	case *ir.StructFieldStorage:
		receiver, err := ctx.EvalExpr(destT.Sel.X)
		if err != nil {
			return err
		}
		strt, ok := elements.Underlying(receiver).(*elements.Struct)
		if !ok {
			return fmterr.Errorf(ctx.File().FileSet(), dest.Source(), "cannot convert %T to %T", receiver, strt)
		}
		strt.SetField(destT.Sel.Src.Sel.Name, value)
		return nil
	case *ir.FieldStorage:
		return fr.Assign(destT.Field.Name.Name, value)
	case *ir.AssignExpr:
		return fr.Assign(destT.NameDef().Name, value)
	default:
		return fmterr.Errorf(ctx.File().FileSet(), dest.Source(), "cannot assign %v to %T: not supported", value, destT)
	}
}

// CurrentFunc returns the current function being run.
func (ctx *Context) CurrentFunc() ir.Func {
	return ctx.currentFrame().owner.function
}

// Find the element in the stack of frame given its identifier.
func (ctx *Context) Find(id *ast.Ident) (ir.Element, error) {
	value, exists := ctx.currentFrame().Find(id.Name)
	if !exists {
		return nil, fmterr.Errorf(ctx.File().FileSet(), id, "undefined: %s", id.Name)
	}
	return value, nil
}

// valueOf is a convenient function only used for debugging.
// It returns a string representation of a given variable name.
func (ctx *Context) valueOf(s string) string {
	val, ok := ctx.currentFrame().Find(s)
	if !ok {
		return fmt.Sprintf("undefined: %s", s)
	}
	return fmt.Sprint(val)
}

// Sub returns a child context given a set of elements.
func (ctx *Context) Sub(elts map[string]ir.Element) (evaluator.Context, error) {
	sub := ctx.branch()
	sub.callInputs = ctx.callInputs
	sub.stack = append([]*blockFrame{}, ctx.stack...)
	bFrame := sub.PushBlockFrame()
	for n, elt := range elts {
		bFrame.Define(n, elt)
	}
	return sub, nil
}

// File returns the current file the interpreter is running code from.
func (ctx *Context) File() *ir.File {
	return ctx.currentFrame().owner.parent.file
}

// EvalFunctionToElement evaluates a function such as it becomes an element.
func (ctx *Context) EvalFunctionToElement(eval evaluator.Evaluator, fn ir.Func, args []ir.Element) ([]ir.Element, error) {
	subctx, err := ctx.newFileContext(fn.File())
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
