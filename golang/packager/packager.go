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

// Command packager packages GX files into a Go library.
// It takes a list of GX files from a given GX package
// and creates a Go package, named like the GX package
// with a "_gx" suffix, with all the GX files embedded in.
//
// The Go generated package also provides Build functions to
// build an intermediate representation of the GX package
// given a builder.
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"flag"
	"github.com/gx-org/gx/golang/packager/pkginfo"
	"github.com/gx-org/gx/golang/packager/goembed"
)

func main() {
	flag.Parse()
	pkgInfo, err := pkginfo.BuildPackageInfo()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%+v\n", err)
		os.Exit(1)
	}
	targetPath := pkgInfo.TargetFile()
	if err := os.MkdirAll(filepath.Dir(targetPath), os.ModePerm); err != nil {
		fmt.Fprintf(os.Stderr, "%+v\n", err)
		os.Exit(1)
	}
	w, err := os.Create(targetPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%+v\n", err)
		os.Exit(1)
	}
	defer w.Close()
	if err := goembed.Write(w, pkgInfo); err != nil {
		fmt.Fprintf(os.Stderr, "%+v\n", err)
		os.Exit(1)
	}
}
