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
	"fmt"
	"strings"

	"github.com/gx-org/gx/api/values"
	gxfmt "github.com/gx-org/gx/base/fmt"
	"github.com/gx-org/gx/build/fmterr"
	"github.com/gx-org/gx/build/ir"
)

// Struct is an instance of a structure.
type Struct struct {
	expr       ValueAt
	fields     map[string]Element
	structType *ir.StructType
}

var (
	_ Copier   = (*Struct)(nil)
	_ Selector = (*Struct)(nil)
)

// NewStructFromElements returns a new node representing a structure instance given a slice of
func NewStructFromElements(structType *ir.StructType, expr ValueAt, vals []Element) *Struct {
	fields := make(map[string]Element, len(vals))
	for i, field := range structType.Fields.Fields() {
		fields[field.Name.Name] = vals[i]
	}
	return NewStruct(structType, expr, fields)
}

// NewStruct returns a new node representing a structure instance.
func NewStruct(structType *ir.StructType, expr ValueAt, fields map[string]Element) *Struct {
	return &Struct{
		expr:       expr,
		fields:     fields,
		structType: structType,
	}
}

// StructType returns the type of the structure.
func (n *Struct) StructType() *ir.StructType {
	return n.structType
}

func (n *Struct) orderedFieldValues() []Element {
	fields := n.structType.Fields.Fields()
	ordered := make([]Element, len(fields))
	for i, field := range fields {
		ordered[i] = n.fields[field.Name.Name]
	}
	return ordered
}

// Flatten returns a flat list of all the elements stored in the structure.
func (n *Struct) Flatten() ([]Element, error) {
	return Flatten(n.orderedFieldValues()...)
}

// Unflatten consumes the next handles to return a GX value.
func (n *Struct) Unflatten(handles *Unflattener) (values.Value, error) {
	elts := n.orderedFieldValues()
	return handles.ParseComposite(ParseCompositeOf(values.NewStruct), n.expr.Node().Type(), elts)
}

// Select returns the value of a field of a structure given its index.
func (n *Struct) Select(expr SelectAt) (Element, error) {
	name := expr.Node().Stor.NameDef().Name
	val, ok := n.fields[name]
	if !ok {
		return nil, fmterr.Errorf(expr.FSet(), expr.Node().Source(), "field %s undefined", name)
	}
	return val, nil
}

// Type of the element.
func (n *Struct) Type() ir.Type {
	return n.structType
}

// Copy the structure to a new node.
func (n *Struct) Copy() Copier {
	cp := &Struct{
		structType: n.structType,
		expr:       n.expr,
	}
	cp.fields = make(map[string]Element, len(n.fields))
	for name, field := range n.fields {
		copyable, ok := field.(Copier)
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
	b.WriteString("{\n")
	for i, fld := range n.structType.Fields.Fields() {
		b.WriteString(gxfmt.Indent(fmt.Sprintf("%d: %s %s = %v\n", i, fld.Name.Name, fld.Type().String(), gxfmt.String(n.fields[fld.Name.Name]))))
	}
	b.WriteString("}")
	return b.String()
}
