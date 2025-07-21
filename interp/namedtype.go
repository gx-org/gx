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

package interp

import (
	"go/ast"

	"github.com/pkg/errors"
	"github.com/gx-org/gx/api/values"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/interp/flatten"
)

// NamedType references a type exported by an imported package.
type NamedType struct {
	newFunc NewFunc
	typ     *ir.NamedType
	funcs   map[string]ir.PkgFunc
	under   Copier
}

var (
	_ Selector = (*NamedType)(nil)
	_ NType    = (*NamedType)(nil)
	_ Copier   = (*NamedType)(nil)
)

// NewNamedType returns a new node representing an exported type.
func NewNamedType(newFunc NewFunc, typ *ir.NamedType, under Copier) *NamedType {
	funcs := make(map[string]ir.PkgFunc)
	for _, fun := range typ.Methods {
		funcs[fun.Name()] = fun
	}
	return &NamedType{
		newFunc: newFunc,
		typ:     typ,
		funcs:   funcs,
		under:   under,
	}
}

// Select returns the field given an index.
// Returns nil if the receiver type cannot select fields.
func (n *NamedType) Select(expr *ir.SelectorExpr) (ir.Element, error) {
	name := expr.Stor.NameDef().Name
	if fn := n.funcs[name]; fn != nil {
		return n.newFunc(fn, NewReceiver(n, fn)), nil
	}
	under, ok := n.under.(Selector)
	if !ok {
		return nil, errors.Errorf("%s is undefined", name)
	}
	return under.Select(expr)
}

// Copy the element.
func (n *NamedType) Copy() Copier {
	return n.RecvCopy()
}

// RecvCopy copies the underlying element and returns the element encapsulated in this named type.
func (n *NamedType) RecvCopy() *NamedType {
	return NewNamedType(n.newFunc, n.typ, n.under.Copy())
}

// Under returns the underlying element of the named type.
func (n *NamedType) Under() ir.Element {
	return n.under
}

// Flatten returns the named type in a slice of elements.
func (n *NamedType) Flatten() ([]ir.Element, error) {
	return flatten.Flatten(n.under)
}

// NamedType returns the type of the element.
func (n *NamedType) NamedType() *ir.NamedType {
	return n.typ
}

// Unflatten consumes the next handles to return a GX value.
func (n *NamedType) Unflatten(handles *flatten.Parser) (values.Value, error) {
	val, err := handles.Unflatten(n.under)
	if err != nil {
		return nil, err
	}
	return values.NewNamedType(val, n.typ), nil
}

// Type of the element.
func (n *NamedType) Type() ir.Type {
	return n.typ
}

// String returns a string representation of the node.
func (n *NamedType) String() string {
	return n.typ.FullName()
}

// Receiver of a function.
type Receiver struct {
	Ident   *ast.Ident
	Element *NamedType
}

// NewReceiver returns a new receiver given a function definition and the element representing the receiver.
func NewReceiver(el *NamedType, fn ir.Func) *Receiver {
	if el == nil {
		return nil
	}
	names := fn.FuncType().Receiver.Src.List[0].Names
	var name *ast.Ident
	if len(names) > 0 {
		name = names[0]
	}
	return &Receiver{
		Ident:   name,
		Element: el,
	}
}

// Underlying returns the underlying element.
func Underlying(val ir.Element) ir.Element {
	named, ok := val.(*NamedType)
	if !ok {
		return val
	}
	return Underlying(named.under)
}
