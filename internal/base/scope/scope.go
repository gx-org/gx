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

// Package scope provides types for modeling execution scopes, as well as simple namespaces.
package scope

import "github.com/pkg/errors"

type scope[V any] interface {
	Assign(string, V) error
	Delete(string) error
	Find(string) (V, bool)
}

// Map is a mapping from names to values.
type Map[V any] map[string]V

// Assign maps `key` to `value`, overwriting if necessary.
func (m *Map[V]) Assign(key string, value V) {
	(*m)[key] = value
}

// Delete removes the mapping for `key`.
func (m *Map[V]) Delete(key string) {
	delete(*m, key)
}

// Find returns the value associated with `key`, if any.
//
// The second return value indicates whether any value was found.
func (m *Map[V]) Find(key string) (V, bool) {
	value, ok := (*m)[key]
	return value, ok
}

// ImmutableScope defines a root-level immutable scope, suitable for defining core builtins.
type ImmutableScope[C any, V any] struct {
	data    Map[V]
	context C
}

var _ scope[any] = (*ImmutableScope[any, any])(nil)

// NewImmutableScope returns a new ImmutableScope. data must be non-nil, since it is immutable.
func NewImmutableScope[C any, V any](context C, data map[string]V) ImmutableScope[C, V] {
	return ImmutableScope[C, V]{data, context}
}

// NewChild returns a new mutable child scope.
//
// The resulting scope inherits the current scope's definitions.
func (s ImmutableScope[C, V]) NewChild(context C) *Scope[C, V] {
	return &Scope[C, V]{
		data:    make(Map[V]),
		parent:  s,
		context: context,
	}
}

// Context returns a pointer to the scope's user-defined context. Note that the context cannot be modified.
func (s ImmutableScope[C, V]) Context() *C {
	return &s.context
}

// Assign returns an error, since an immutable scope cannot be modified.
func (s ImmutableScope[C, V]) Assign(key string, _ V) error {
	return errors.Errorf("cannot assign %s: immutable scope", key)
}

// Delete returns an error, since an immutable scope cannot be modified.
func (s ImmutableScope[C, V]) Delete(key string) error {
	return errors.Errorf("cannot delete %s: immutable scope", key)
}

// Find returns the value associated with `key`, if any.
//
// The second return value indicates whether any value was found.
func (s ImmutableScope[C, V]) Find(key string) (V, bool) {
	value, ok := s.data[key]
	return value, ok
}

// Scope defines a set of nested namespaces. Each scope may carry user-defined context.
type Scope[C any, V any] struct {
	data    Map[V]
	parent  scope[V]
	context C
}

var _ scope[any] = (*Scope[any, any])(nil)

// NewScope returns a new Scope with the provided user-defined context, and optional initial
// contents.
func NewScope[C any, V any](context C, data map[string]V) *Scope[C, V] {
	if data == nil {
		data = make(map[string]V)
	}
	return &Scope[C, V]{
		data:    data,
		context: context,
	}
}

// NewChild returns a new mutable child scope.
//
// The resulting scope holds a new empty namespace, while inheriting the current scope's definitions.
func (s *Scope[C, V]) NewChild(context C) *Scope[C, V] {
	return &Scope[C, V]{
		data:    make(Map[V]),
		parent:  s,
		context: context,
	}
}

// Map returns a pointer to the scope's innermost namespace.
func (s *Scope[C, V]) Map() *Map[V] {
	return &s.data
}

// Context returns a pointer to the scope's user-defined context.
func (s *Scope[C, V]) Context() *C {
	return &s.context
}

// Define maps `key` to `value`, overwriting if necessary.
func (s *Scope[C, V]) Define(key string, value V) {
	s.data[key] = value
}

// Delete removes the mapping for `key`.
func (s *Scope[C, V]) Delete(key string) error {
	if _, exists := s.data.Find(key); exists {
		s.data.Delete(key)
		return nil
	}
	if s.parent == nil {
		return errors.Errorf("cannot delete %s: not defined in scope", key)
	}
	return s.parent.Delete(key)
}

// Assign maps an existing `key` to `value`, failing if no matching mapping is found. The assignment
// starts at the scope's innermost namespace and cascades upwards through successive parent scopes.
func (s *Scope[C, V]) Assign(key string, value V) error {
	if _, exists := s.data.Find(key); exists {
		s.Define(key, value)
		return nil
	}
	if s.parent == nil {
		return errors.Errorf("cannot assign %s: not defined in scope", key)
	}
	return s.parent.Assign(key, value)
}

// Find returns the value associated with `key`, if any. The lookup starts at the scope's innermost
// namespace and cascades upwards through successive parent scopes until a matching mapping is found.
//
// The second return value indicates whether any value was found.
func (s *Scope[C, V]) Find(key string) (V, bool) {
	value, ok := s.data.Find(key)
	if ok || s.parent == nil {
		return value, ok
	}
	return s.parent.Find(key)
}
