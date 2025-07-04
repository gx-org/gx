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

package sync

import "sync"

// Map is a generic synchronized map. It is a wrapper around Go's standard
// sync.Map, with all the same caveats.
type Map[K comparable, V any] struct {
	m sync.Map
}

// Store a key,value pair.
func (sm *Map[K, V]) Store(k K, v V) {
	sm.m.Store(k, v)
}

// Load returns a value given a key.
func (sm *Map[K, V]) Load(k K) (v V) {
	vAny, ok := sm.m.Load(k)
	if !ok {
		return
	}
	return vAny.(V)
}

// LoadAndDelete deletes the value for a key, returning the previous value (if any) and whether the key was present.
func (sm *Map[K, V]) LoadAndDelete(k K) (any, bool) {
	return sm.m.LoadAndDelete(k)
}

// Delete removes a pair given a key.
func (sm *Map[K, V]) Delete(k K) {
	sm.m.Delete(k)
}

// Empty returns true if the map is empty.
func (sm *Map[K, V]) Empty() bool {
	for range sm.Iter() {
		return false
	}
	return true
}

// Size returns the number of elements in the map. This takes O(n) time.
func (sm *Map[K, V]) Size() (i int) {
	for range sm.Iter() {
		i++
	}
	return
}

// Iter returns an iterator to range over the elements of the map.
func (sm *Map[K, V]) Iter() func(func(K, V) bool) {
	return func(yield func(K, V) bool) {
		sm.m.Range(func(k, v any) bool {
			return yield(k.(K), v.(V))
		})
	}
}
