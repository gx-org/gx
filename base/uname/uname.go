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

import (
	"fmt"
	"go/ast"
)

// Unique generates unique names.
type Unique struct {
	names map[string]int
}

// New name generator.
func New() *Unique {
	return &Unique{names: make(map[string]int)}
}

func (n *Unique) name(root string, alwaysSuffix bool) string {
	nextIndex, ok := n.names[root]
	if !ok && !alwaysSuffix {
		n.names[root] = 1
		return root
	}
	name := fmt.Sprintf("%s%d", root, nextIndex)
	n.names[root] = nextIndex + 1
	return name
}

// Name returns a unique name given a desired base name.
// If the base name is available, it is returned directly. Else, a unique suffix is appended.
func (n *Unique) Name(root string) string {
	return n.name(root, false)
}

// NameNumbered returns a unique name given a base name,
// always with a number suffix in the name even if the root
// has not been queried before.
func (n *Unique) NameNumbered(root string) string {
	return n.name(root, true)
}

// Ident returns an ident with a unique name.
func (n *Unique) Ident(id *ast.Ident) *ast.Ident {
	return &ast.Ident{Name: n.Name(id.Name)}
}

// Default returns a default value if a name is empty or equal to "_".
func Default(name, def string) string {
	if name == "" || name == "_" {
		return def
	}
	return name
}

// DefaultIdent returns a default value if the name in the ident is empty or equal to "_".
func DefaultIdent(id *ast.Ident, def string) *ast.Ident {
	return &ast.Ident{Name: Default(id.Name, def)}
}
