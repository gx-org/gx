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

// Package togo provides a mechanism to convert GX values to Go values.
package togo

import (
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/base/cast"
)

// WithGoValue is an element able to convert its value into an equivalent Go value.
type WithGoValue interface {
	ir.Element
	// GoValue of the underlying element.
	GoValue() (any, error)
}

// Value converts an element to its Go value.
// If the conversion is not possible, returns the input element.
func Value(el ir.Element) (any, error) {
	togo, err := cast.To[WithGoValue](el)
	if err != nil {
		return nil, err
	}
	return togo.GoValue()
}

// ValueT converts an element to a specific Go value.
func ValueT[T any](el ir.Element) (valT T, err error) {
	valAny, err := Value(el)
	if err != nil {
		return
	}
	return cast.To[T](valAny)
}
