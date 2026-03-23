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

import "github.com/gx-org/gx/build/ir"

// WithGoValue is an element able to convert its value into an equivalent Go value.
type WithGoValue interface {
	ir.Element
	// GoValue of the underlying element.
	GoValue() (any, error)
}

// Value converts an element to its Go value.
// If the conversion is not possible, returns the input element.
func Value(el ir.Element) (any, error) {
	togo, ok := el.(WithGoValue)
	if !ok {
		return togo, nil
	}
	return togo.GoValue()
}

// Values converts a slice of an element to a slice of Go values.
func Values(els []ir.Element) ([]any, error) {
	vals := make([]any, len(els))
	for i, el := range els {
		var err error
		vals[i], err = Value(el)
		if err != nil {
			return nil, err
		}
	}
	return vals, nil
}
