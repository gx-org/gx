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
	"github.com/gx-org/backend/dtype"
	"github.com/gx-org/backend/graph"
	"github.com/gx-org/backend/shape"
	"github.com/gx-org/gx/api/values"
	"github.com/gx-org/gx/build/fmterr"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/base/scope"
	"github.com/gx-org/gx/internal/tracer"
	"github.com/gx-org/gx/interp/elements"
	"github.com/gx-org/gx/interp/state"
)

type (
	frameI interface {
		pushBlockFrame() *frame
		pushFuncFrame(fn ir.Func) (*frame, error)
		popFrame()

		set(tok token.Token, dest ir.Assignable, value state.Element) error
		find(id *ast.Ident) (state.Element, error)

		currentFile() *ir.File
		currentFrame() *frame
	}

	// Evaluator implements GX operators.
	Evaluator interface {
		// UnaryOp applies a unary operator on x.
		UnaryOp(ctx tracer.Context, expr *ir.UnaryExpr, x state.Element) (state.Element, error)
		// BinaryOp applies a binary operator to x and y.
		BinaryOp(ctx tracer.Context, expr *ir.BinaryExpr, x, y state.Element) (state.Element, error)
		// CallFuncLit calls a function literal.
		CallFuncLit(ctx tracer.Context, ref *ir.FuncLit, args []state.Element) (state.Element, error)
		// Einsum calls an einstein sum on x and y given the expression in ref.
		Einsum(ctx tracer.Context, ref *ir.EinsumExpr, x, y state.Element) (state.Element, error)
		// Reshape an element into a given shape.
		Reshape(ctx tracer.Context, expr ir.Expr, x state.Element, axisLengths []int) (state.Element, error)
		// Cast an element into a given data type.
		Cast(ctx tracer.Context, expr ir.Expr, x state.Element, dtype dtype.DataType) (state.Element, error)
		// Concat concatenates scalars elements into an array with one axis.
		Concat(ctx tracer.Context, expr ir.Expr, xs []state.Element) (state.Element, error)
		// Set a slice in an array.
		Set(ctx tracer.Context, call *ir.CallExpr, x, updates, index state.Element) (state.Element, error)
		// ElementFromValue returns an element from a GX value.
		ElementFromValue(src elements.NodeAt, val values.Array) (elements.NumericalElement, error)
	}

	// Context is the evaluation of the GX interpreter.
	Context interface {
		// CallInputs returns the receiver and arguments with which the function was called.
		CallInputs() *elements.CallInputs

		// FileSet returns the fileset of the context.
		FileSet() fmterr.FileSet

		// Graph returns the current state of the interpreter.
		State() *state.State

		// Backend returns the backend for which the interpreter is running.
		Backend() backend.Backend

		// BuildGraph builds a function in the given graph within a new Context.
		BuildGraph(fn ir.Func, g graph.Graph, args []state.Element) (state.Element, error)

		// File returns the current file the interpreter is running code from.
		File() *ir.File

		// ExprAt returns an expression located in a file.
		ExprAt(expr ir.Expr) elements.ExprAt

		// Evaluator returns the evaluator used by the context.
		Evaluator() Evaluator

		frame() frameI
	}
)

type frameState struct {
	Context
	file     *ir.File
	function ir.Func
}

type frame = scope.Scope[frameState, state.Element]

// context of an evaluation while running the interpreter.
// It contains the current frame stack, values of variables,
// and the graph being built.
type context struct {
	itrp       *Interpreter
	state      *state.State
	callInputs *elements.CallInputs

	packageToFrame map[*ir.Package]*frame
	fileToFrame    map[*ir.File]*frame

	builtin *frame
	stack   []*frame

	evaluator Evaluator
}

var _ Context = (*context)(nil)

func newContext(itrp *Interpreter, g *state.State, fn ir.Func, receiver values.Value, args []values.Value) (*context, error) {
	ctx := &context{
		itrp:  itrp,
		state: g,
		callInputs: &elements.CallInputs{
			Args:     args,
			Receiver: receiver,
		},
		packageToFrame: make(map[*ir.Package]*frame),
		fileToFrame:    make(map[*ir.File]*frame),
		evaluator:      tracer.NewEvaluator(g),
	}
	return ctx, ctx.buildBuiltinFrame()
}

// Evaluator returns the evaluator used by the context.
func (ctx *context) Evaluator() Evaluator {
	return ctx.evaluator
}

func (ctx *context) FileSet() fmterr.FileSet {
	frame := ctx.builtin
	if len(ctx.stack) > 0 {
		frame = ctx.currentFrame()
	}
	return fmterr.FileSet{FSet: frame.Context().file.Package.FSet}
}

func (ctx *context) State() *state.State {
	return ctx.state
}

func (ctx *context) Backend() backend.Backend {
	return ctx.itrp.backend
}

func (ctx *context) CallInputs() *elements.CallInputs {
	return ctx.callInputs
}

func (ctx *context) frame() frameI {
	return ctx
}

func (ctx *context) BuildGraph(fn ir.Func, g graph.Graph, args []state.Element) (state.Element, error) {
	subctx, err := newContext(ctx.itrp, state.New(fn, g), fn, nil, nil)
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
	defer subctx.popFrame()

	var body *ir.BlockStmt
	switch fn := fn.(type) {
	case *ir.FuncDecl:
		body = fn.Body
	case *ir.FuncLit:
		body = fn.Body
	}
	return evalFuncBody(subctx, body)
}

