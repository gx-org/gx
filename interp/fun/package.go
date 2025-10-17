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

var (
	_ ir.Element        = (*Package)(nil)
	_ elements.Selector = (*Package)(nil)
)

// NewPackage returns a package grouping everything that a package exports.
func NewPackage(pkg *ir.Package, defs scope.Scope[ir.Element]) *Package {
	return &Package{pkg: pkg, defs: defs}
}

// Type of the element.
func (pkg *Package) Type() ir.Type {
	return ir.PackageType()
}

// Select a member of the package.
func (pkg *Package) Select(expr *ir.SelectorExpr) (ir.Element, error) {
	name := expr.Stor.NameDef().Name
	el, ok := pkg.defs.Find(name)
	if !ok {
		return nil, errors.Errorf("%s.%s undefined", pkg.pkg.Name, name)
	}
	return el, nil
}

// String returns a string representation of the node.
func (pkg *Package) String() string {
	return "package"
}
