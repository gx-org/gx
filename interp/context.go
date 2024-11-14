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
	"go/ast"
	"go/token"

	"github.com/gx-org/backend"
	"github.com/gx-org/backend/graph"
	"github.com/gx-org/backend/shape"
	"github.com/gx-org/gx/api/values"
	"github.com/gx-org/gx/build/fmterr"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/interp/state"
)

// Context is the evaluation of the GX interpreter.
type Context interface {
	state.Context

	// FileSet returns the fileset of the context.
	FileSet() fmterr.FileSet

	// Graph returns the current state of the interpreter.
	State() *state.State

	// Backend returns the backend for which the interpreter is running.
	Backend() backend.Backend
}

// context of an evaluation while running the interpreter.
// It contains the current frame stack, values of variables,
// and the graph being built.
type context struct {
	itrp     *Interpreter
	state    *state.State
	receiver values.Value
	args     []values.Value

	packageToFrame map[*ir.Package]*frame
	fileToFrame    map[*ir.File]*frame

	builtin *frame
	stack   []*frame
}

var _ Context = (*context)(nil)

func newContext(itrp *Interpreter, g *state.State, fn ir.Func, receiver values.Value, args []values.Value) *context {
	ctx := &context{
		itrp:           itrp,
		state:          g,
		args:           args,
		receiver:       receiver,
		packageToFrame: make(map[*ir.Package]*frame),
		fileToFrame:    make(map[*ir.File]*frame),
	}
	ctx.builtin = ctx.newBuiltinFrame(fn)
	return ctx
}

func (ctx *context) FileSet() fmterr.FileSet {
	frame := ctx.builtin
	if len(ctx.stack) > 0 {
		frame = ctx.currentFrame()
	}
	return fmterr.FileSet{FSet: frame.function.File().Package.FSet}
}

func (ctx *context) State() *state.State {
	return ctx.state
}

func (ctx *context) Backend() backend.Backend {
	return ctx.itrp.backend
}

func (ctx *context) Receiver() values.Value {
	return ctx.receiver
}

func (ctx *context) Args() []values.Value {
	return ctx.args
}

func (ctx *context) packageFrame(pkg *ir.Package) (*frame, error) {
	packageFrame := ctx.packageToFrame[pkg]
	if packageFrame != nil {
		return packageFrame, nil
	}
	var err error
	packageFrame, err = ctx.newPackageFrame(pkg)
	if err != nil {
		return nil, err
	}
	ctx.packageToFrame[pkg] = packageFrame
	return packageFrame, nil
}

func (ctx *context) fileFrame(file *ir.File) (*frame, error) {
	fileFrame := ctx.fileToFrame[file]
	if fileFrame != nil {
		return fileFrame, nil
	}
	packageFrame, err := ctx.packageFrame(file.Package)
	if err != nil {
		return nil, err
	}
	fileFrame = packageFrame.newFileFrame(file)
	ctx.fileToFrame[file] = fileFrame
	return fileFrame, nil
}

func (ctx *context) pushFuncFrame(fn ir.Func) (*frame, error) {
	fileFrame, err := ctx.fileFrame(fn.File())
	if err != nil {
		return nil, err
	}
	frame := fileFrame.newFunctionFrame(fn)
	ctx.stack = append(ctx.stack, frame)
	return frame, nil
}

func (ctx *context) pushBlockFrame() *frame {
	frame := ctx.currentFrame().newBlockFrame()
	ctx.stack = append(ctx.stack, frame)
	return frame
}

func (ctx *context) popFrame() {
	ctx.stack = ctx.stack[:len(ctx.stack)-1]
}

func (ctx *context) currentFrame() *frame {
	return ctx.stack[len(ctx.stack)-1]
}

func (ctx *context) set(tok token.Token, dest ir.Assignable, value state.Element) error {
	fr := ctx.currentFrame()
	switch destT := dest.(type) {
	case *ir.LocalVarAssign:
		fr.set(tok, destT.Src, value)
		return nil
	case *ir.StructFieldAssign:
		receiver, err := evalExpr(ctx, destT.X)
		if err != nil {
			return err
		}
		strt, ok := receiver.(*state.Struct)
		if !ok {
			return ctx.FileSet().Errorf(dest.Source(), "cannot convert %T to %T", receiver, strt)
		}
		strt.SetField(destT.FieldID, value)
		return nil
	default:
		return ctx.FileSet().Errorf(dest.Source(), "cannot assign %v to %T: not supported", value, destT)
	}
}

func (ctx *context) find(id *ast.Ident) (state.Element, error) {
	return ctx.currentFrame().find(id)
}

func (ctx *context) evalNumericalExpr(x ir.Expr) (graph.Node, *shape.Shape, error) {
	el, err := evalExpr(ctx, x)
	if err != nil {
		return nil, nil, err
	}
	return state.NodeFromElement(el)
}

func (ctx *context) currentFile() *ir.File {
	return ctx.currentFrame().function.File()
}

func (ctx *context) exprAt(expr ir.Expr) state.ExprAt {
	return nodeAt(ctx, expr)
}

func nodeAt[T ir.Expr](ctx *context, node T) state.ExprFile[T] {
	return state.NewExprAt(ctx.currentFile(), node)
}
