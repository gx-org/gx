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

package state

import (
	"github.com/pkg/errors"
	"github.com/gx-org/gx/build/fmterr"
	"github.com/gx-org/gx/build/ir"
)

// Package groups elements exported by a package.
type Package struct {
	state  *State
	errFmt fmterr.Pos
	pkg    *ir.Package
	funcs  map[ir.Func]*Func
	types  map[*ir.NamedType]*NamedType
}

var (
	_ Element        = (*Package)(nil)
	_ MethodSelector = (*Package)(nil)
)

// Package returns a package grouping everything that a package exports.
func (g *State) Package(errFmt fmterr.Pos, pkg *ir.Package) *Package {
	node := &Package{
		errFmt: errFmt,
		pkg:    pkg,
	}
	node.funcs = make(map[ir.Func]*Func, len(pkg.Funcs))
	for _, fct := range pkg.Funcs {
		node.funcs[fct] = g.Func(fct, nil)
	}
	node.types = make(map[*ir.NamedType]*NamedType)
	for _, tp := range pkg.Types {
		node.types[tp] = g.NamedType(errFmt.FileSet, tp)
	}
	return node
}

// Flatten returns the package in a slice of elements.
func (pkg *Package) Flatten() ([]Element, error) {
	return []Element{pkg}, nil
}

// ErrPos returns the error formatter for the position of the token representing the node in the graph.
func (pkg *Package) ErrPos() fmterr.Pos {
	return pkg.errFmt
}

// Type of a package.
func (pkg *Package) Type() ir.Type {
	return nil
}

// State owning the element.
func (pkg *Package) State() *State {
	return pkg.state
}

// SelectMethod returns a functions given an index.
func (pkg *Package) SelectMethod(fn ir.Func) (*Func, error) {
	fun, ok := pkg.funcs[fn]
	if !ok {
		return nil, errors.Errorf("cannot find function %q pointer in package %s", fn.Name(), pkg.pkg.FullName())
	}
	return fun, nil
}

// SelectType returns a functions given an index.
func (pkg *Package) SelectType(tp *ir.NamedType) Element {
	return pkg.types[tp]
}

// String returns a string representation of the node.
func (pkg *Package) String() string {
	return "package"
}
