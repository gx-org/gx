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
	"io"
	"os"
	"path/filepath"

	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/stdlib/bindings/go/genstdlib"
)

var (
	outputFolder        = flag.String("output_folder", "/tmp", "output path where the bindings are generated")
	packageFolderSuffix = flag.String("package_folder_suffix", "", "Suffix to append to the package folder")
)

func newSourceFile(pkg *ir.Package) (io.WriteCloser, error) {
	bindingsPath := filepath.Join(*outputFolder, pkg.FullName()) + *packageFolderSuffix
	if err := os.MkdirAll(bindingsPath, 0777); err != nil {
		return nil, fmt.Errorf("cannot create package path %s: %v", bindingsPath, err)
	}
	srcFilePath := filepath.Join(bindingsPath, pkg.Name.Name+"_go_gx.go")
	srcFile, err := os.Create(srcFilePath)
	if err != nil {
		return nil, fmt.Errorf("cannot create file %s: %v", srcFilePath, err)
	}
	return srcFile, nil
}

func main() {
	flag.Parse()
	if len(*outputFolder) == 0 {
		fmt.Fprintf(os.Stderr, "no output folder specified: please use --output_folder")
		os.Exit(1)
	}
	if err := genstdlib.BuildAll(newSourceFile); err != nil {
		fmt.Fprintf(os.Stderr, "%+v", err)
		os.Exit(1)
	}
}
