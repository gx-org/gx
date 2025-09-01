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

// Package handle wrap/unwrap Go pointers for C.
package handle

import (
	"fmt"
	"runtime"
	"strings"
	"sync/atomic"

	"github.com/gx-org/gx/base/sync"
)

// Handle to a Go object.
type Handle uintptr

// Memory visible outside Go.
var (
	// handles holds the mapping between Handle and Go values. Implementation is borrowed from
	// "runtime/cgo"; we copy it here to allow inspecting the contents of this map.
	handles   = sync.Map[Handle, any]{}
	handleIdx = atomic.Uintptr{}
)

// Wrap converts a Go value to a cgx_handle.
//
// Handles must be unwrapped with the Unwrap() function using the same type T.
func Wrap[T comparable](v T) Handle {
	var zero T
	if v == zero {
		return 0
	}

	h := Handle(handleIdx.Add(1))
	if h == 0 {
		panic("cgx: ran out of handle space")
	}

	handles.Store(h, v)
	return Handle(h)
}

// Unwrap converts a handle returned by Wrap() into the original Go value.
//
// Unwrap() must be called with the same type T used in the original Wrap() call.
func Unwrap[T any](h Handle) T {
	if h == 0 {
		var zero T
		return zero
	}
	return handles.Load(h).(T)
}

// Release deletes a cgx handle.
//
// The handle must not be used (either through Unwrap or Release) after deletion.
func Release(h Handle) {
	if h != 0 {
		_, ok := handles.LoadAndDelete(h)
		if !ok {
			panic(fmt.Sprintf("cgx: deleting invalid handle %v", h))
		}
	}
	if handles.Empty() {
		// Force garbage collection when the final CGX reference is released; this is mostly for the
		// benefit of tests which may have leak detection enabled.
		runtime.GC()
	}
}

// WrapSlice wraps all the element of a slice.
func WrapSlice[T comparable](vs []T) []Handle {
	if len(vs) == 0 {
		return nil
	}
	refs := make([]Handle, len(vs))
	for i, v := range vs {
		refs[i] = Wrap[T](v)
	}
	return refs
}

// Count returns the total number of active handles and pinned slices.
func Count() int {
	return handles.Size() + pinners.Size()
}

// Dump returns a string representation of all existing handles and pinned slices.
func Dump() string {
	s := strings.Builder{}
	for h, v := range handles.Iter() {
		fmt.Fprintf(&s, "%T handle: %v\n", v, h)
	}
	for ptr := range pinners.Iter() {
		fmt.Fprintf(&s, "slice: %p\n", ptr)
	}
	return s.String()
}
