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

// Package irb keeps track of IRs being built
// and how to add them to the package declarations.
package irb

import "github.com/gx-org/gx/build/ir"

type (
	// Node builds an IR node.
	Node[T any] interface {
		Build(*Builder[T]) (ir.IR, bool)
	}

	// NodeF builds a node.
	NodeF[T any] func(*Builder[T]) (ir.IR, bool)

	// Builder are IR nodes already built as well as their declarators.
	Builder[T any] struct {
		scope T
		built map[Node[T]]ir.IR
		pkg   *ir.Package

		declarators []Declarator
	}

	// Declarator declares a node in the IR package declarations.
	Declarator func(*ir.Declarations)
)

// Build the node.
func (f NodeF[T]) Build(ibld *Builder[T]) (ir.IR, bool) {
	return f(ibld)
}

// New returns a new
func New[T any](scope T, pkg *ir.Package) *Builder[T] {
	return &Builder[T]{
		scope: scope,
		built: make(map[Node[T]]ir.IR),
		pkg:   pkg,
	}
}

// Pkg returns the IR package.
func (ibld *Builder[T]) Pkg() *ir.Package {
	return ibld.pkg
}

// Register appends a declarator to declare a node in the package declarations.
func (ibld *Builder[T]) Register(decl Declarator) {
	ibld.declarators = append(ibld.declarators, decl)
}

// Scope returns the root scope used to build the IR nodes.
func (ibld *Builder[T]) Scope() T {
	return ibld.scope
}

// Cache return the IR of a node that has already been built.
func (ibld *Builder[T]) Cache(bld Node[T]) (ir.IR, bool) {
	node, ok := ibld.built[bld]
	return node, ok
}

// Set the IR of a given builder node.
func (ibld *Builder[T]) Set(bld Node[T], n ir.IR) {
	ibld.built[bld] = n
}

// Build an IR node or returns the node if it has already been built.
func (ibld *Builder[T]) Build(bld Node[T]) (ir.IR, bool) {
	node, ok := ibld.Cache(bld)
	if ok {
		return node, true
	}
	node, ok = bld.Build(ibld)
	ibld.built[bld] = node
	return node, ok
}

// Decls returns the IR declarations of everything that has been declared
// until this point.
func (ibld *Builder[T]) Decls() *ir.Declarations {
	decls := &ir.Declarations{}
	for _, decl := range ibld.declarators {
		decl(decls)
	}
	return decls
}
