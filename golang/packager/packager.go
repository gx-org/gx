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

	"github.com/gx-org/gx/golang/template"
	"github.com/gx-org/gx/tools/gxflag"
)

var (
	targetName   = flag.String("target_name", "", "target filename")
	targetFolder = flag.String("target_folder", "", "target folder location")

	goPackageName = flag.String("go_package_name", "", "name of the Go package")
	gxFiles       = gxflag.StringList("gx_files", "list of GX files to package")
	dependencies  = gxflag.StringList("gx_deps", "list of GX dependencies")
)

const packagerGoSource = `// Package {{.GoPackageName}} encapsulates a GX package into a Go package.
// Automatically generated from google3/third_party/gxlang/gx/golang/packager/package.go.
//
// DO NOT EDIT
package {{.GoPackageName}}

import (
	"embed"

	"github.com/gx-org/gx/build/builder"
	"github.com/gx-org/gx/build/importers/embedpkg"
{{range $dep := .Dependencies}}
	{{$dep.GoPackageName}} "{{$dep.GoPackagePath}}"
{{end}}
)

//go:embed {{range $filename := .Embed}}{{$filename}} {{end}}
var srcs embed.FS

var dependencies = []struct{
	path string
	buildFunc embedpkg.BuildFunc
}{
{{range $dep := .Dependencies}}
	{
		path: "{{$dep.GXPackage}}",
		buildFunc: {{$dep.GoPackageName}}.Build,
	},
{{end}}
}

var inputFiles = []string{
{{range $dep := .Embed}}
	"{{$dep}}",
{{end}}
}

func init() {
	for _, dep := range dependencies {
		embedpkg.RegisterPackage(dep.path, dep.buildFunc)
	}
	embedpkg.RegisterPackage("google3/{{.GXPackagePath}}/{{.GXPackageName}}", Build)
}

var _ embedpkg.BuildFunc = Build

// Build GX package.
func Build(bld *builder.Builder) (builder.Package, error) {
	return bld.BuildFiles("google3/{{.GXPackagePath}}", srcs, inputFiles)
}

// Source of the package.
func Source() embed.FS {
	return srcs
}
`

type (
	dependency struct {
		GoPackageName string
		GoPackagePath string
		GXPackage     string
	}

	packager struct {
		// GoPackageName is the name of the Go package packaging GX files together.
		GoPackageName string

		// GXPackageName is the name of the GX package.
		GXPackageName string

		// GXPackagePath is the path of the GX package being packaged.
		GXPackagePath string

		// Embed is the list of GX files constituting the GX package.
		Embed []string

		// Dependencies is the list of dependencies in the package.
		Dependencies []dependency
	}
)

func main() {
	flag.Parse()
	pkg := packager{
		GoPackageName: *goPackageName,
		GXPackageName: strings.TrimSuffix(*goPackageName, "_gx"),
	}

	// Build gx path
	if len(*gxFiles) == 0 {
		fmt.Fprint(os.Stderr, "no GX file to process")
		os.Exit(1)
	}
	pkg.GXPackagePath = (*gxFiles)[0]
	pkg.GXPackagePath = strings.TrimSpace(pkg.GXPackagePath)
	pkg.GXPackagePath = filepath.Dir(pkg.GXPackagePath)

	// Build the set of source files.
	for _, filename := range *gxFiles {
		filename := "google3/" + filename
		pkg.Embed = append(pkg.Embed, filename)
	}
	sort.Strings(pkg.Embed)

	// Build the dependencies
	sort.Strings(*dependencies)
	for i, dep := range *dependencies {
		pkg.Dependencies = append(pkg.Dependencies, dependency{
			GoPackageName: fmt.Sprintf("gxdep%d", i),
			GoPackagePath: "google3/" + dep + "_gx",
			GXPackage:     "google3/" + dep,
		})
	}

	// Generate the Go source file.
	goTarget := filepath.Join(*targetFolder, *targetName)
	if err := template.Exec(packagerGoSource, goTarget, pkg); err != nil {
		fmt.Fprintf(os.Stderr, "%+v", err)
		os.Exit(1)
	}
}
