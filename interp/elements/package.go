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

package elements

import (
	"github.com/pkg/errors"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/base/scope"
	"github.com/gx-org/gx/interp/engine"
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

// Find an element given its name.
func (pkg *Package) Find(name string) (ir.Element, error) {
	el, _ := pkg.defs.Find(name)
	return el, nil
}

// Package encapsulated.
func (pkg *Package) Package() *ir.Package {
	return pkg.pkg
}

type runtimePackage struct {
	pkg *ir.Package
}

// RuntimePackage creates a proxy runtime package used at compile time to disable the compiler checks.
func RuntimePackage(pkg *ir.Package) (ir.PackageElement, error) {
	return &runtimePackage{pkg: pkg}, nil
}

func (pkg *runtimePackage) Type() ir.Type {
	return ir.PackageType()
}

func (pkg *runtimePackage) Find(name string) (ir.Element, error) {
	return Unknown(), nil
}

func (pkg *runtimePackage) Package() *ir.Package {
	return pkg.pkg
}

// Import is an element representing an import in a file.
type Import struct {
	pkg  ir.PackageElement
	stor ir.Storage
}

var (
	_ ir.StorageElement = (*Import)(nil)
	_ engine.Selector   = (*Import)(nil)
)

// NewImport returns a new import element for a given package.
func NewImport(pkg ir.PackageElement, stor ir.Storage) ir.Element {
	return &Import{pkg: pkg, stor: stor}
}

// Type of the element.
func (imp *Import) Type() ir.Type {
	return ir.PackageType()
}

// Store returns the IR storage.
func (imp *Import) Store() ir.Storage {
	return imp.stor
}

// Select a member of the package.
func (imp *Import) Select(env *engine.Env, expr *ir.SelectorExpr) (ir.Element, error) {
	name := expr.Stor.NameDef().Name
	el, err := imp.pkg.Find(name)
	if err != nil {
		return nil, err
	}
	if el == nil {
		pkgName := imp.pkg.Package().Path()
		if imp.stor != nil {
			pkgName = imp.stor.NameDef().Name
		}
		return nil, errors.Errorf("%s.%s undefined", pkgName, name)
	}
	return el, nil
}
