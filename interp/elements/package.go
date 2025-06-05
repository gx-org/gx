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

package elements

import (
	"github.com/pkg/errors"
	"github.com/gx-org/gx/api/values"
	"github.com/gx-org/gx/build/fmterr"
	"github.com/gx-org/gx/build/ir"
)

// Package groups elements exported by a package.
type Package struct {
	pkg  *ir.Package
	defs map[string]Element
}

var (
	_ Element  = (*Package)(nil)
	_ Selector = (*Package)(nil)
)

// NewPackage returns a package grouping everything that a package exports.
func NewPackage(pkg *ir.Package, newFunc NewFunc) *Package {
	node := &Package{
		pkg:  pkg,
		defs: make(map[string]Element),
	}
	for _, fct := range pkg.Decls.Funcs {
		node.defs[fct.Name()] = newFunc(fct, nil)
	}
	for _, tp := range pkg.Decls.Types {
		node.defs[tp.Name()] = NewNamedType(newFunc, tp, nil)
	}
	return node
}

// Define an element in the package
// (used for consts)
func (pkg *Package) Define(name string, el Element) {
	pkg.defs[name] = el
}

// Flatten returns the package in a slice of
func (pkg *Package) Flatten() ([]Element, error) {
	return []Element{pkg}, nil
}

// Unflatten creates a GX value from the next handles available in the Unflattener.
func (pkg *Package) Unflatten(handles *Unflattener) (values.Value, error) {
	return nil, fmterr.Internal(errors.Errorf("%T does not support converting device handles into GX values", pkg))
}

// Kind of the element.
func (pkg *Package) Kind() ir.Kind {
	return ir.InvalidKind
}

// Select a member of the package.
func (pkg *Package) Select(sel SelectAt) (Element, error) {
	name := sel.Node().Stor.NameDef().Name
	el, ok := pkg.defs[name]
	if !ok {
		return nil, errors.Errorf("%s.%s undefined", pkg.pkg.Name, name)
	}
	return el, nil
}

// String returns a string representation of the node.
func (pkg *Package) String() string {
	return "package"
}
