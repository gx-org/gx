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

// Package base provides a base runtime package for testing.
package base

import (
	"strings"

	"github.com/gx-org/gx/build/importers"
	"github.com/gx-org/gx/build/ir"
	irh "github.com/gx-org/gx/build/ir/irhelper"
	"github.com/gx-org/gx/interp/elements"
)

type base struct {
}

// New importer for the runtime base package.
func New() importers.Importer {
	return &base{}
}

// Support checks if the importer supports the import path given its prefix.
func (b *base) Support(path string) bool {
	return strings.HasPrefix(path, "rtpkgbase")
}

type basePackage struct {
	pkg *ir.Package
}

// Import a path.
func (b *base) Import(bld importers.Builder, path string) (importers.Package, error) {
	p := &basePackage{
		pkg: &ir.Package{},
	}
	p.pkg.Runtime = p.runtime
	return p, nil
}

// IR returns the current package intermediate representation.
func (b *basePackage) IR() *ir.Package {
	return b.pkg
}

func (b *basePackage) runtime() (ir.PackageElement, error) {
	return b, nil
}

func (b *basePackage) Type() ir.Type {
	return ir.PackageType()
}

func (b *basePackage) Package() *ir.Package {
	return b.pkg
}

func (b *basePackage) Find(name string) (ir.Element, error) {
	return elements.NewFunc(&ir.FuncDecl{
		FFile: &ir.File{
			Package: b.pkg,
		},
		FType: ir.UnknownFType(),
		Body: irh.SingleReturn(
			irh.IntNumberAs(5, ir.IntType()),
		),
	}, nil), nil
}
