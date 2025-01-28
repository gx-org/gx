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
	"fmt"
	"go/ast"

	"github.com/gx-org/gx/api/values"
	"github.com/gx-org/gx/build/ir"
)

type (
	// Func is an instance of a structure.
	Func struct {
		state *State
		fn    ir.Func
		recv  *Receiver
	}

	// Receiver of a function.
	Receiver struct {
		Ident   *ast.Ident
		Element Element
	}
)

var _ Element = (*Func)(nil)

// Func returns a new node representing a structure instance.
func (g *State) Func(fn ir.Func, recv *Receiver) *Func {
	return &Func{state: g, fn: fn, recv: recv}
}

// Type of the function.
func (st *Func) Type() ir.Type {
	return st.fn.Type()
}

// Flatten returns the element in a slice.
func (st *Func) Flatten() ([]Element, error) {
	return []Element{st}, nil
}

// Func returns the function represented by the node.
func (st *Func) Func() ir.Func {
	return st.fn
}

// Recv returns the receiver of the function or nil if the function has no receiver.
func (st *Func) Recv() *Receiver {
	return st.recv
}

// State owning the element.
func (st *Func) State() *State {
	return st.state
}

func (st *Func) valueFromHandle(handles *handleParser) (values.Value, error) {
	return values.NewIRNode(st.fn)
}

// String representation of the node.
func (st *Func) String() string {
	return fmt.Sprintf("func(%s)", st.Func().Name())
}
