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

// Utility genbind generates bindings for GX packages.
package main

import (
	"flag"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"

	"github.com/gx-org/gx/build/builder"
	"github.com/gx-org/gx/build/importers"
	"github.com/gx-org/gx/build/importers/localfs"
	"github.com/gx-org/gx/build/module"
	"github.com/gx-org/gx/golang/binder"
	"github.com/gx-org/gx/stdlib"
)

var (
	targetFolder = flag.String("target_folder", "", "target location")
	targetName   = flag.String("target_name", "", "name of the file")
	language     = flag.String("language", "go", "Language for which to generate the bindings")
	gxPackage    = flag.String("gx_package", "", "GX package to generate the bindings for")
)

func exit(format string, a ...any) {
	fmt.Fprintf(os.Stderr, format, a...)
	fmt.Fprintln(os.Stderr)
	os.Exit(1)
}

func adjustFlags(mod *module.Module) error {
	_, name := filepath.Split(*gxPackage)
	bindingsName := name + "_go_gx"
	if *targetName == "" {
		*targetName = bindingsName
	}
	if *targetFolder != "" {
		return nil
	}
	folder, err := mod.ImportToOSPath(*gxPackage)
	if err != nil {
		return err
	}
	*targetFolder = filepath.Join(folder, bindingsName)
	return nil
}

func main() {
	flag.Parse()
	localImporter, err := localfs.New()
	if err != nil {
		exit("cannot create local importer: %v", err)
	}
	if err := adjustFlags(localImporter.Module()); err != nil {
		exit("%v", err)
	}
	fullFilePath := filepath.Join(*targetFolder, *targetName)
	if err := os.MkdirAll(filepath.Dir(fullFilePath), os.ModePerm); err != nil {
		exit("cannot create target directory: %v", err)
	}
	bndConstructor, ok := binder.Binders[*language]
	if !ok {
		exit("cannot create bindings for language %q: no binder available. Available binders are %v", *language, slices.Collect(maps.Keys(binder.Binders)))
	}
	bld := builder.New(importers.NewCacheLoader(
		stdlib.Importer(nil),
		localImporter,
	))
	pkg, err := bld.Build(*gxPackage)
	if err != nil {
		exit("%+v", err)
	}
	bnd, err := bndConstructor(pkg.IR())
	if err != nil {
		exit("%+v", err)
	}
	for _, file := range bnd.Files() {
		f, err := os.Create(fullFilePath + file.Extension())
		if err != nil {
			exit("cannot create target file: %v", err)
		}
		defer f.Close()

		if err := file.WriteBindings(f); err != nil {
			exit("%+v", err)
		}
	}
}
