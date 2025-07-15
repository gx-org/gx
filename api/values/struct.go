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

package values

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"github.com/gx-org/backend/platform"
	"github.com/gx-org/gx/base/sync"
	"github.com/gx-org/gx/build/ir"
)

// Struct stores the GX values of a structure.
type Struct struct {
	vals       sync.Map[string, Value]
	typ        ir.Type
	structType *ir.StructType
}

var _ Value = (*Struct)(nil)

// NewStruct returns a new structure given a set of values.
func NewStruct(typ ir.Type, vals []Value) (*Struct, error) {
	under := ir.Underlying(typ)
	var ok bool
	structType, ok := under.(*ir.StructType)
	if !ok {
		return nil, errors.Errorf("cannot create a structure value for type %T", under)
	}
	vs := &Struct{
		typ:        typ,
		structType: structType,
	}
	if len(vals) == 0 {
		return vs, nil
	}
	fields := structType.Fields.Fields()
	if len(vals) != len(fields) {
		return nil, errors.Errorf("incorrect number of values: got %d values to initialize %d fields", len(vals), len(fields))
	}
	for i, field := range fields {
		vs.SetField(field.Name.Name, vals[i])
	}
	return vs, nil
}

func (*Struct) value() {}

// ToHost transfers all the elements of the slice to the host.
func (vs *Struct) ToHost(alloc platform.Allocator) (Value, error) {
	vals := make([]Value, vs.structType.NumFields())
	for i, field := range vs.structType.Fields.Fields() {
		v := vs.FieldValue(field.Name.Name)
		vHost, err := v.ToHost(alloc)
		if err != nil {
			return nil, err
		}
		vals[i] = vHost
	}
	return NewStruct(vs.Type(), vals)
}

// Type of the structure.
func (vs *Struct) Type() ir.Type {
	return vs.typ
}

// SetField sets a field value.
func (vs *Struct) SetField(name string, val Value) {
	vs.vals.Store(name, val)
}

// Select a field in the structure.
func (vs *Struct) Select(expr *ir.SelectorExpr) (ir.Element, error) {
	return vs.FieldValue(expr.Src.Sel.Name), nil
}

// FieldValue returns the value of the ith field.
func (vs *Struct) FieldValue(name string) Value {
	return vs.vals.Load(name)
}

// StructType returns the structure type of the structure.
func (vs *Struct) StructType() *ir.StructType {
	return vs.structType
}

func indent(s string) string {
	ss := strings.Split(s, "\n")
	for i, s := range ss {
		if i == 0 {
			continue
		}
		ss[i] = "\t" + s
	}
	return strings.Join(ss, "\n")
}

func (vs *Struct) toString(prefix string) string {
	fields := vs.structType.Fields.Fields()
	fieldStrs := make([]string, len(fields))
	for i, field := range fields {
		childS := fmt.Sprint(vs.FieldValue(field.Name.Name))
		if strings.Index(childS, "\n") > 0 {
			childS = indent(childS)
		}
		fieldStrs[i] = fmt.Sprintf("\t%s: %s,\n", field.Name.Name, childS)
	}
	fieldsStr := "\n" + strings.Join(fieldStrs, "")
	return fmt.Sprintf("%s{%s}", prefix, fieldsStr)
}

// String representation of the structure.
func (vs *Struct) String() string {
	return vs.toString("struct")
}
