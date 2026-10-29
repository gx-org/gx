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
	"github.com/gx-org/gx/api/hostio"
	gxfmt "github.com/gx-org/gx/base/fmt"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/golang/backend/kernels"
	"github.com/gx-org/gx/internal/base/cast"
	"github.com/gx-org/gx/internal/interp/flatten"
	"github.com/gx-org/gx/interp/engine"
	"github.com/gx-org/gx/src/gx/ast"
)

// Struct is an instance of a structure.
type Struct struct {
	fields map[string]ir.Element
	typ    *ir.StructType
}

var (
	_ IStruct         = (*Struct)(nil)
	_ engine.Copier   = (*Struct)(nil)
	_ engine.Selector = (*Struct)(nil)
	_ hostio.Value    = (*Struct)(nil)
)

// NewStructFromElements returns a new node representing a structure instance given a slice of
func NewStructFromElements(typ *ir.StructType, vals []ir.Element) (*Struct, error) {
	fields := make(map[string]ir.Element, len(vals))
	for i, field := range typ.Fields.Fields() {
		fields[field.Name.Name] = vals[i]
	}
	return NewStruct(typ, fields)
}

// NewStruct returns a new node representing a structure instance.
// fields can be nil when the fields are set later.
func NewStruct(typ *ir.StructType, fields map[string]ir.Element) (*Struct, error) {
	if fields == nil {
		fields = make(map[string]ir.Element, typ.NumFields())
	}
	return &Struct{
		fields: fields,
		typ:    typ,
	}, nil
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
func (n *Struct) Unflatten(parser *flatten.Parser) (hostio.Value, error) {
	elts := n.orderedFieldValues()
	return parser.ParseComposite(flatten.ParseCompositeOf(newStructFromValues), n.typ, elts)
}

// newStructFromValues returns a structure from the GX values of its fields.
func newStructFromValues(typ ir.Type, vals []hostio.Value) (*Struct, error) {
	structType, err := cast.To[*ir.StructType](typ)
	if err != nil {
		return nil, err
	}
	return NewStructFromElements(structType, hostio.ToElements(vals))
}

// gxValues returns the values of the fields of the structure as GX values.
// An error is returned if a field value is not a GX value.
func (n *Struct) gxValues() ([]hostio.Value, error) {
	elts := n.orderedFieldValues()
	vals := make([]hostio.Value, len(elts))
	for i, el := range elts {
		var err error
		if vals[i], err = cast.To[hostio.Value](el); err != nil {
			return nil, err
		}
	}
	return vals, nil
}

// ToHost transfers all the field values of the structure to the host.
func (n *Struct) ToHost(alloc kernels.Allocator) (hostio.Value, error) {
	vals, err := n.gxValues()
	if err != nil {
		return nil, err
	}
	hostVals, err := hostio.ToHost(alloc, vals)
	if err != nil {
		return nil, err
	}
	return NewStructFromElements(n.typ, hostio.ToElements(hostVals))
}

// Finalize releases the resources allocated for the field values of the structure.
// Fields without a value are skipped.
func (n *Struct) Finalize() error {
	for _, el := range n.orderedFieldValues() {
		if el == nil {
			continue
		}
		val, err := cast.To[hostio.Value](el)
		if err != nil {
			return err
		}
		if err := val.Finalize(); err != nil {
			return err
		}
	}
	return nil
}

// Select returns the value of a field of a structure given its index.
func (n *Struct) Select(env *engine.Env, expr *ir.SelectorExpr) (ir.Element, error) {
	name := expr.Stor.NameDef().Name
	val, ok := n.fields[name]
	if !ok {
		return nil, errors.Errorf("field %s has no value", name)
	}
	return val, nil
}

// Type of the element.
func (n *Struct) Type() ir.Type {
	return n.typ
}

// ExprWithName returns a structure literal.
func (n *Struct) ExprWithName(ev ir.Evaluator, src ast.Expr, tp ir.Type) ([]ir.Expr, error) {
	fields := n.typ.Fields.Fields()
	lit := &ir.StructLitExpr{Typ: tp}
	for _, field := range fields {
		fieldVal, has := n.fields[field.Name.Name]
		if !has {
			continue
		}
		x, err := ir.ToSingleExpr(ev, src, fieldVal)
		if err != nil {
			return nil, err
		}
		lit.Elts = append(lit.Elts, &ir.FieldLit{
			FieldStorage: field.Storage(),
			X:            x,
		})
	}
	return []ir.Expr{lit}, nil
}

// Copy the structure to a new node.
func (n *Struct) Copy() engine.Copier {
	cp := &Struct{typ: n.typ}
	cp.fields = make(map[string]ir.Element, len(n.fields))
	for name, field := range n.fields {
		cp.fields[name] = engine.Copy(field)
	}
	return cp
}

// SetField sets the field in the structure.
func (n *Struct) SetField(name string, value ir.Element) {
	n.fields[name] = value
}

// FieldValue returns the value of a field given its name.
// Returns nil if the field has no value.
func (n *Struct) FieldValue(name string) ir.Element {
	return n.fields[name]
}

// SourceString returns the GX source code of the structure.
func (n *Struct) SourceString(from *ir.File) string {
	return n.SourceStringWithName(from, "struct")
}

// SourceStringWithName returns the GX source code of the structure
// given the name of its type.
func (n *Struct) SourceStringWithName(from *ir.File, name string) string {
	var b strings.Builder
	b.WriteString(name)
	b.WriteString("{\n")
	for _, field := range n.typ.Fields.Fields() {
		fieldS := gxfmt.IndentSkip(1, ir.DefineString(from, n.fields[field.Name.Name]))
		fmt.Fprintf(&b, "\t%s: %s,\n", field.Name.Name, fieldS)
	}
	b.WriteString("}")
	return b.String()
}

func (n *Struct) String() string {
	var b strings.Builder
	b.WriteString(n.StructType().ReferString(nil))
	b.WriteString("{\n")
	for i, fld := range n.typ.Fields.Fields() {
		b.WriteString(gxfmt.Indent(fmt.Sprintf("%d: %s %s = %v\n", i, fld.Name.Name, fld.Type().ReferString(nil), gxfmt.String(n.fields[fld.Name.Name]))))
	}
	b.WriteString("}")
	return b.String()
}