func (ctx *context) packageFrame(pkg *ir.Package) (*frame, error) {
	packageFrame := ctx.packageToFrame[pkg]
	if packageFrame != nil {
		return packageFrame, nil
	}
	packageFrame = ctx.builtin.NewChild(frameState{Context: ctx})
	for _, f := range pkg.Funcs {
		packageFrame.Define(f.Name(), elements.NewFunc(f, nil))
	}
	ctx.packageToFrame[pkg] = packageFrame
	if err := ctx.evalPackageConsts(pkg, packageFrame); err != nil {
		return nil, err
	}
	if err := ctx.evalPackageOptions(pkg, packageFrame); err != nil {
		return nil, err
	}
	return packageFrame, nil
}

func (ctx *context) evalPackageConst(cst *ir.ConstDecl, fr *frame) error {
	fileFrame, err := ctx.fileFrame(cst.FFile)
	if err != nil {
		return err
	}
	ctx.stack = append(ctx.stack, fileFrame)
	defer ctx.popFrame()
	for _, expr := range cst.Exprs {
		var el state.Element
		if ir.IsNumber(expr.Value.Type().Kind()) {
			el = elements.NewNumber(expr.Value)
		} else {
			el, err = evalValue(ctx, expr.Value)
		}
		if err != nil {
			return err
		}
		fr.Define(expr.VName.Name, el)
	}
	return nil
}

func (ctx *context) evalPackageConsts(pkg *ir.Package, fr *frame) error {
	for _, cst := range pkg.Consts {
		if err := ctx.evalPackageConst(cst, fr); err != nil {
			return err
		}
	}
	return nil
}

func (ctx *context) evalPackageOptions(pkg *ir.Package, fr *frame) error {
	options := ctx.itrp.packageOptions[pkg.FullName()]
	for _, option := range options {
		if err := option(ctx, pkg, fr); err != nil {
			return err
		}
	}
	return nil
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
	fileFrame = packageFrame.NewChild(frameState{Context: ctx, file: file})
	fset := fmterr.FileSet{FSet: file.Package.FSet}
	for _, imp := range file.Imports {
		fileFrame.Define(imp.Name().Name, elements.NewPackage(fset.Pos(imp.Source()), imp.Package))
	}
	ctx.fileToFrame[file] = fileFrame
	return fileFrame, nil
}

func (ctx *context) pushFuncFrame(fn ir.Func) (*frame, error) {
	fileFrame, err := ctx.fileFrame(fn.File())
	if err != nil {
		return nil, err
	}
	frame := fileFrame.NewChild(frameState{Context: ctx, file: fn.File(), function: fn})
	ctx.stack = append(ctx.stack, frame)
	return frame, nil
}

func (ctx *context) pushBlockFrame() *frame {
	parent := ctx.currentFrame()
	frame := parent.NewChild(frameState{Context: parent.Context(), file: parent.Context().file, function: parent.Context().function})
	ctx.stack = append(ctx.stack, frame)
	return frame
}

func (ctx *context) popFrame() {
	ctx.stack = ctx.stack[:len(ctx.stack)-1]
}

func (ctx *context) currentFrame() *frame {
	if len(ctx.stack) == 0 {
		return ctx.builtin
	}
	return ctx.stack[len(ctx.stack)-1]
}

func (ctx *context) set(tok token.Token, dest ir.Assignable, value state.Element) error {
	fr := ctx.currentFrame()
	switch destT := dest.(type) {
	case *ir.LocalVarAssign:
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
	case *ir.StructFieldAssign:
		receiver, err := evalExpr(ctx, destT.X)
		if err != nil {
			return err
		}
		strt, ok := receiver.(*elements.Struct)
		if !ok {
			return ctx.FileSet().Errorf(dest.Source(), "cannot convert %T to %T", receiver, strt)
		}
		strt.SetField(destT.FieldName, value)
		return nil
	default:
		return ctx.FileSet().Errorf(dest.Source(), "cannot assign %v to %T: not supported", value, destT)
	}
}

func (ctx *context) find(id *ast.Ident) (state.Element, error) {
	value, exists := ctx.currentFrame().Find(id.Name)
	if !exists {
		return nil, ctx.FileSet().Errorf(id, "undefined: %s", id.Name)
	}
	return value, nil
}

func evalNumericalExpr(ctx *context, x ir.Expr) (graph.Node, *shape.Shape, error) {
	el, err := evalExpr(ctx, x)
	if err != nil {
		return nil, nil, err
	}
	return state.NodeFromElement(el)
}

func (ctx *context) currentFile() *ir.File {
	return ctx.currentFrame().Context().file
}

// ExprAt returns an expression located in a file.
func (ctx *context) ExprAt(expr ir.Expr) elements.ExprAt {
	return nodeAt(ctx, expr)
}

// File returns the current file the interpreter is running code from.
func (ctx *context) File() *ir.File {
	return ctx.frame().currentFile()
}

func nodeAt[T ir.Node](ctx Context, node T) elements.NodeFile[T] {
	return elements.NewNodeAt(ctx.File(), node)
}
