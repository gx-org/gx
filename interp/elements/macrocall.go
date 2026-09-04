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

package elements

import (
	"go/ast"

	"github.com/gx-org/gx/build/ir"
)

// MacroCall is a helper structure to implement macros.
type MacroCall struct {
	mac  *ir.Macro
	call CallAt
}

var _ ir.MacroElement = (*MacroCall)(nil)

// NewMacroCall returns a core macro element for custom elements.
func NewMacroCall(mac *ir.Macro, file *ir.File, call *ir.FuncCallExpr) MacroCall {
	return MacroCall{
		mac:  mac,
		call: NewNodeAt(file, call),
	}
}

// Type returns the type of a macro function.
func (MacroCall) Type() ir.Type {
	return ir.UnknownType()
}

// From returns the macro function that has generated this macro element.
func (m *MacroCall) From() *ir.Macro {
	return m.mac
}

// Call returns the source call from where the element was created.
func (m *MacroCall) Call() CallAt {
	return m.call
}

// Source returns the source call from where the element was created.
func (m *MacroCall) Source() ast.Node {
	return m.call.Source()
}
