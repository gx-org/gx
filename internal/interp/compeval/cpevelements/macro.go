// Copyright 2025 Google LLC
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

package cpevelements

import (
	"reflect"

	"github.com/pkg/errors"
	"github.com/gx-org/gx/api/values"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/interp/elements"
)

// MacroImpl is a builtin opaque function to produce an IR.
type MacroImpl func(call elements.CallAt, fn *Macro, args []elements.Element) (*SyntheticFunc, error)

// Macro is a macro function to build synthetic functions.
type Macro struct {
	macro *ir.Macro
	recv  *elements.Receiver
}

// NewMacro creates a new macro given its definition and a receiver.
func NewMacro(fn *ir.Macro, recv *elements.Receiver) elements.Func {
	return &Macro{macro: fn, recv: recv}
}

// Func returns the macro function.
func (f *Macro) Func() ir.Func {
	return f.macro
}

// Recv returns the receiver of the macro function.
func (f *Macro) Recv() *elements.Receiver {
	return f.recv
}

// Call the macro to build the synthetic element.
func (f *Macro) Call(fctx ir.Evaluator, call *ir.CallExpr, args []elements.Element) ([]elements.Element, error) {
	if f.macro.BuildSynthetic == nil {
		return nil, errors.Errorf("macro %s.%s has no implementation to build the synthetic function type", f.macro.FFile.Package.Name.Name, f.macro.Name())
	}
	buildSynthetic, ok := f.macro.BuildSynthetic.(MacroImpl)
	if !ok {
		return nil, errors.Errorf("%T cannot converted to %s", f.macro.BuildSynthetic, reflect.TypeFor[MacroImpl]())
	}
	el, err := buildSynthetic(elements.NewNodeAt(fctx.File(), call), f, args)
	return []elements.Element{el}, err
}

// Flatten the macro.
func (f *Macro) Flatten() ([]elements.Element, error) {
	return []elements.Element{f}, nil
}

// Unflatten the macro.
func (f *Macro) Unflatten(handles *elements.Unflattener) (values.Value, error) {
	return nil, errors.Errorf("not implemented")
}

// Kind returns the function kind.
func (f *Macro) Kind() ir.Kind {
	return ir.FuncKind
}
