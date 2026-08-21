// Copyright 2026 Google LLC
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

// Package coreiface provides core interfaces for the interpreter.
package coreiface

import (
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/base/cast"
)

// Under is an element with an underlying element.
type Under interface {
	ir.Element
	// Under returns the element of the type that has been named.
	Under() (ir.Element, error)
}

// ToCore removes all wrapper around an element.
func ToCore(el ir.Element) (ir.Element, error) {
	el = ir.BareValue(el)
	return Underlying(el)
}

// CastOk casts an element to type.
func CastOk[T ir.Element](el ir.Element) (T, bool, error) {
	coreEl, err := ToCore(el)
	if err != nil {
		var zero T
		return zero, false, err
	}
	elT, elOk := coreEl.(T)
	return elT, elOk, nil
}

// Cast an element to a given type.
// Returns an error if the cast is not possible.
func Cast[T ir.Element](el ir.Element) (T, error) {
	coreEl, err := ToCore(el)
	if err != nil {
		var zero T
		return zero, err
	}
	return cast.To[T](coreEl)
}

// Underlying returns the underlying element.
func Underlying(val ir.Element) (ir.Element, error) {
	named, ok := val.(Under)
	if !ok {
		return val, nil
	}
	under, err := named.Under()
	if err != nil {
		return nil, err
	}
	return Underlying(under)
}

// Map transforms a collection of element into a different type.
func Map[T any](f func(ir.Element) (T, error), el ir.Element) ([]T, error) {
	slice, err := ToWithElements(el)
	if err != nil {
		return nil, err
	}
	return MapSlice[T](f, slice.Elements())
}

// MapSlice transforms a slice of elements into a different type.
func MapSlice[T any, U ir.Element](f func(ir.Element) (T, error), elts []U) ([]T, error) {
	if len(elts) == 0 {
		return nil, nil
	}
	ts := make([]T, len(elts))
	for i, el := range elts {
		var err error
		ts[i], err = f(el)
		if err != nil {
			return nil, err
		}
	}
	return ts, nil
}

// MapSliceOk transforms a slice of elements into a different type.
// Stop as soon as an element cannot be converted.
func MapSliceOk[T any, U ir.Element](f func(ir.Element) (T, bool), elts []U) ([]T, bool) {
	ts := make([]T, len(elts))
	for i, el := range elts {
		var ok bool
		ts[i], ok = f(el)
		if !ok {
			return nil, false
		}
	}
	return ts, true
}

// WithElements is an element able to returns the elements it contains.
type WithElements interface {
	Elements() []ir.Element
}

// ToWithElements returns the string value stored in a element.
func ToWithElements(el ir.Element) (WithElements, error) {
	under, err := Underlying(el)
	if err != nil {
		return nil, err
	}
	return cast.To[WithElements](under)
}
