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

// FuncInputsToElements converts the receiver and the arguments of a function call to elements.
func FuncInputsToElements(conv InputConverter, ev ir.Evaluator, fn ir.Func, receiver Value, args []Value) (*InputElements, error) {
	fType := fn.FuncType()
	var recvEl ir.Element
	if receiver != nil {
		recvField := fType.ReceiverField()
		var err error
		if recvEl, err = conv.ElementFromReceiverValue(ev, recvField, receiver); err != nil {
			return nil, err
		}
	}
	params := fType.AllParams()
	if err := checkMissingParams(fn, args, params); err != nil {
		return nil, err
	}
	argsEl := make([]ir.Element, len(params))
	for i, param := range params {
		var err error
		argsEl[i], err = conv.ElementFromArgValue(ev, param, i, args[i])
		if err != nil {
			return nil, err
		}
	}
	return &InputElements{
		Receiver: recvEl,
		Args:     argsEl,
	}, nil
}
