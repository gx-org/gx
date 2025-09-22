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

	"github.com/pkg/errors"
	"github.com/gx-org/gx/api/values"
	gxfmt "github.com/gx-org/gx/base/fmt"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/interp/flatten"
)

// Struct is an instance of a structure.
type Struct struct {
	fields map[string]ir.Element
	typ    *ir.StructType
}

var (
	_ Copier   = (*Struct)(nil)
	_ Selector = (*Struct)(nil)
)

// NewStructFromElements returns a new node representing a structure instance given a slice of
func NewStructFromElements(typ *ir.StructType, vals []ir.Element) *Struct {
	fields := make(map[string]ir.Element, len(vals))
	for i, field := range typ.Fields.Fields() {
		fields[field.Name.Name] = vals[i]
	}
	return NewStruct(typ, fields)
}

// NewStruct returns a new node representing a structure instance.
func NewStruct(typ *ir.StructType, fields map[string]ir.Element) *Struct {
	return &Struct{
		fields: fields,
		typ:    typ,
	}
}

// StructType returns the type of the structure.
func (n *Struct) StructType() *ir.StructType {
	return n.typ
}

func (n *Struct) orderedFieldValues() []ir.Element {
	fields := n.typ.Fields.Fields()
	ordered := make([]ir.Element, len(fields))
	for i, field := range fields {
		ordered[i] = n.fields[field.Name.Name]
	}
	return ordered
}

// Flatten returns a flat list of all the elements stored in the structure.
func (n *Struct) Flatten() ([]ir.Element, error) {
	return flatten.Flatten(n.orderedFieldValues()...)
}

// Unflatten consumes the next handles to return a GX value.
func (n *Struct) Unflatten(handles *flatten.Parser) (values.Value, error) {
	elts := n.orderedFieldValues()
	return handles.ParseComposite(flatten.ParseCompositeOf(values.NewStruct), n.typ, elts)
}

// Select returns the value of a field of a structure given its index.
func (n *Struct) Select(expr *ir.SelectorExpr) (ir.Element, error) {
	name := expr.Stor.NameDef().Name
	val, ok := n.fields[name]
	if !ok {
		return nil, errors.Errorf("field %s undefined", name)
	}
	return val, nil
}

// Type of the element.
func (n *Struct) Type() ir.Type {
	return n.typ
}

// Copy the structure to a new node.
func (n *Struct) Copy() Copier {
	cp := &Struct{
		typ: n.typ,
	}
	cp.fields = make(map[string]ir.Element, len(n.fields))
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
func (n *Struct) SetField(name string, value ir.Element) {
	n.fields[name] = value
}

func (n *Struct) String() string {
	var b strings.Builder
	b.WriteString(n.StructType().String())
	b.WriteString("{\n")
	for i, fld := range n.typ.Fields.Fields() {
		b.WriteString(gxfmt.Indent(fmt.Sprintf("%d: %s %s = %v\n", i, fld.Name.Name, fld.Type().String(), gxfmt.String(n.fields[fld.Name.Name]))))
	}
	b.WriteString("}")
	return b.String()
}
