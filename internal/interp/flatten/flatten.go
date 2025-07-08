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

// Package flatten provides utilities to flatten/unflatten interpreter elements.
package flatten

import (
	"github.com/gx-org/gx/api/values"
	"github.com/gx-org/gx/build/ir"
)

type (
	// Flattener is an element that can be flattened.
	Flattener interface {
		// Flatten the element, that is:
		// returns itself if the element is atomic,
		// returns its components if the element is a composite.
		Flatten() ([]ir.Element, error)
	}

	// Unflattener uses the parser to build GX values.
	Unflattener interface {
		// Unflatten creates a GX value from the next handles available in the Unflattener.
		Unflatten(handles *Parser) (values.Value, error)
	}
)

// Flatten elements.
func Flatten(elts ...ir.Element) ([]ir.Element, error) {
	var flat []ir.Element
	for _, elt := range elts {
		subs, err := elt.(Flattener).Flatten()
		if err != nil {
			return nil, err
		}
		flat = append(flat, subs...)
	}
	return flat, nil
}
