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

// Package imports encapsulates GX source files
// into a Go package.
//
// Automatically generated from google3/third_party/gxlang/gx/golang/packager/package.go.
//
// DO NOT EDIT
package imports

import (
	"embed"

	"github.com/gx-org/gx/build/builder"
	"github.com/gx-org/gx/build/importers/embedpkg"

	_ "github.com/gx-org/gx/tests/bindings/basic"
)

//go:embed imports.gx 
var srcs embed.FS

var inputFiles = []string{
"imports.gx",
}

func init() {
	embedpkg.RegisterPackage("github.com/gx-org/gx/tests/bindings/imports", Build)
}

var _ embedpkg.BuildFunc = Build

// Build GX package.
func Build(bld *builder.Builder) (builder.Package, error) {
	return bld.BuildFiles("github.com/gx-org/gx/tests/bindings", "imports", srcs, inputFiles)
}
