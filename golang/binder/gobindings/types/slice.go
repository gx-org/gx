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

package types

import (
	"reflect"

	"github.com/pkg/errors"
	"github.com/gx-org/gx/api/values"
	"github.com/gx-org/gx/build/ir"
)

// Slice of Go instances.
type Slice[T Bridger] struct {
	bridge  SliceBridge
	vals    []T
	factory func() (Bridge, error)
}

var _ = (*Slice[Bridger])(nil)

// NewSlice returns a new slice storing data on a device.
func NewSlice[T Bridger](typ ir.Type, vals []T) (*Slice[T], error) {
	gxVals := make([]values.Value, len(vals))
	for i, val := range vals {
		gxVals[i] = val.Bridge().GXValue()
	}
	gxSlice, err := values.NewSlice(typ, gxVals)
	if err != nil {
		return nil, err
	}
	s := &Slice[T]{
		vals: vals,
	}
	s.bridge = SliceBridge{
		goSlice: s,
		gxSlice: gxSlice,
	}
	return s, nil
}

// NewEmptySlice creates an empty slice that will be allocated later.
func NewEmptySlice[T Bridger](typ ir.Type, factory func() (Bridge, error)) (*Slice[T], error) {
	s := &Slice[T]{factory: factory}
	gxSlice, err := values.NewSlice(typ, nil)
	if err != nil {
		return nil, err
	}
	s.bridge = SliceBridge{
		goSlice: s,
		gxSlice: gxSlice,
	}
	return s, nil
}

func (s *Slice[T]) allocate(size int) error {
	if len(s.vals) != size {
		s.vals = make([]T, size)
	}
	s.bridge.gxSlice.Allocate(size)
	if s.factory == nil {
		return nil
	}
	for i := range s.vals {
		val, err := s.factory()
		if err != nil {
			return err
		}
		s.set(i, val)
		s.bridge.gxSlice.Set(i, s.vals[i].Bridge().GXValue())
	}
	return nil
}

func (s *Slice[T]) set(i int, el Bridge) error {
	bridger := el.Bridger()
	tVal, ok := bridger.(T)
	if !ok {
		return errors.Errorf("cannot cast %T to %s", bridger, reflect.TypeFor[T]().String())
	}
	s.vals[i] = tVal
	return nil
}

func (s *Slice[T]) get(i int) Bridge {
	return s.vals[i].Bridge()
}

// Bridge returns the bridge of the slice.
func (s *Slice[T]) Bridge() Bridge {
	return &s.bridge
}

// Size of the slice.
func (s *Slice[T]) Size() int {
	return len(s.vals)
}

// At returns the element at the ith position.
func (s *Slice[T]) At(i int) T {
	return s.vals[i]
}

// SliceBridge is the bridge between Go and GX values for a slice.
type SliceBridge struct {
	goSlice interface {
		Bridger
		allocate(size int) error
		get(i int) Bridge
		set(i int, el Bridge) error
	}
	gxSlice *values.Slice
}

// Allocate the slice to a given number of elements.
func (b *SliceBridge) Allocate(size int) {
	b.goSlice.allocate(size)
}

// Get returns the value of the ith position.
func (b *SliceBridge) Get(i int) Bridge {
	return b.goSlice.get(i)
}

// Set an element in the slice.
// The slice needs to have been pre-allocated before.
func (b *SliceBridge) Set(i int, el Bridge) error {
	return b.goSlice.set(i, el)
}

// Bridger returns the Go value of the slice.
func (b *SliceBridge) Bridger() Bridger {
	return b.goSlice
}

// GXValue returns the underlying GX value.
func (b *SliceBridge) GXValue() values.Value {
	return b.gxSlice
}

// DType returns the data type of the slice.
func (b *SliceBridge) DType() ir.Type {
	return b.gxSlice.SliceType().DType
}
