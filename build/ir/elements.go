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

package ir

type (

	// Element is a value returned by the evaluator.
	Element interface {
		Type() Type
	}

	// PackageElement is an element encapsulating a package.
	PackageElement interface {
		Element
		Package() *Package
	}

	// StorageElement represents an element able to store values.
	StorageElement interface {
		Element
		WithStore
	}

	// WithLength is an element with a length (arrays, slices, ...)
	WithLength interface {
		Element
		// Length returns the evaluation of the len built-in.
		Length(ev Evaluator) (int, error)
	}

	// FixedSlice is a slice where the number of elements is known.
	FixedSlice interface {
		WithLength
		Len() int
		Elements() []Element
	}
)
