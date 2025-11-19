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

package cpevelements

import (
	"go/ast"

	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/interp/elements"
)

// CoreMacroElement is a helper structure to implement macros.
type CoreMacroElement struct {
	mac  *ir.Macro
	call elements.CallAt
}

var _ ir.MacroElement = (*CoreMacroElement)(nil)

// MacroElement returns a core macro element for custom elements.
func MacroElement(mac *ir.Macro, file *ir.File, call *ir.FuncCallExpr) CoreMacroElement {
	return CoreMacroElement{
		mac:  mac,
		call: elements.NewNodeAt(file, call),
	}
}

// Type returns the type of a macro function.
func (CoreMacroElement) Type() ir.Type {
	return ir.UnknownType()
}

// From returns the macro function that has generated this macro element.
func (m *CoreMacroElement) From() *ir.Macro {
	return m.mac
}

// Call returns the source call from where the element was created.
func (m *CoreMacroElement) Call() elements.CallAt {
	return m.call
}

// Source returns the source call from where the element was created.
func (m *CoreMacroElement) Source() ast.Node {
	return m.call.Source()
}
