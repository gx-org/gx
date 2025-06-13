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

package cpevelements

import (
	"github.com/pkg/errors"
	"github.com/gx-org/gx/api/values"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/interp/elements"
)

type axis struct {
	src elements.NodeFile[ir.AxisLengths]
}

var (
	_ elements.Element = (*axis)(nil)
)

func newAxis(file *ir.File, src ir.AxisLengths) elements.Element {
	return &axis{src: elements.NewNodeAt(file, src)}
}

// Flatten the axis into a slice of elements.
func (a *axis) Flatten() ([]elements.Element, error) {
	return []elements.Element{a}, nil
}

// Unflatten creates a GX value from the next handles available in the Unflattener.
func (a *axis) Unflatten(handles *elements.Unflattener) (values.Value, error) {
	return nil, errors.Errorf("not implemented")
}

// Kind of the element.
func (a *axis) Kind() ir.Kind {
	return a.src.Node().Type().Kind()
}
