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
	"github.com/gx-org/backend/graph"
	"github.com/gx-org/gx/api/values"
	"github.com/gx-org/gx/build/ir"
)

// Struct is an instance of a structure.
type Struct struct {
	state      *State
	expr       ExprAt
	fields     []Element
	fieldInit  FieldSelector
	structType *ir.StructType
}

var (
	_ Element         = (*Struct)(nil)
	_ MethodSelector  = (*Struct)(nil)
	_ FieldSelector   = (*Struct)(nil)
	_ Copyable        = (*Struct)(nil)
	_ handleProcessor = (*Struct)(nil)
	_ backendElement  = (*Struct)(nil)
)

// StructWithInit returns a new node representing a structure instance where the field
// will be constructed only on demand.
// This is used for structures passed as arguments to GX.
func (g *State) StructWithInit(structType *ir.StructType, expr ExprAt, fieldInit FieldSelector) *Struct {
	return &Struct{
		state:      g,
		expr:       expr,
		fieldInit:  fieldInit,
		structType: structType,
		fields:     make([]Element, structType.NumFields()),
	}
}

// Struct returns a new node representing a structure instance.
func (g *State) Struct(structType *ir.StructType, expr ExprAt, fields []Element) *Struct {
	return &Struct{
		expr:       expr,
		state:      g,
		fields:     fields,
		structType: structType,
	}
}

// State owning the element.
func (n *Struct) State() *State {
	return n.state
}

func (n *Struct) nodes() ([]*graph.OutputNode, error) {
	for i, field := range n.fields {
		if field == nil {
			field := n.structType.Field(i)
			n.fields[i] = n.state.NewZero(field.Type())
		}
	}
	return OutputsFromElements(n.fields)
}

func (n *Struct) valueFromHandle(handles *handleParser) (values.Value, error) {
	return handles.parseComposite(parseCompositeOf(values.NewStruct), n.expr.Type(), n.fields)
}

// SelectField returns the value of a field of a structure given its index.
func (n *Struct) SelectField(expr ExprAt, id int) (Element, error) {
	field := n.fields[id]
	if field != nil {
		return field, nil
	}
	var err error
	n.fields[id], err = n.fieldInit.SelectField(expr, id)
	if err != nil {
		return nil, err
	}
	return n.fields[id], nil
}

// SelectMethod returns the method of a type given its IR.
func (n *Struct) SelectMethod(fn ir.Func) (*Func, error) {
	recv := &Receiver{
		Ident:   receiverIdent(fn),
		Element: n,
	}
	return n.state.Func(fn, recv), nil
}

// Copy the structure to a new node.
func (n *Struct) Copy() Element {
	cp := &Struct{
		state:      n.state,
		fieldInit:  n.fieldInit,
		structType: n.structType,
		expr:       n.expr,
	}
	cp.fields = make([]Element, len(n.fields))
	for i, field := range n.fields {
		copyable, ok := field.(Copyable)
		if ok {
			field = copyable.Copy()
		}
		cp.fields[i] = field
	}
	return cp
}

// SetField sets the field in the structure.
func (n *Struct) SetField(i int, value Element) {
	n.fields[i] = value
}
