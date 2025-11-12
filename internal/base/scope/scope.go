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

import (
	"fmt"
	"iter"
	"strings"

	"github.com/pkg/errors"
	"github.com/gx-org/gx/base/ordered"
)

type (
	// Scope provides a set of values that can be find given their name.
	Scope[V any] interface {
		CanAssign(string) bool
		Find(string) (V, bool)
		Items() *ordered.Map[string, V]
	}

	roScope[V any] struct {
		parent Scope[V]
		local  Scope[V]
	}
)

// NewReadOnly returns a scope that can only be queried and not modified.
func NewReadOnly[V any](parent, data Scope[V]) Scope[V] {
	return &roScope[V]{
		parent: parent,
		local:  data,
	}
}

func find[V any](key string, local Scope[V], parent Scope[V]) (value V, ok bool) {
	value, ok = local.Find(key)
	if ok || parent == nil {
		return
	}
	return parent.Find(key)
}

// Find returns the value associated with `key`, if any.
//
// The second return value indicates whether any value was found.
func (s *roScope[V]) Find(key string) (value V, ok bool) {
	return find(key, s.local, s.parent)
}

func (s *roScope[V]) CanAssign(key string) bool {
	return false
}

func mergeItems[V any](scopes ...Scope[V]) *ordered.Map[string, V] {
	all := ordered.NewMap[string, V]()
	for _, scope := range scopes {
		if scope == nil {
			continue
		}
		for k, v := range scope.Items().Iter() {
			all.Store(k, v)
		}
	}
	return all
}

func (s *roScope[V]) Items() *ordered.Map[string, V] {
	return mergeItems(s.parent, s.local)
}

func (s *roScope[V]) String() string {
	return scopeString(s.local, s.parent)
}

type localScope[V any] struct {
	data *ordered.Map[string, V]
}

// NewScopeWithValues returns a scope with predefined values.
func NewScopeWithValues[V any](vals map[string]V) Scope[V] {
	data := ordered.NewMap[string, V]()
	for k, v := range vals {
		data.Store(k, v)
	}
	return NewReadOnly(nil, &localScope[V]{data: data})
}

func (s *localScope[V]) Find(key string) (value V, ok bool) {
	return s.data.Load(key)
}

func (s *localScope[V]) CanAssign(key string) bool {
	_, ok := s.data.Load(key)
	return ok
}

func (s *localScope[V]) Items() *ordered.Map[string, V] {
	return s.data.Clone()
}

func (s *localScope[V]) String() string {
	if s.data.Size() == 0 {
		return "empty"
	}
	var kvs []string
	for k, v := range s.data.Iter() {
		kvs = append(kvs, fmt.Sprintf("%s: %T:%v", k, v, v))
	}
	return strings.Join(kvs, "\n")
}

// RWScope stores key,value pairs and the Scope interface.
// A key, value pair is always defined within the scope.
// A value can retrieved from its key by querying the scope and,
// if not found, its parents recursively.
type RWScope[V any] struct {
	parent Scope[V]
	local  *localScope[V]
}

var _ Scope[any] = (*RWScope[any])(nil)

// NewScope returns a new scope given a parent, which can be nil.
func NewScope[V any](parent Scope[V]) *RWScope[V] {
	return &RWScope[V]{
		parent: parent,
		local: &localScope[V]{
			data: ordered.NewMap[string, V](),
		},
	}
}

// Define maps `key` to `value`, overwriting if necessary.
func (s *RWScope[V]) Define(k string, v V) {
	s.local.data.Store(k, v)
}

// CanAssign returns true if a key can be assigned.
func (s *RWScope[V]) CanAssign(key string) bool {
	if can := s.local.CanAssign(key); can || s.parent == nil {
		return can
	}
	return s.parent.CanAssign(key)
}

// LocalKeys returns the keys of the local scope without the parent.
func (s *RWScope[V]) LocalKeys() iter.Seq[string] {
	return s.local.data.Keys()
}

// IsLocal returns true if the key is defined in the local scope.
func (s *RWScope[V]) IsLocal(key string) bool {
	_, ok := s.local.data.Load(key)
	return ok
}

// Find a key in the scope and its parent.
func (s *RWScope[V]) Find(key string) (value V, ok bool) {
	return find[V](key, s.local, s.parent)
}

// Assign maps an existing `key` to `value`, failing if no matching mapping is found. The assignment
// starts at the scope's innermost namespace and cascades upwards through successive parent scopes.
func (s *RWScope[V]) Assign(key string, value V) error {
	if _, exists := s.local.data.Load(key); exists {
		s.Define(key, value)
		return nil
	}
	if s.parent == nil {
		return errors.Errorf("cannot assign %s: not defined in scope", key)
	}
	rwParent, ok := s.parent.(*RWScope[V])
	if !ok {
		return errors.Errorf("cannot assign %s: scope parent of type %T does support assignment", key, s.parent)
	}
	return rwParent.Assign(key, value)
}

// Items returns the list of the items in the scope.
func (s *RWScope[V]) Items() *ordered.Map[string, V] {
	return mergeItems(s.parent, s.local)
}

// Collect all the elements in the scope.
// Note that all scopes are merged in a single scope, meaning
// that values with overwritten values are lost.
func (s *RWScope[V]) Collect() Scope[V] {
	return &RWScope[V]{
		local: &localScope[V]{
			data: s.Items(),
		},
	}
}

// ReadOnly returns a scope to which values cannot be assigned or defined.
func (s *RWScope[V]) ReadOnly() Scope[V] {
	return &roScope[V]{parent: s.parent, local: s.local}
}

func (s *RWScope[V]) localString() string {
	return s.local.String()
}

func scopeString[V any](local any, parent Scope[V]) string {
	parentS := "root"
	if parent != nil {
		if rws, ok := parent.(*RWScope[V]); ok {
			parentS = rws.localString()
		} else {
			parentS = fmt.Sprint(parent)
		}
	}
	return fmt.Sprintf("%s\n-- %p --\n%v\n", parentS, local, local)
}

// String representation of the scope.
func (s *RWScope[V]) String() string {
	return scopeString(s.local, s.parent)
}
