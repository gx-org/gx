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

package flatten

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"github.com/gomlx/compute"
	gxfmt "github.com/gx-org/gx/base/fmt"
	"github.com/gx-org/gx/build/ir"
)

// Parser unflattens the output of a graph computation
// into values.
type Parser[T any] struct {
	// Handles to unflatten.
	compOutput []compute.Buffer
	nextPos    int
}

// NewParser returns a new parser given the output of a graph computation.
func NewParser[T any](handles []compute.Buffer) *Parser[T] {
	return &Parser[T]{
		compOutput: handles,
	}
}

// Next returns the next handle and moves the cursor to the succeeding handle.
func (h *Parser[T]) Next() compute.Buffer {
	n := h.compOutput[h.nextPos]
	h.nextPos++
	return n
}

// Unflatten consumes the next available handles and returns a value matching the given element.
func (h *Parser[T]) Unflatten(el ir.Element) (T, error) {
	var zero T
	// Check for an unflattener first: some values (for example, slices)
	// can store elements which also need to be unflattened.
	unf, ok := el.(Unflattener[T])
	if !ok {
		val, ok := el.(T)
		if !ok {
			return zero, errors.Errorf("cannot unflatten element type %T", el)
		}
		return val, nil
	}
	val, err := unf.Unflatten(h)
	if err != nil {
		return zero, err
	}
	if any(val) == nil {
		return zero, errors.Errorf("state element %T returned a nil value", el)
	}
	return val, nil
}

// ParseComposite unflattens a slice of elements into a single value.
func (h *Parser[T]) ParseComposite(ncv func(ir.Type, []T) (T, error), typ ir.Type, els []ir.Element) (T, error) {
	var zero T
	vals := make([]T, len(els))
	for i, el := range els {
		var err error
		vals[i], err = h.Unflatten(el)
		if err != nil {
			return zero, err
		}
	}
	return ncv(typ, vals)
}

func (h *Parser[T]) String() string {
	var s strings.Builder
	fmt.Fprintf(&s, "%T{", h)
	for i, hdl := range h.compOutput {
		prefix := "  "
		if i == h.nextPos {
			prefix = "->"
		}
		fmt.Fprintf(&s, "%s%d: %s\n", prefix, i, gxfmt.String(hdl))
	}
	s.WriteString("}")
	return s.String()
}
