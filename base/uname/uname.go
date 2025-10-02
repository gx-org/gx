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
	"strconv"
)

type (
	// Root generates unique numbered names given a common root.
	Root struct {
		unique       *Unique
		root         string
		alwaysSuffix bool

		nextSuffix int
	}

	// Name generated from a root.
	Name struct {
		root         *Root
		numberSuffix int
		uniqueSuffix int
	}
)

func newName(r *Root, numberSuffix int) Name {
	n := Name{
		root:         r,
		numberSuffix: numberSuffix,
	}
	s := n.String()
	for r.unique.has(s) {
		n.uniqueSuffix = r.unique.nextUniqueSuffix(s)
		s = n.String()
	}
	return n
}

// Sub returns a new sub-root from this name.
func (n Name) Sub() *Root {
	return n.root.unique.Root(n.String() + "_")
}

// NameFor returns a numbered name given a new root.
func (n Name) NameFor(r *Root) Name {
	return newName(r, n.numberSuffix)
}

// String representation of the name.
func (n Name) String() string {
	uniqueSuffix := ""
	if n.uniqueSuffix > 0 {
		uniqueSuffix = "x" + strconv.Itoa(n.uniqueSuffix)
	}
	numberSuffix := ""
	if n.root.alwaysSuffix || n.numberSuffix > 0 {
		numberSuffix = strconv.Itoa(n.numberSuffix)
	}
	return fmt.Sprintf("%s%s%s", n.root.root, numberSuffix, uniqueSuffix)
}

// Next generates the next name.
func (r *Root) Next() Name {
	n := newName(r, r.nextSuffix)
	r.nextSuffix++
	return n
}

// Root returns the root as a string.
// Note that this may not be unique.
func (r *Root) Root() string {
	return r.root
}

// Unique generates unique names.
type Unique struct {
	done  map[string]int
	roots map[string]*Root
}

// New name generator.
func New() *Unique {
	return &Unique{
		done:  make(map[string]int),
		roots: make(map[string]*Root),
	}
}

func (n *Unique) newRoot(root string, alwaysSuffix bool) *Root {
	r := &Root{
		unique:       n,
		root:         root,
		alwaysSuffix: alwaysSuffix,
		nextSuffix:   0,
	}
	n.roots[root] = r
	return r
}

// Register a name as not available.
func (n *Unique) Register(name string) {
	n.done[name] = 1
}

// Root returns a name generator for a given root.
func (n *Unique) Root(root string) *Root {
	return n.newRoot(root, true)
}

// Name returns a unique name given a root.
func (n *Unique) Name(root string) string {
	r, ok := n.roots[root]
	if !ok {
		r = n.newRoot(root, false)
	}
	return r.Next().String()
}

func (n *Unique) has(s string) bool {
	_, ok := n.done[s]
	return ok
}

func (n *Unique) nextUniqueSuffix(s string) int {
	next := n.done[s]
	n.done[s] = next + 1
	return next
}

// Ident returns an ident with a unique name.
func (n *Unique) Ident(id *ast.Ident) *ast.Ident {
	name := n.Name(id.Name)
	return &ast.Ident{Name: name}
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
