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

// Package iter provides common iterators.
package iter

// All iterates over the element of multiple slices.
func All[T any](slices ...[]T) func(yield func(T) bool) {
	return func(yield func(T) bool) {
		for _, slice := range slices {
			for _, el := range slice {
				if !yield(el) {
					return
				}
			}
		}
	}
}

// Filter iterates over the element of multiple slices
// and excludes elements for which the filter returns false.
func Filter[T any](f func(T) bool, slices ...[]T) func(yield func(T) bool) {
	return func(yield func(T) bool) {
		for el := range All(slices...) {
			if !f(el) {
				continue
			}
			if !yield(el) {
				return
			}
		}
	}
}
