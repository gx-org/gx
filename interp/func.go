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
	"strings"

	"github.com/pkg/errors"
	"github.com/gx-org/gx/build/fmterr"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/interp/context"
	"github.com/gx-org/gx/interp/elements"
	"github.com/gx-org/gx/interp/engine"
)

func evalFuncBody(fitp *Interpreter, body *ir.BlockStmt) ([]ir.Element, error) {
	outs, _, err := evalBlockStmt(fitp, body)
	return outs, err
}

func fieldNames(fields []*ir.FieldGroup) (r []*ast.Ident) {
	for _, arg := range fields {
		for _, name := range arg.Src.Names {
			r = append(r, name)
		}
	}
	return
}

func toConcreteType(ctx *context.Context, src ast.Node, frame *context.Frame, tp ir.Type) (ir.Type, error) {
	typeParam, isTypeParam := tp.(*ir.GenericTypeParam)
	if !isTypeParam {
		return tp, nil
	}
	el, err := frame.Find(typeParam.OrigField().Name)
	if err != nil {
		return nil, fmterr.InternalAt(ctx.File().FileSet(), src, "cannot cast to %s: %v", tp.ReferString(ctx.File()), err)
	}
	tp, isType := ir.BareValue(el).(ir.Type)
	if !isType {
		return nil, fmterr.InternalAt(ctx.File().FileSet(), src, "element %T is not a type", el)
	}
	return tp, nil
}

func assignTypeParameters(ctx *context.Context, callee ir.Callee, funcFrame *context.Frame, args []ir.Element) []ir.Element {
	funRef, ok := callee.(*ir.FuncValExpr)
	if !ok {
		return args
	}
	genVals := funRef.FuncType().GenericValues
	for i, genVal := range genVals {
		funcFrame.Define(genVal.Generic().NameDef(), args[i])
	}
	return args[len(genVals):]
}

func assignArgumentValues(ftype *ir.FuncType, funcFrame *context.Frame, args []ir.Element) error {
	fields := ftype.Params.Fields()
	if len(args) != len(fields) {
		return fmterr.Internal(errors.Errorf("number of arguments (%d) does not match the number of parameters (%d) in function type %s", len(args), len(fields), ftype.ReferString(nil)))
	}
	// For each parameter of the function, assign its argument value to the frame.
	for i, arg := range args {
		arg = engine.Copy(arg)
		funcFrame.Define(fields[i].Name, arg)
	}
	return nil
}

// EvalFunc evaluates a function.
func (itp *Base) EvalFunc(fn *ir.FuncDecl, in *elements.InputElements) (outs []ir.Element, err error) {
	if fn.Body == nil {
		return nil, errors.Errorf("%s: missing function body", fn.Name())
	}
	fitp, err := itp.ForFile(fn.File())
	if err != nil {
		return nil, err
	}
	// Create a frame for the function to evaluate.
	frame, err := fitp.Context().PushFuncFrame(fn)
	if err != nil {
		return nil, err
	}
	defer fitp.Context().PopFrame()
	// Add the result names to the Context.
	if fn.FType.Results != nil {
		for _, resultName := range fieldNames(fn.FType.Results.List) {
			frame.Define(resultName, nil)
		}
	}
	// Add the receiver to the Context.
	recv := fn.FType.ReceiverField()
	if recv != nil {
		if in.Receiver == nil {
			return nil, errors.Errorf("function has a receiver but a nil value has been passed as a receiver value")
		}
		frame.Define(recv.Name, in.Receiver)
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
		frame.Define(param.Name, in.Args[i])
	}
	// Evaluate the function body.
	outs, err = evalFuncBody(fitp, fn.Body)
	return
}
