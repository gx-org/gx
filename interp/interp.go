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

// Package interp evaluates GX code given an evaluator.
//
// All values in the interpreter are represented in elements.
// The GX Context evaluates GX code represented as an
// intermediate representation (IR) tree
// (see [github.com/gx-org/gx/build/ir]),
// evaluates a function given a receiver and arguments passed as interpreter elements.
package interp

import (
	"strings"

	"github.com/pkg/errors"
	"github.com/gx-org/gx/api/options"
	"github.com/gx-org/gx/api/values"
	"github.com/gx-org/gx/build/fmterr"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/interp/elements"
	"github.com/gx-org/gx/interp/evaluator"
)

// FuncBuiltin of a builtin function by a backend.
type FuncBuiltin func(ctx evaluator.Context, call elements.CallAt, fn elements.Func, irFunc *ir.FuncBuiltin, args []elements.Element) ([]elements.Element, error)

// EvalFunctionToElement evaluates a function such as it becomes an element.
func (ctx *Context) EvalFunctionToElement(eval evaluator.Evaluator, fn ir.Func, args []elements.Element) ([]elements.Element, error) {
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

// EvalExprInContext uses an evaluator to evaluation an IR expression.
func EvalExprInContext(ctx evaluator.Context, expr ir.Expr) (elements.Element, error) {
	return ctx.(*Context).evalExpr(expr)
}

// EvalFunc evaluates a function.
func EvalFunc(eval evaluator.Evaluator, fn *ir.FuncDecl, in *elements.InputElements, options []options.PackageOption) (outs []elements.Element, err error) {
	if fn.Body == nil {
		return nil, errors.Errorf("%s: missing function body", fn.Name())
	}
	ectx, err := NewInterpContext(eval, options)
	if err != nil {
		return nil, err
	}
	ctx, err := ectx.newFileContext(fn.File())
	if err != nil {
		return nil, err
	}
	// Set the function inputs in the Context.
	ctx.callInputs = in
	// Create a frame for the function to evaluate.
	frame, err := ctx.pushFuncFrame(fn)
	if err != nil {
		return nil, err
	}
	defer ctx.popFrame()
	// Add the result names to the Context.
	for _, resultName := range fieldNames(fn.FType.Results.List) {
		frame.Define(resultName.Name, nil)
	}
	// Add the receiver to the Context.
	recv := fn.FType.ReceiverField()
	if recv != nil {
		if in.Receiver == nil {
			return nil, errors.Errorf("function has a receiver but a nil value has been passed as a receiver value")
		}
		frame.Define(recv.Name.Name, in.Receiver)
	}
	// Add the parameters to the Context.
	paramFields := fn.FType.Params.Fields()
	for i, param := range paramFields {
		if i >= len(in.Args) {
			missingParams := paramFields[len(in.Args):]
			builder := strings.Builder{}
			for n, param := range missingParams {
				if n > 0 {
					builder.WriteString(", ")
				}
				builder.WriteString(param.Name.String())
			}
			return nil, errors.Errorf("missing parameter(s): %s", builder.String())
		}
		frame.Define(param.Name.Name, in.Args[i])
	}
	// Evaluate the function body.
	outs, err = evalFuncBody(ctx, fn.Body)
	return
}

func dimsAsElements(ctx *Context, expr ir.AssignableExpr, dims []int) ([]elements.NumericalElement, error) {
	els := make([]elements.NumericalElement, len(dims))
	for i, di := range dims {
		val, err := values.AtomIntegerValue[int64](ir.IntLenType(), int64(di))
		if err != nil {
			return nil, err
		}
		els[i], err = ctx.evaluator.ElementFromAtom(elements.NewExprAt(ctx.File(), expr), val)
		if err != nil {
			return nil, err
		}
	}
	return els, nil
}

func rankOf(ctx evaluator.Context, src ir.SourceNode, typ ir.ArrayType) (ir.ArrayRank, error) {
	switch rank := typ.Rank().(type) {
	case *ir.Rank:
		return rank, nil
	case *ir.RankInfer:
		if rank.Rnk == nil {
			return nil, fmterr.Errorf(ctx.File().FileSet(), src.Source(), "array rank has not been resolved")
		}
		return rank.Rnk, nil
	default:
		return nil, fmterr.Errorf(ctx.File().FileSet(), src.Source(), "rank %T not supported", rank)
	}
}

// ToSingleElement packs multiple elements into a tuple.
// If the slice els contains only one element, this element is returned.
func ToSingleElement(ctx ir.Evaluator, node ir.SourceNode, els []elements.Element) (elements.Element, error) {
	switch len(els) {
	case 0:
		return nil, nil
	case 1:
		return els[0], nil
	default:
		return elements.NewTuple(ctx.File(), node, els), nil
	}

}
