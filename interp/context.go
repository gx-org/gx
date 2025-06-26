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

package interp

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

// EvalContext returns the context for the interpreter.
type EvalContext struct {
	options        []options.PackageOption
	packageOptions map[string][]packageOption
	evaluator      evaluator.Evaluator
	builtin        *baseFrame

	packageToFrame map[*ir.Package]*packageFrame
}

var _ evaluator.EvaluationContext = (*EvalContext)(nil)

// NewInterpContext returns a new interpreter context.
func NewInterpContext(eval evaluator.Evaluator, options []options.PackageOption) (*EvalContext, error) {
	pkgctx := &EvalContext{
		options:        options,
		packageToFrame: make(map[*ir.Package]*packageFrame),
		packageOptions: make(map[string][]packageOption),
		evaluator:      eval,
	}
	if err := pkgctx.buildBuiltinFrame(); err != nil {
		return nil, err
	}
	if err := pkgctx.processOptions(options); err != nil {
		return nil, err
	}
	return pkgctx, nil
}

// Evaluator returns the evaluator used by in evaluations.
func (eval *EvalContext) Evaluator() evaluator.Evaluator {
	return eval.evaluator
}

// context of an evaluation while running the interpreter.
// It contains the current frame stack, values of variables,
// and the evaluator to execute operations.
type context struct {
	eval       *EvalContext
	callInputs *elements.InputElements
	stack      []*blockFrame
}

var _ evaluator.Context = (*context)(nil)

// NewFileContext returns a context for a given file.
func (eval *EvalContext) NewFileContext(file *ir.File) (evaluator.Context, error) {
	return eval.newFileContext(file)
}

func (eval *EvalContext) newFileContext(file *ir.File) (*context, error) {
	ctx := &context{eval: eval}
	flFrame, err := ctx.fileFrame(file)
	if err != nil {
		return nil, err
	}
	flFrame.pushFuncFrame(ctx, nil)
	return ctx, nil
}

func (ctx *context) EvalFunc(f ir.Func, call *ir.CallExpr, args []elements.Element) ([]elements.Element, error) {
	fnEl := NewRunFunc(f, nil)
	return fnEl.Call(ctx, call, args)
}

// Evaluator returns the evaluator used by the context.
func (ctx *context) Evaluation() evaluator.EvaluationContext {
	return ctx.eval
}

// CallInputs returns the value with which a function has been called.
func (ctx *context) CallInputs() *elements.InputElements {
	return ctx.callInputs
}

func (ctx *context) pushFrame(fr *blockFrame) *blockFrame {
	ctx.stack = append(ctx.stack, fr)
	return fr
}

func (ctx *context) popFrame() {
	ctx.stack = ctx.stack[:len(ctx.stack)-1]
}

func (ctx *context) currentFrame() *blockFrame {
	return ctx.stack[len(ctx.stack)-1]
}

func (ctx *context) set(tok token.Token, dest ir.Storage, value elements.Element) error {
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
		receiver, err := ctx.evalExpr(destT.Sel.X)
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

func (ctx *context) find(id *ast.Ident) (elements.Element, error) {
	value, exists := ctx.currentFrame().Find(id.Name)
	if !exists {
		return nil, fmterr.Errorf(ctx.File().FileSet(), id, "undefined: %s", id.Name)
	}
	return value, nil
}

// valueOf is a convenient function only used for debugging.
// It returns a string representation of a given variable name.
func (ctx *context) valueOf(s string) string {
	val, ok := ctx.currentFrame().Find(s)
	if !ok {
		return fmt.Sprintf("undefined: %s", s)
	}
	return fmt.Sprint(val)
}

// Sub returns a child context given a set of elements.
func (ctx *context) Sub(elts map[string]elements.Element) (evaluator.Context, error) {
	sub := &context{
		eval:       ctx.eval,
		callInputs: ctx.callInputs,
		stack:      append([]*blockFrame{}, ctx.stack...),
	}
	bFrame := sub.pushBlockFrame()
	for n, elt := range elts {
		bFrame.Define(n, elt)
	}
	return sub, nil
}

// File returns the current file the interpreter is running code from.
func (ctx *context) File() *ir.File {
	return ctx.currentFrame().owner.parent.file
}

func (ctx *context) String() string {
	s := strings.Builder{}
	for i, fr := range ctx.stack {
		s.WriteString(fmt.Sprintf("Stack %d:\n%s", i, gxfmt.Indent(fr.String())))
	}
	return s.String()
}
