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
	"strings"

	"github.com/gx-org/gx/api/values"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/interp/elements"
)

// Struct is an instance of a structure.
type Struct struct {
	state      *State
	expr       elements.ExprAt
	fields     map[string]Element
	fieldInit  FieldSelector
	structType *ir.StructType
}

var (
	_ Element         = (*Struct)(nil)
	_ MethodSelector  = (*Struct)(nil)
	_ FieldSelector   = (*Struct)(nil)
	_ Copyable        = (*Struct)(nil)
	_ handleProcessor = (*Struct)(nil)
)

// StructWithInit returns a new node representing a structure instance where the field
// will be constructed only on demand.
// This is used for structures passed as arguments to GX.
func (g *State) StructWithInit(structType *ir.StructType, expr elements.ExprAt, fieldInit FieldSelector) *Struct {
	return &Struct{
		state:      g,
		expr:       expr,
		fieldInit:  fieldInit,
		structType: structType,
		fields:     make(map[string]Element, structType.NumFields()),
	}
}

// StructFromElements returns a new node representing a structure instance given a slice of elements.
func (g *State) StructFromElements(structType *ir.StructType, expr elements.ExprAt, vals []Element) *Struct {
	fields := make(map[string]Element, len(vals))
	for i, field := range structType.Fields.Fields() {
		fields[field.Name.Name] = vals[i]
	}
	return g.Struct(structType, expr, fields)
}

// Struct returns a new node representing a structure instance.
func (g *State) Struct(structType *ir.StructType, expr elements.ExprAt, fields map[string]Element) *Struct {
	return &Struct{
		expr:       expr,
		state:      g,
		fields:     fields,
		structType: structType,
	}
}

// StructType returns the type of the structure.
func (n *Struct) StructType() *ir.StructType {
	return n.structType
}

// State owning the element.
func (n *Struct) State() *State {
	return n.state
}

func (n *Struct) orderedFieldValues() ([]Element, error) {
	fields := n.structType.Fields.Fields()
	ordered := make([]Element, len(fields))
	for i, field := range fields {
		name := field.Name.Name
		fieldVal := n.fields[name]
		var err error
		if fieldVal == nil {
			// If a field has not been assigned before,
			// then forces a select now so that the missing
			// value is created.
			fieldVal, err = n.SelectField(n.expr, field.Name.Name)
			if err != nil {
				return nil, err
			}
		}
		ordered[i] = fieldVal
	}
	return ordered, nil
}

// Flatten returns a flat list of all the elements stored in the structure.
func (n *Struct) Flatten() ([]Element, error) {
	elts, err := n.orderedFieldValues()
	if err != nil {
		return nil, err
	}
	return flattenAll(elts)
}

func (n *Struct) valueFromHandle(handles *handleParser) (values.Value, error) {
	elts, err := n.orderedFieldValues()
	if err != nil {
		return nil, err
	}
	return handles.parseComposite(parseCompositeOf(values.NewStruct), n.expr.Node().Type(), elts)
}

// SelectField returns the value of a field of a structure given its index.
func (n *Struct) SelectField(expr elements.ExprAt, name string) (Element, error) {
	field := n.fields[name]
	if field != nil {
		return field, nil
	}
	var err error
	field, err = n.fieldInit.SelectField(expr, name)
	if err != nil {
		return nil, err
	}
	n.fields[name] = field
	return field, nil
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
	cp.fields = make(map[string]Element, len(n.fields))
	for name, field := range n.fields {
		copyable, ok := field.(Copyable)
		if ok {
			field = copyable.Copy()
		}
		cp.fields[name] = field
	}
	return cp
}

// SetField sets the field in the structure.
func (n *Struct) SetField(name string, value Element) {
	n.fields[name] = value
}

func (n *Struct) String() string {
	var b strings.Builder
	b.WriteString(n.StructType().String())
	b.WriteString(":\n")
	for i, fld := range n.structType.Fields.Fields() {
		b.WriteString(fmt.Sprintf(" Field %d (%s): %v\n", i, fld.Name.Name, n.fields[fld.Name.Name]))
	}
	return b.String()
}
