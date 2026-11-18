// Copyright 2026 Google LLC
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

package hostio

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"github.com/gx-org/gx/build/ir"
)

// InputElements is the receiver and arguments with which the function was called.
type InputElements struct {
	// Receiver on which the function call was done.
	// Can be nil.
	Receiver ir.Element

	// Args returns list of arguments passed to the interpreter at call time.
	Args []ir.Element
}

// InputConverter converts function call inputs from the host into interpreter elements.
type InputConverter interface {
	ElementFromReceiverValue(ev ir.Evaluator, field *ir.Field, val Value) (ir.Element, error)
	ElementFromArgValue(ev ir.Evaluator, field *ir.Field, index int, val Value) (ir.Element, error)
}

func checkMissingParams(fn ir.Func, got []Value, want []*ir.Field) error {
	if len(got) == len(want) {
		return nil
	}
	if len(got) > len(want) {
		return errors.Errorf("too many arguments in call to %s: have %d but want %d", fn.FuncType().ReferString(nil), len(got), len(want))
	}
	missingParams := want[len(got):]
	builder := strings.Builder{}
	for n, param := range missingParams {
		if n > 0 {
			builder.WriteString(", ")
		}
		builder.WriteString(param.Name.String())
	}
	return errors.Errorf("missing parameter(s): %s", builder.String())
}

func isStructType(typ ir.Type) bool {
	_, ok := ir.Underlying(typ).(*ir.StructType)
	return ok
}

// InputTypeError returns an error when an input value does not match the expected type.
func InputTypeError(name string, want ir.Type, got ir.Element) error {
	gotDesc := fmt.Sprintf("%T", got)
	if got != nil && got.Type() != nil && !ir.IsUnknown(got.Type()) {
		gotDesc = got.Type().ReferString(nil)
	}
	if name == "" {
		return errors.Errorf("cannot use %s as %s", gotDesc, want.ReferString(nil))
	}
	return errors.Errorf("cannot use %s as %s for argument %s", gotDesc, want.ReferString(nil), name)
}

func fieldName(field *ir.Field) string {
	if field == nil || field.Name == nil {
		return ""
	}
	return field.Name.Name
}

func checkType(ev ir.TypeEval, field *ir.Field, el ir.Element) error {
	typ := field.Type()
	if el == nil || el.Type() == nil || (ir.IsUnknown(el.Type()) && isStructType(typ)) {
		return InputTypeError(fieldName(field), typ, el)
	}
	ok, err := ir.AssignableTo(ev, el.Type(), typ)
	if err != nil || ok {
		return err
	}
	return InputTypeError(fieldName(field), typ, el)
}

func convertArgument(conv InputConverter, ev ir.TypeEval, field *ir.Field, index int, val Value) (ir.Element, error) {
	if val == nil {
		return nil, InputTypeError(fieldName(field), field.Type(), nil)
	}
	el, err := conv.ElementFromArgValue(ev, field, index, val)
	if err != nil {
		return nil, err
	}
	if err := checkType(ev, field, el); err != nil {
		return nil, err
	}
	return el, nil
}

// FuncInputsToElements converts the receiver and the arguments of a function call to elements.
func FuncInputsToElements(conv InputConverter, ev ir.TypeEval, fn ir.Func, receiver Value, args []Value) (*InputElements, error) {
	fType := fn.FuncType()
	recvField := fType.ReceiverField()
	if recvField != nil && receiver == nil {
		return nil, errors.Errorf("function has a receiver but a nil value has been passed as a receiver value")
	}
	if recvField == nil && receiver != nil {
		return nil, errors.Errorf("function %s has no receiver but a receiver value was passed", fType.ReferString(nil))
	}
	params := fType.AllParams()
	if err := checkMissingParams(fn, args, params); err != nil {
		return nil, err
	}
	numGen := fType.GenParams.Fields.Len()
	argsEl := make([]ir.Element, len(params))
	for i := range numGen {
		var err error
		argsEl[i], err = convertArgument(conv, ev, params[i], i, args[i])
		if err != nil {
			return nil, err
		}
	}
	var recvEl ir.Element
	if recvField != nil {
		var err error
		if recvEl, err = conv.ElementFromReceiverValue(ev, recvField, receiver); err != nil {
			return nil, err
		}
		if err := checkType(ev, recvField, recvEl); err != nil {
			return nil, err
		}
	}
	for i := numGen; i < len(params); i++ {
		var err error
		argsEl[i], err = convertArgument(conv, ev, params[i], i, args[i])
		if err != nil {
			return nil, err
		}
	}
	return &InputElements{
		Receiver: recvEl,
		Args:     argsEl,
	}, nil
}
