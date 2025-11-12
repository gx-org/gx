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

package fun

import (
	"github.com/pkg/errors"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/base/scope"
	"github.com/gx-org/gx/interp/elements"
)

// Package groups elements exported by a package.
type Package struct {
	pkg  *ir.Package
	defs scope.Scope[ir.Element]
}

var _ ir.PackageElement = (*Package)(nil)

// NewPackage returns a package grouping everything that a package exports.
func NewPackage(pkg *ir.Package, defs scope.Scope[ir.Element]) *Package {
	return &Package{pkg: pkg, defs: defs}
}

// Type of the element.
func (pkg *Package) Type() ir.Type {
	return ir.PackageType()
}

// Package encapsulated.
func (pkg *Package) Package() *ir.Package {
	return pkg.pkg
}

// String returns a string representation of the node.
func (pkg *Package) String() string {
	return "package"
}

// Import is an element representing an import in a file.
type Import struct {
	imp *ir.ImportDecl
	pkg *Package
}

var (
	_ ir.StorageElement = (*Import)(nil)
	_ elements.Selector = (*Import)(nil)
)

// NewImport returns a new import element for a given package.
func NewImport(imp *ir.ImportDecl, pkg ir.PackageElement) ir.Element {
	return &Import{imp: imp, pkg: pkg.(*Package)}
}

// Type of the element.
func (imp *Import) Type() ir.Type {
	return ir.PackageType()
}

// Store returns the IR storage.
func (imp *Import) Store() ir.Storage {
	return imp.imp
}

// Select a member of the package.
func (imp *Import) Select(expr *ir.SelectorExpr) (ir.Element, error) {
	name := expr.Stor.NameDef().Name
	el, ok := imp.pkg.defs.Find(name)
	if !ok {
		return nil, errors.Errorf("%s.%s undefined", imp.pkg.pkg.Name, name)
	}
	return el, nil
}
