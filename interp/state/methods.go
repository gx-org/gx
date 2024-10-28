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

package state

import (
	"go/ast"

	"github.com/pkg/errors"
	"github.com/gx-org/backend/graph"
	"github.com/gx-org/gx/api/values"
	"github.com/gx-org/gx/build/ir"
)

// Methods maintains a table of methods with a receiver.
type Methods struct {
	state *State
	recv  Element
}

var (
	_ FieldSelector  = (*Methods)(nil)
	_ MethodSelector = (*Methods)(nil)
	_ backendElement = (*Methods)(nil)
)

// Methods returns a new node representing a table of methods.
func (g *State) Methods(recv Element) *Methods {
	return &Methods{state: g, recv: recv}
}

func receiverIdent(f ir.Func) *ast.Ident {
	funcDecl, ok := f.(*ir.FuncDecl)
	if !ok {
		return nil
	}
	return funcDecl.Src.Recv.List[0].Names[0]
}

// SelectMethod returns the method of a type given its IR.
func (n *Methods) SelectMethod(fn ir.Func) (*Func, error) {
	recv := &Receiver{
		Ident:   receiverIdent(fn),
		Element: n.recv,
	}
	return n.state.Func(fn, recv), nil
}

func (n *Methods) nodes() ([]*graph.OutputNode, error) {
	return OutputsFromElement(n.recv)
}

// SelectField returns the field given an index.
// Returns nil if the receiver type cannot select fields.
func (n *Methods) SelectField(expr ExprAt, i int) (Element, error) {
	st, ok := n.recv.(FieldSelector)
	if !ok {
		return nil, errors.Errorf("%T cannot be converted to %T", n.recv, st)
	}
	return st.SelectField(expr, i)
}

// State owning the element.
func (n *Methods) State() *State {
	return n.state
}

func (n *Methods) valueFromHandle(handles *handleParser) (values.Value, error) {
	return handles.parse(n.recv)
}
