// Copyright 2025 Google LLC
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

	"github.com/gx-org/backend/platform"
	"github.com/gx-org/gx/api/hostio"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/base/cast"
	"github.com/gx-org/gx/interp/engine"
)

// NamedType is the GX runtime value of a named type.
type NamedType struct {
	val hostio.Value
	typ ir.TypeMethods
}

var _ hostio.Value = (*NamedType)(nil)
var _ engine.NamedType = (*NamedType)(nil)

// NewNamedType returns a new named type from a GX runtime value and a named type.
func NewNamedType(val hostio.Value, typ ir.TypeMethods) *NamedType {
	return &NamedType{val: val, typ: typ}
}

// Type returns the type of the value.
func (n *NamedType) Type() ir.Type {
	return n.typ
}

// Underlying returns the underlying value.
func (n *NamedType) Underlying() hostio.Value {
	return n.val
}

// Under returns the element stored by this type.
func (n *NamedType) Under() (ir.Element, error) {
	return n.val, nil
}

// Select a field in the structure.
func (n *NamedType) Select(env *engine.Env, expr *ir.SelectorExpr) (ir.Element, error) {
	sel, err := cast.To[engine.Selector](n.val)
	if err != nil {
		return nil, err
	}
	return sel.Select(env, expr)
}

// Copy the element.
func (n *NamedType) Copy() engine.Copier {
	return n.RecvCopy()
}

// RecvCopy copies the underlying element and returns the element encapsulated in this named type.
func (n *NamedType) RecvCopy() engine.NamedType {
	return NewNamedType(engine.Copy(n.val).(hostio.Value), n.typ)
}

// TypeMethods returns the IR named type of the value.
func (n *NamedType) TypeMethods() ir.TypeMethods {
	return n.typ
}

// ToHost transfers the value to host given an allocator.
func (n *NamedType) ToHost(alloc platform.Allocator) (hostio.Value, error) {
	hostVal, err := n.val.ToHost(alloc)
	if err != nil {
		return nil, err
	}
	return NewNamedType(hostVal, n.typ), nil
}

// SourceString returns the GX source code of the implementation.
func (n *NamedType) SourceString(from *ir.File) string {
	underStruct, ok := n.val.(*Struct)
	if ok {
		return underStruct.toString(from, n.typ.ReferString(from))
	}
	return fmt.Sprintf("%s(%s)", n.typ.ReferString(from), n.val.SourceString(from))
}
