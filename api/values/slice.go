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
	"reflect"
	"strings"

	"github.com/pkg/errors"
	"github.com/gx-org/backend/platform"
	"github.com/gx-org/gx/build/ir"
)

// Slice of GX values.
type Slice struct {
	vals      []Value
	typ       ir.Type
	sliceType *ir.SliceType
}

var _ Value = (*Slice)(nil)

// NewSlice returns a new slice of GX values.
// vals can be nil when the slice will be allocated
// and constructed later.
func NewSlice(typ ir.Type, vals []Value) (*Slice, error) {
	under := ir.Underlying(typ)
	sliceType, ok := under.(*ir.SliceType)
	if !ok {
		return nil, errors.Errorf("cannot convert %T to %s", under, reflect.TypeFor[*ir.SliceType]().String())
	}
	return &Slice{
		vals:      vals,
		typ:       typ,
		sliceType: sliceType,
	}, nil
}

func (*Slice) value() {}

// ToHost transfers all the elements of the slice to the host.
func (s *Slice) ToHost(alloc platform.Allocator) (Value, error) {
	vals, err := ToHost(alloc, s.vals)
	if err != nil {
		return nil, err
	}
	return NewSlice(s.Type(), vals)
}

// Allocate a slice of GX values given a size.
func (s *Slice) Allocate(size int) {
	s.vals = make([]Value, size)
}

// Set the ith element of the slice.
func (s *Slice) Set(i int, val Value) {
	s.vals[i] = val
}

// Size of the slice.
func (s *Slice) Size() int {
	return len(s.vals)
}

// Element at the ith position.
func (s *Slice) Element(i int) Value {
	var v any = s.vals[i]
	if value, ok := v.(Value); ok {
		return value
	}
	return v.(Valuer).GXValue()
}

// Type of the slice.
func (s *Slice) Type() ir.Type {
	return s.typ
}

// SliceType returns the type of the slice.
func (s *Slice) SliceType() *ir.SliceType {
	return s.sliceType
}

// String representation of the slice.
func (s *Slice) String() string {
	ss := make([]string, len(s.vals))
	for i, val := range s.vals {
		ss[i] = fmt.Sprint(val)
	}
	return "[" + strings.Join(ss, ",") + "]"
}
