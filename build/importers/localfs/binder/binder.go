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

// Package binder provides strings for GX binder to generate the code
// to load GX packages from the local filesystem.
package binder

import (
	"fmt"
	"path/filepath"

	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/golang/binder/gobindings"
)

// GenImports generates the source code to build GX packages
// using the local filesystem.
type GenImports struct{}

var _ gobindings.DependenciesBuildCall = (*GenImports)(nil)

// SourceImport generates the import when the GX source needs to be imported
// from a Go module.
func (GenImports) SourceImport(pkg *ir.Package) string {
	return fmt.Sprintf(`_ "%s"`, pkg.Path)
}

// DependencyImport generates the import to insert in the package source code.
func (GenImports) DependencyImport(pkg *ir.Package) string {
	return fmt.Sprintf("%s/%s/%s_go_gx", pkg.Path, pkg.Name.Name, pkg.Name.Name)
}

// StdlibDependencyImport returns the path to the Go bindings of a GX standard library given
// the standard library path.
func (GenImports) StdlibDependencyImport(path string) string {
	dir, name := filepath.Split(path)
	return fmt.Sprintf("github.com/gx-org/gx/stdlib/bindings/go/%s%s_go_gx", dir, name)
}
