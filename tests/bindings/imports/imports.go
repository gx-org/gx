// Package imports encapsulates GX source files
// into a Go package.
//
// Automatically generated from google3/third_party/gxlang/gx/golang/packager/package.go.
//
// DO NOT EDIT
package imports

import (
	"embed"

	"github.com/gx-org/gx/build/importers"
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
func Build(bld importers.Builder) (importers.Package, error) {
	return bld.BuildFiles("github.com/gx-org/gx/tests/bindings", "imports", srcs, inputFiles)
}
