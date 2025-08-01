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
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/interp/elements"
	"github.com/gx-org/gx/interp"
)

// MacroImpl is a builtin opaque function to produce an IR.
type MacroImpl func(call elements.CallAt, fn *Macro, args []ir.Element) (*SyntheticFunc, error)

// Macro is a macro function to build synthetic functions.
type Macro struct {
	macro *ir.Macro
	recv  *interp.Receiver
}

// NewMacro creates a new macro given its definition and a receiver.
func NewMacro(fn *ir.Macro, recv *interp.Receiver) interp.Func {
	return &Macro{macro: fn, recv: recv}
}

// Func returns the macro function.
func (f *Macro) Func() ir.Func {
	return f.macro
}

// Recv returns the receiver of the macro function.
func (f *Macro) Recv() *interp.Receiver {
	return f.recv
}

// Call the macro to build the synthetic element.
func (f *Macro) Call(fctx *interp.FileScope, call *ir.CallExpr, args []ir.Element) ([]ir.Element, error) {
	if f.macro.BuildSynthetic == nil {
		return nil, errors.Errorf("macro %s.%s has no implementation to build the synthetic function type", f.macro.FFile.Package.Name.Name, f.macro.Name())
	}
	buildSynthetic, ok := f.macro.BuildSynthetic.(MacroImpl)
	if !ok {
		return nil, errors.Errorf("%T cannot converted to %s", f.macro.BuildSynthetic, reflect.TypeFor[MacroImpl]())
	}
	el, err := buildSynthetic(elements.NewNodeAt(fctx.File(), call), f, args)
	return []ir.Element{el}, err
}

// Type returns the type of the function.
func (f *Macro) Type() ir.Type {
	return f.macro.Type()
}
