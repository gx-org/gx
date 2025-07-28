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

// Package genstdlib provides helpers to build the bindings of the standard library packages.
package genstdlib

import (
	"fmt"
	"io"

	"github.com/gx-org/gx/build/builder"
	"github.com/gx-org/gx/build/importers"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/golang/binder/gobindings"
	"github.com/gx-org/gx/stdlib"
)

type stdlibImporter struct{}

func (stdlibImporter) SourceImport(*ir.Package) string { return "" }

func (stdlibImporter) StdlibDependencyImport(stdlibPath string) string { return "" }

func (stdlibImporter) DependencyImport(*ir.Package) string { return "" }

var importer = stdlibImporter{}

// BuildAll builds the bindings for the standard library package.
func BuildAll(newWriter func(*ir.Package) (io.WriteCloser, error)) error {
	libs := stdlib.Importer(nil)
	bld := builder.New(importers.NewCacheLoader(libs))
	for _, path := range libs.Paths() {
		bpkg, err := bld.Build(path)
		if err != nil {
			return err
		}
		pkg := bpkg.IR()
		w, err := newWriter(pkg)
		if err != nil {
			return err
		}
		defer w.Close()
		bnd, err := gobindings.NewWithImporter(importer, pkg)
		if err != nil {
			return fmt.Errorf("cannot create binder for %s: %w", pkg.Name.Name, err)
		}
		if err := bnd.Files()[0].WriteBindings(w); err != nil {
			return fmt.Errorf("cannot generate source for %s: %w", pkg.Name.Name, err)
		}
	}
	return nil
}
