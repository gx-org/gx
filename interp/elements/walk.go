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

package elements

import (
	"github.com/gx-org/gx/build/ir"
)

// walkable is an element that can be walked.
type walkable interface {
	ir.Element
	// Walk the element and its children.
	Walk(walker) (ir.Element, error)
}

// walker walks across a tree of elements.
type walker interface {
	// NamedType walks through a named type.
	NamedType(*NamedType) (ir.Element, error)
	// Struct walks through a structure.
	Struct(*Struct) (ir.Element, error)
	// Slice walks through a slice.
	Slice(*Slice) (ir.Element, error)
}

// roWalker finalizes the content of the elements it walks through.
type roWalker struct {
	cb func(ir.Element) error
}

func (w roWalker) process(el ir.Element) error {
	if walkable, canWalk := el.(walkable); canWalk {
		_, err := walkable.Walk(w)
		return err
	}
	return w.cb(el)
}

// NamedType finalizes the underlying element of a named type.
func (w roWalker) NamedType(n *NamedType) (ir.Element, error) {
	under, err := n.Under()
	if err != nil {
		return nil, err
	}
	return n, w.process(under)
}

// Struct finalizes the field values of a structure.
func (w roWalker) Struct(s *Struct) (ir.Element, error) {
	for _, field := range s.fields {
		if err := w.process(field); err != nil {
			return nil, err
		}
	}
	return s, nil
}

// Slice finalizes the elements of a slice.
func (w roWalker) Slice(s *Slice) (ir.Element, error) {
	for _, el := range s.values {
		if err := w.process(el); err != nil {
			return nil, err
		}
	}
	return s, nil
}

// Walk calls cb on all the leaves of the element.
func Walk(cb func(ir.Element) error, el ir.Element) error {
	return roWalker{cb: cb}.process(el)
}

// trWalker transforms the leaves of the elements it walks through.
type trWalker struct {
	cb func(ir.Element) (ir.Element, error)
}

func (w trWalker) process(el ir.Element) (ir.Element, error) {
	if walkable, canWalk := el.(walkable); canWalk {
		return walkable.Walk(w)
	}
	return w.cb(el)
}

// NamedType returns a new named type with the transformed underlying element.
func (w trWalker) NamedType(n *NamedType) (ir.Element, error) {
	under, err := n.Under()
	if err != nil {
		return nil, err
	}
	under, err = w.process(under)
	if err != nil {
		return nil, err
	}
	return NewNamedType(n.typ, under), nil
}

// Struct returns a new structure with the transformed field values.
func (w trWalker) Struct(s *Struct) (ir.Element, error) {
	fields := make(map[string]ir.Element, len(s.fields))
	for name, field := range s.fields {
		val, err := w.process(field)
		if err != nil {
			return nil, err
		}
		fields[name] = val
	}
	return NewStruct(s.typ, fields)
}

// Slice returns a new slice with the transformed elements.
func (w trWalker) Slice(s *Slice) (ir.Element, error) {
	values := make([]ir.Element, len(s.values))
	for i, el := range s.values {
		var err error
		if values[i], err = w.process(el); err != nil {
			return nil, err
		}
	}
	return NewSlice(s.typ, values)
}

// Transform recursively transform an element into another.
func Transform(cb func(ir.Element) (ir.Element, error), el ir.Element) (ir.Element, error) {
	return trWalker{cb: cb}.process(el)
}
