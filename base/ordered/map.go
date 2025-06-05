// Package ordered provides ordered data structure.
package ordered

// Map is an ordered map. Iter iterates over the map
// using the same order in which the keys have been added.
type Map[K comparable, V any] struct {
	keys []K
	m    map[K]V
}

// NewMap returns a new ordered map.
func NewMap[K comparable, V any]() *Map[K, V] {
	return &Map[K, V]{m: make(map[K]V)}
}

// Store a key,value pair.
func (m *Map[K, V]) Store(k K, v V) {
	_, in := m.m[k]
	if !in {
		m.keys = append(m.keys, k)
	}
	m.m[k] = v
}

// Load returns a value given a key.
func (m *Map[K, V]) Load(k K) (V, bool) {
	v, ok := m.m[k]
	return v, ok
}

// Iter returns an iterator to range over the elements of the map.
func (m *Map[K, V]) Iter() func(func(K, V) bool) {
	return func(yield func(K, V) bool) {
		for _, k := range m.keys {
			if !yield(k, m.m[k]) {
				break
			}
		}
	}
}

// Keys returns an iterator to range over the keys of the map.
func (m *Map[K, V]) Keys() func(func(K) bool) {
	return func(yield func(K) bool) {
		for _, k := range m.keys {
			if !yield(k) {
				break
			}
		}
	}
}

// Values returns an iterator to range over the values of the map.
func (m *Map[K, V]) Values() func(func(V) bool) {
	return func(yield func(V) bool) {
		for _, k := range m.keys {
			if !yield(m.m[k]) {
				break
			}
		}
	}
}

// Clone creates a new map with the same keys and values.
// This is a shallow clone.
func (m *Map[K, V]) Clone() *Map[K, V] {
	r := NewMap[K, V]()
	for k, v := range m.Iter() {
		r.Store(k, v)
	}
	return r
}

// Size returns the number of elements in the map.
func (m *Map[K, V]) Size() int {
	return len(m.keys)
}
