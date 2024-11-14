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
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/gx-org/gx/golang/packager/pkginfo"
	"github.com/gx-org/gx/golang/template"
	"github.com/gx-org/gx/tools/gxflag"
)

var (
	targetName   = flag.String("target_name", "", "target filename")
	targetFolder = flag.String("target_folder", "", "target folder location")

	gxImportPath  = flag.String("gx_import_path", "", "import path of the GX package")
	goPackageName = flag.String("go_package_name", "", "name of the Go package")
	gxFiles       = gxflag.StringList("gx_files", "list of GX files to package")
	dependencies  = gxflag.StringList("gx_deps", "list of GX dependencies")

	gxPackageModule = flag.String("gx_package_module", "", "gx package path within the module")
)

const packagerGoSource = `// Package {{.GoPackageName}} encapsulates GX source files
// into a Go package.
//
// Automatically generated from google3/third_party/gxlang/gx/golang/packager/package.go.
//
// DO NOT EDIT
package {{.GoPackageName}}

import (
	"embed"

	"github.com/gx-org/gx/build/builder"
	"github.com/gx-org/gx/build/importers/embedpkg"
{{range $dep := .Dependencies}}
	_ "{{$dep}}"
{{- end}}
)

//go:embed {{range $filename := .Embed}}{{$filename}} {{end}}
var srcs embed.FS

var inputFiles = []string{
{{range $dep := .Embed -}}
	"{{$dep}}",
{{- end}}
}

func init() {
	embedpkg.RegisterPackage("{{.GXPackagePath}}/{{.GXPackageName}}", Build)
}

var _ embedpkg.BuildFunc = Build

// Build GX package.
func Build(bld *builder.Builder) (builder.Package, error) {
	return bld.BuildFiles("{{.GXPackagePath}}", "{{.GXPackageName}}", srcs, inputFiles)
}
`

type packager struct {
	// GoPackageName is the name of the Go package packaging GX files together.
	GoPackageName string

	// GXPackageName is the name of the GX package.
	GXPackageName string

	// GXPackagePath is the path to the GX package.
	GXPackagePath string

	// Embed is the list of GX files constituting the GX package.
	Embed []string

	// Dependencies is the list of dependencies in the package.
	Dependencies []string
}

func setEmptyFlagsFromPackageModule() error {
	pkgInfo, err := pkginfo.Load(*gxPackageModule)
	if err != nil {
		return fmt.Errorf("cannot load package %s: %v", *gxPackageModule, err)
	}
	if *targetName == "" {
		*targetName = pkgInfo.TargetFileName()
	}
	if *targetFolder == "" {
		*targetFolder = pkgInfo.TargetFolder()
	}
	if *goPackageName == "" {
		*goPackageName = pkgInfo.GoPackageName()
	}
	if *gxImportPath == "" {
		*gxImportPath = *gxPackageModule
	}
	if len(*gxFiles) == 0 {
		*gxFiles, err = pkgInfo.SourceFiles()
		if err != nil {
			return fmt.Errorf("cannot list source files: %v", err)
		}
	}
	if len(*dependencies) == 0 {
		*dependencies, err = pkgInfo.Dependencies()
		if err != nil {
			return fmt.Errorf("cannot list dependencies: %v", err)
		}
	}
	return nil
}

func main() {
	flag.Parse()
	if *gxPackageModule != "" {
		if err := setEmptyFlagsFromPackageModule(); err != nil {
			fmt.Fprintf(os.Stderr, "%+v\n", err)
			os.Exit(1)
		}
	}
	gxPackagePath, gxPackageName := filepath.Split(*gxImportPath)
	gxPackagePath = strings.TrimSuffix(gxPackagePath, "/")
	pkg := packager{
		GoPackageName: *goPackageName,
		GXPackageName: gxPackageName,
		GXPackagePath: gxPackagePath,
		Dependencies:  *dependencies,
	}

	// Build gx path
	if len(*gxFiles) == 0 {
		fmt.Fprint(os.Stderr, "no GX file to process\n")
		os.Exit(1)
	}

	// Build the set of source files.
	for _, filename := range *gxFiles {
		pkg.Embed = append(pkg.Embed, filename)
	}
	sort.Strings(pkg.Embed)

	// Create the target folder if necessary.
	if err := os.MkdirAll(*targetFolder, os.ModePerm); err != nil {
		fmt.Fprintf(os.Stderr, "%+v\n", err)
		os.Exit(1)
	}

	// Generate the Go source file.
	goTarget := filepath.Join(*targetFolder, *targetName)
	if err := template.Exec(packagerGoSource, goTarget, pkg); err != nil {
		fmt.Fprintf(os.Stderr, "%+v\n", err)
		os.Exit(1)
	}
}
