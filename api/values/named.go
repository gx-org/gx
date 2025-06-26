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
	"github.com/gx-org/backend/platform"
	"github.com/gx-org/gx/build/ir"
)

// NamedType is the GX runtime value of a named type.
type NamedType struct {
	val Value
	typ *ir.NamedType
}

var _ Value = (*NamedType)(nil)

// NewNamedType returns a new named type from a GX runtime value and a named type.
func NewNamedType(val Value, typ *ir.NamedType) *NamedType {
	return &NamedType{val: val, typ: typ}
}

func (*NamedType) value() {}

// Type returns the type of the value.
func (n *NamedType) Type() ir.Type {
	return n.typ
}

// Underlying returns the underlying value.
func (n *NamedType) Underlying() Value {
	return n.val
}

// NamedType returns the IR named type of the value.
func (n *NamedType) NamedType() *ir.NamedType {
	return n.typ
}

// ToHost transfers the value to host given an allocator.
func (n *NamedType) ToHost(alloc platform.Allocator) (Value, error) {
	hostVal, err := n.val.ToHost(alloc)
	if err != nil {
		return nil, err
	}
	return NewNamedType(hostVal, n.typ), nil
}

// String representation of the value.
// The returned string is a string reported to the user.
func (n *NamedType) String() string {
	return n.val.String()
}
