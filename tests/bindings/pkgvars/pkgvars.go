// Package pkgvars encapsulates GX source files
// into a Go package.
//
// Automatically generated from google3/third_party/gxlang/gx/golang/packager/package.go.
//
// DO NOT EDIT
package pkgvars

import (
	"embed"

	"github.com/gx-org/gx/build/builder"
	"github.com/gx-org/gx/build/importers/embedpkg"

)

//go:embed pkgvars.gx 
var srcs embed.FS

var inputFiles = []string{
"pkgvars.gx",
}

func init() {
	embedpkg.RegisterPackage("github.com/gx-org/gx/tests/bindings/pkgvars", Build)
}

var _ embedpkg.BuildFunc = Build

// Build GX package.
func Build(bld *builder.Builder) (builder.Package, error) {
	return bld.BuildFiles("github.com/gx-org/gx/tests/bindings", "pkgvars", srcs, inputFiles)
}
