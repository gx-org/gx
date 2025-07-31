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

// Package goembed generates a Go source to embed GX source file.
package goembed

import (
	"fmt"
	"io"

	"text/template"
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
	embedpkg.RegisterPackage("{{.GXPackage}}", Build)
}

var _ embedpkg.BuildFunc = Build

// Build GX package.
func Build(bld *builder.Builder) (builder.Package, error) {
	return bld.BuildFiles("{{.GXPackagePath}}", "{{.GXPackageName}}", srcs, inputFiles)
}
`

// PackageInfo defines all the values to fill in the template.
type PackageInfo interface {
	// GoPackageName is the name of the Go package packaging GX files together.
	GoPackageName() string

	// GXPackageName is the name of the GX package.
	GXPackageName() string

	// GXPackagePath is the path to the GX package.
	GXPackagePath() string

	// GXPackage is the path to the GX package.
	GXPackage() string

	// Embed is the list of GX files constituting the GX package.
	Embed() []string

	// Dependencies is the list of dependencies in the package.
	Dependencies() []string

	// TargetFile returns the file in which the data needs to be written.
	TargetFile() string
}

// Write the source to the writer.
func Write(w io.Writer, info PackageInfo) error {
	tpl, err := template.New("").Parse(packagerGoSource)
	if err != nil {
		return fmt.Errorf("cannot parse template source: %v", err)
	}
	return tpl.Execute(w, info)
}
