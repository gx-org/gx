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

	"github.com/pkg/errors"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/interp/state"
)

func callFunc(ctx Context, call *ir.CallExpr, fn *state.Func, args []state.Element) (output state.Element, err error) {
	switch fnT := fn.Func().(type) {
	case *ir.FuncDecl:
		output, err = callFuncDecl(ctx, fn, fnT, args)
	case *ir.FuncBuiltin:
		output, err = callFuncBuiltin(ctx, call, fn, fnT, args)
	case *ir.FuncLit:
		output, err = ctx.Evaluator().CallFuncLit(ctx, fnT, args)
	default:
		err = errors.Errorf("calling function of type %T not supported", fnT)
	}
	return
}

func callFuncBuiltin(ctx Context, call *ir.CallExpr, fn *state.Func, irFunc *ir.FuncBuiltin, args []state.Element) (output state.Element, err error) {
	defer func() {
		if err != nil {
			err = ctx.FileSet().Position(call.Expr(), err)
		}
	}()
	var impl FuncBuiltin
	if irFunc.Impl != nil {
		impl = irFunc.Impl.Implementation().(FuncBuiltin)
	}
	if impl == nil {
		err = errors.Errorf("function %s has no backend implementation", irFunc.Name())
		return
	}
	return impl(ctx, nodeAt[*ir.CallExpr](ctx, call), fn, irFunc, args)
}

// callFuncDecl calls a function implemented in GX.
func callFuncDecl(ctx Context, fn *state.Func, fnDecl *ir.FuncDecl, args []state.Element) (state.Element, error) {
	if fnDecl.Body == nil {
		return nil, ctx.FileSet().Errorf(fnDecl.Source(), "missing function body")
	}
	// Create a new function frame.
	funcFrame, err := ctx.frame().pushFuncFrame(fnDecl)
	if err != nil {
		return nil, err
	}
	defer ctx.frame().popFrame()
	for _, resultName := range fieldNames(fnDecl.FType.Results.List) {
		funcFrame.Define(resultName.Name, nil)
	}
	// Add the receiver name to the function frame if present.
	if recv := fn.Recv(); recv != nil {
		recvNode := recv.Element
		copyable, ok := recvNode.(state.Copyable)
		if ok {
			recvNode = copyable.Copy()
		}
		funcFrame.Define(recv.Ident.Name, recvNode)
	}
	assignArgumentValues(fnDecl.FType, funcFrame, args)
	// Evaluate the function within the frame.
	return evalFuncBody(ctx, fnDecl.Body)
}

func evalFuncBody(ctx Context, body *ir.BlockStmt) (state.Element, error) {
	element, stop, err := evalBlockStmt(ctx, body)
	if !stop {
		// No return statement was processed during the eval of the function.
		return nil, ctx.FileSet().Errorf(body.Src, "missing return")
	}
	return element, err
}

func fieldNames(fields []*ir.FieldGroup) (r []*ast.Ident) {
	for _, arg := range fields {
		for _, name := range arg.Src.Names {
			r = append(r, name)
		}
	}
	return
}

func assignArgumentValues(funcType *ir.FuncType, funcFrame *frame, args []state.Element) {
	// For each parameter of the function, assign its argument value to the frame.
	names := fieldNames(funcType.Params.List)
	for i, arg := range args {
		copyable, ok := arg.(state.Copyable)
		if ok {
			arg = copyable.Copy()
		}
		funcFrame.Define(names[i].Name, arg)
	}
}

func evalCallExpr(ctx Context, expr *ir.CallExpr) (state.Element, error) {
	// Fetch the function and check that it is callable.
	fnNode, err := evalExpr(ctx, expr.Func)
	if err != nil {
		return nil, err
	}
	fn, ok := fnNode.(*state.Func)
	if !ok {
		return nil, ctx.FileSet().Errorf(expr.Source(), "%T is not callable", fnNode)
	}

	// Evaluate the arguments to pass to the function.
	args := make([]state.Element, len(expr.Args))
	for i, arg := range expr.Args {
		el, err := evalExpr(ctx, arg)
		if err != nil {
			return nil, err
		}
		args[i] = el
	}
	return callFunc(ctx, expr, fn, args)
}
