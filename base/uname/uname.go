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

// Package uname provides unique names.
package uname

import "fmt"

// Unique generates unique names.
type Unique struct {
	names map[string]int
}

// New name generator.
func New() *Unique {
	return &Unique{names: make(map[string]int)}
}

// Name returns a unique name given a desired base name.
// If the base name is available, it is returned directly. Else, a unique suffix is appended.
func (n *Unique) Name(root string) string {
	nextIndex, ok := n.names[root]
	if !ok {
		n.names[root] = 1
		return root
	}
	name := fmt.Sprintf("%s%d", root, nextIndex)
	n.names[root] = nextIndex + 1
	return name
}
