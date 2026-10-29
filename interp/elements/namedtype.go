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
	"reflect"

	"github.com/pkg/errors"
	"github.com/gx-org/gx/api/hostio"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/golang/backend/kernels"
	"github.com/gx-org/gx/internal/base/cast"
	"github.com/gx-org/gx/internal/interp/flatten"
	"github.com/gx-org/gx/interp/engine"
	"github.com/gx-org/gx/src/gx/ast"
)

// NamedType references a type exported by an imported package.
type NamedType struct {
	typ   ir.TypeMethods
	funcs map[string]ir.PkgFunc
	under ir.Element
}

var (
	_ ir.StorageElement = (*NamedType)(nil)
	_ ir.WithExpr       = (*NamedType)(nil)
	_ engine.NamedType  = (*NamedType)(nil)
	_ hostio.Value      = (*NamedType)(nil)
	_ hostio.Under      = (*NamedType)(nil)
)

// NewNamedType returns a new node representing an exported type.
func NewNamedType(typ ir.TypeMethods, under ir.Element) *NamedType {
	funcs := make(map[string]ir.PkgFunc)
	for _, fun := range typ.Methods() {
		funcs[fun.Name()] = fun
	}
	return &NamedType{
		typ:   typ,
		funcs: funcs,
		under: under,
	}
}

// Select returns the field given an index.
// Returns nil if the receiver type cannot select fields.
func (n *NamedType) Select(env *engine.Env, expr *ir.SelectorExpr) (ir.Element, error) {
	name := expr.Stor.NameDef().Name
	if fn := n.funcs[name]; fn != nil {
		return NewFunc(fn, NewReceiver(n, fn)), nil
	}
	under, err := cast.To[engine.Selector](n.under)
	if err != nil {
		return nil, err
	}
	return under.Select(env, expr)
}

// Copy the element.
func (n *NamedType) Copy() engine.Copier {
	return n.RecvCopy()
}

// RecvCopy copies the underlying element and returns the element encapsulated in this named type.
func (n *NamedType) RecvCopy() engine.NamedType {
	return NewNamedType(n.typ, engine.Copy(n.under))
}

// Under returns the underlying element of the named type.
func (n *NamedType) Under() (ir.Element, error) {
	return n.under, nil
}

// Underlying returns the underlying GX value of the named type.
// Returns nil if the underlying element is not a GX value.
func (n *NamedType) Underlying() hostio.Value {
	val, _ := n.under.(hostio.Value)
	return val
}

// Flatten returns the named type in a slice of elements.
func (n *NamedType) Flatten() ([]ir.Element, error) {
	return flatten.Flatten(n.under)
}

// TypeMethods returns the type of the element.
func (n *NamedType) TypeMethods() ir.TypeMethods {
	return n.typ
}

// Unflatten consumes the next handles to return a GX value.
func (n *NamedType) Unflatten(parser *flatten.Parser) (hostio.Value, error) {
	val, err := parser.Unflatten(n.under)
	if err != nil {
		return nil, err
	}
	return NewNamedType(n.typ, val), nil
}

// ToHost transfers the underlying value to the host.
func (n *NamedType) ToHost(alloc kernels.Allocator) (hostio.Value, error) {
	val, err := cast.To[hostio.Value](n.under)
	if err != nil {
		return nil, err
	}
	hostVal, err := val.ToHost(alloc)
	if err != nil {
		return nil, err
	}
	return NewNamedType(n.typ, hostVal), nil
}

// Finalize releases the resources allocated for the underlying value.
func (n *NamedType) Finalize() error {
	val, err := cast.To[hostio.Value](n.under)
	if err != nil {
		return err
	}
	return val.Finalize()
}

// Type of the element.
func (n *NamedType) Type() ir.Type {
	return n.typ
}

// Store returns the storage owning the type definition.
func (n *NamedType) Store() ir.Storage {
	return n.typ
}

// Expr returns an expression representing the named expression.
func (n *NamedType) Expr(ev ir.Evaluator, src ast.Expr) ([]ir.Expr, error) {
	switch underT := n.under.(type) {
	case *Struct:
		return underT.ExprWithName(ev, src, n.typ)
	default:
		return ir.ToExpr(ev, src, n.under)
	}
}

// String returns a string representation of the node.
func (n *NamedType) String() string {
	return n.typ.ReferString(nil)
}

// SourceString returns the GX source code of the implementation.
func (n *NamedType) SourceString(from *ir.File) string {
	if underStruct, ok := n.under.(*Struct); ok {
		return underStruct.SourceStringWithName(from, n.typ.ReferString(from))
	}
	return fmt.Sprintf("%s(%s)", n.typ.ReferString(from), ir.DefineString(from, n.under))
}

// NewReceiver returns a new receiver given a function definition and the element representing the receiver.
func NewReceiver(el *NamedType, fn ir.Func) *engine.Receiver {
	if el == nil {
		return nil
	}
	var name *ast.Ident
	recv := fn.FuncType().Receiver
	if recv != nil {
		names := recv.Src.List[0].Names
		if len(names) > 0 {
			name = names[0]
		}
	}
	return &engine.Receiver{
		Ident:   name,
		Element: el,
	}
}

// ToNamedType converts a generic element to a named type.
func ToNamedType(el ir.Element) (*NamedType, error) {
	named, ok := el.(*NamedType)
	if !ok {
		return nil, errors.Errorf("cannot convert %T to %s", el, reflect.TypeFor[*NamedType]().Name())
	}
	return named, nil
}
