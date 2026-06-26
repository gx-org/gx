// Package unexported encapsulates GX source files
// into a Go package.
//
// Automatically generated from google3/third_party/gxlang/gx/golang/packager/package.go.
//
// DO NOT EDIT
package unexported

import (
	"embed"

	"github.com/gx-org/gx/build/importers"
	"github.com/gx-org/gx/build/importers/embedpkg"

)

//go:embed unexported.gx 
var srcs embed.FS

var inputFiles = []string{
"unexported.gx",
}

func init() {
	embedpkg.RegisterPackage("github.com/gx-org/gx/tests/bindings/unexported", Build)
}

var _ embedpkg.BuildFunc = Build

// Build GX package.
func Build(bld importers.Builder) (importers.Package, error) {
	return bld.BuildFiles("github.com/gx-org/gx/tests/bindings", "unexported", srcs, inputFiles)
}
