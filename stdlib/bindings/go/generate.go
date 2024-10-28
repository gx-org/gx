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

// Generate the bindings of the GX standard library for Go.
package main

//go:generate go run generate.go --output_folder=. --package_folder_suffix=_go_gx

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/gx-org/gx/build/builder"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/golang/binder/gobindings"
	"github.com/gx-org/gx/stdlib"
)

var (
	outputFolder        = flag.String("output_folder", "/tmp", "output path where the bindings are generated")
	packageFolderSuffix = flag.String("package_folder_suffix", "", "Suffix to append to the package folder")
)

type stdlibImporter struct{}

func (stdlibImporter) SourceImport(*ir.Package) string { return "" }

func (stdlibImporter) StdlibDependencyImport(stdlibPath string) string { return "" }

func (stdlibImporter) DependencyImport(path string) string { return "" }

func (stdlibImporter) CallBuild(pkg *ir.Package) (string, error) {
	return fmt.Sprintf(`irPackage, err := rtm.Builder().Build("%s")
	if err != nil {
		return nil, err
	}
`, pkg.FullName()), nil
}

var importer = stdlibImporter{}

func generateBindings(pkg *ir.Package) error {
	bindingsPath := filepath.Join(*outputFolder, pkg.FullName()) + *packageFolderSuffix
	if err := os.MkdirAll(bindingsPath, 0777); err != nil {
		return fmt.Errorf("cannot create package path %s: %v", bindingsPath, err)
	}
	srcFilePath := filepath.Join(bindingsPath, pkg.Name.Name+"_go_gx.go")
	srcFile, err := os.Create(srcFilePath)
	if err != nil {
		return fmt.Errorf("cannot create file %s: %v", srcFilePath, err)
	}
	defer srcFile.Close()
	if err := gobindings.Generate(importer, srcFile, pkg); err != nil {
		return fmt.Errorf("cannot generate source for %s: %v", pkg.Name.Name, err)
	}
	return nil
}

func buildAll() error {
	libs := stdlib.Importer(nil)
	bld := builder.New([]builder.Importer{libs})
	for _, path := range libs.Paths() {
		pkg, err := bld.Build(path)
		if err != nil {
			return err
		}
		if err := generateBindings(pkg); err != nil {
			return err
		}
	}
	return nil
}

func main() {
	flag.Parse()
	if len(*outputFolder) == 0 {
		fmt.Fprintf(os.Stderr, "no output folder specified: please use --output_folder")
		os.Exit(1)
	}
	if err := buildAll(); err != nil {
		fmt.Fprintf(os.Stderr, "%+v", err)
		os.Exit(1)
	}
}
