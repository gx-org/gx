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
	"github.com/gx-org/gx/build/ir"
)

// Struct stores the GX values of a structure.
type Struct struct {
	vals       []Value
	typ        ir.Type
	structType *ir.StructType
}

var _ Value = (*Struct)(nil)

// NewStruct returns a new structure given a set of values.
func NewStruct(typ ir.Type, vals []Value) (*Struct, error) {
	vs := &Struct{
		typ:  typ,
		vals: vals,
	}
	under := ir.Underlying(typ)
	var ok bool
	vs.structType, ok = under.(*ir.StructType)
	if !ok {
		return nil, errors.Errorf("cannot create a structure value for type %T", under)
	}
	if vs.vals == nil {
		vs.vals = make([]Value, vs.structType.NumFields())
	}
	return vs, nil
}

func (*Struct) value() {}

// ToHost transfers all the elements of the slice to the host.
func (vs *Struct) ToHost(alloc platform.Allocator) (Value, error) {
	vals, err := ToHost(alloc, vs.vals)
	if err != nil {
		return nil, err
	}
	return NewStruct(vs.Type(), vals)
}

// Type of the structure.
func (vs *Struct) Type() ir.Type {
	return vs.typ
}

// SetField sets a field value.
func (vs *Struct) SetField(i int, val Value) {
	vs.vals[i] = val
}

// FieldValue returns the value of the ith field.
func (vs *Struct) FieldValue(i int) Value {
	return vs.vals[i]
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

// String representation of the structure.
func (vs *Struct) String() string {
	fields := vs.structType.Fields.Fields()
	fieldStrs := make([]string, len(fields))
	for i, field := range fields {
		childS := fmt.Sprint(vs.vals[i])
		if strings.Index(childS, "\n") > 0 {
			childS = indent(childS)
		}
		fieldStrs[i] = fmt.Sprintf("\t%s: %s,\n", field.Name.Name, childS)
	}
	fieldsStr := "\n" + strings.Join(fieldStrs, "")
	return fmt.Sprintf("%s{%s}", vs.Type().String(), fieldsStr)
}
