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

	"github.com/pkg/errors"
	"github.com/gx-org/gx/api/values"
	"github.com/gx-org/gx/build/ir"
)

// Methods maintains a table of methods with a receiver.
type Methods struct {
	recv Element
}

var (
	_ Element        = (*Methods)(nil)
	_ FieldSelector  = (*Methods)(nil)
	_ MethodSelector = (*Methods)(nil)
)

// NewMethods returns a new node representing a table of methods.
func NewMethods(recv Element) *Methods {
	return &Methods{recv: recv}
}

func receiverIdent(f ir.Func) *ast.Ident {
	funcDecl, ok := f.(*ir.FuncDecl)
	if !ok {
		return nil
	}
	return funcDecl.Src.Recv.List[0].Names[0]
}

// Flatten returns the elements unpacked from the element supporting the methods.
func (n *Methods) Flatten() ([]Element, error) {
	return n.recv.Flatten()
}

// SelectMethod returns the method of a type given its IR.
func (n *Methods) SelectMethod(fn ir.Func) (*Func, error) {
	recv := &Receiver{
		Ident:   receiverIdent(fn),
		Element: n.recv,
	}
	return NewFunc(fn, recv), nil
}

// SelectField returns the field given an index.
// Returns nil if the receiver type cannot select fields.
func (n *Methods) SelectField(expr ExprAt, name string) (Element, error) {
	st, ok := n.recv.(FieldSelector)
	if !ok {
		return nil, errors.Errorf("%T cannot be converted to %T", n.recv, st)
	}
	return st.SelectField(expr, name)
}

// Unflatten consumes the next handles to return a GX value.
func (n *Methods) Unflatten(handles *Unflattener) (values.Value, error) {
	return handles.Unflatten(n.recv)
}
