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
	"github.com/pkg/errors"
	"github.com/gx-org/gx/api/hostio"
	"github.com/gx-org/gx/build/fmterr"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/interp/context"
	"github.com/gx-org/gx/interp/engine"
	"github.com/gx-org/gx/src/gx/ast"
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

func assignTypeParameters(ctx *context.Context, callee ir.Callee, funcFrame *context.Frame, genArgs []ir.Element) {
	funRef, ok := callee.(*ir.FuncValExpr)
	if !ok {
		return
	}
	genVals := funRef.FuncType().GenParams.Values
	for i, genVal := range genVals {
		funcFrame.Define(genVal.Generic().NameDef(), genArgs[i])
	}
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

type defineFuncInputsInFrame struct {
	conv  hostio.InputConverter
	frame *context.Frame
}

func (d *defineFuncInputsInFrame) ElementFromReceiverValue(ev ir.Evaluator, field *ir.Field, val hostio.Value) (ir.Element, error) {
	el, err := d.conv.ElementFromReceiverValue(ev, field, val)
	if err != nil {
		return nil, err
	}
	d.frame.Define(field.Name, engine.Copy(el))
	return el, nil
}

func (d *defineFuncInputsInFrame) ElementFromArgValue(ev ir.Evaluator, field *ir.Field, index int, val hostio.Value) (ir.Element, error) {
	el, err := d.conv.ElementFromArgValue(ev, field, index, val)
	if err != nil {
		return nil, err
	}
	d.frame.Define(field.Name, engine.Copy(el))
	return el, nil
}

// EvalFunc evaluates a function.
func (itp *Base) EvalFunc(conv hostio.InputConverter, fn *ir.FuncDecl, receiver hostio.Value, args []hostio.Value) (outs []ir.Element, err error) {
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
	if fn.FType.ReceiverField() != nil && receiver == nil {
		return nil, errors.Errorf("function has a receiver but a nil value has been passed as a receiver value")
	}
	if _, err := hostio.FuncInputsToElements(&defineFuncInputsInFrame{conv: conv, frame: frame}, fitp, fn, receiver, args); err != nil {
		return nil, err
	}
	// Evaluate the function body.
	outs, err = evalFuncBody(fitp, fn.Body)
	return
}
