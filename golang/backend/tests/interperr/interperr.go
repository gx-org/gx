// Package interperr encapsulates GX source files
// into a Go package.
//
// Automatically generated from google3/third_party/gxlang/gx/golang/packager/package.go.
//
// DO NOT EDIT
package interperr

import (
	"embed"

	"github.com/gx-org/gx/build/importers"
	"github.com/gx-org/gx/build/importers/embedpkg"

)

//go:embed interperr.gx 
var srcs embed.FS

var inputFiles = []string{
"interperr.gx",
}

func init() {
	embedpkg.RegisterPackage("github.com/gx-org/gx/golang/backend/tests/interperr", Build)
}

var _ embedpkg.BuildFunc = Build

// Build GX package.
func Build(bld importers.Builder) (importers.Package, error) {
	return bld.BuildFiles("github.com/gx-org/gx/golang/backend/tests", "interperr", srcs, inputFiles)
}
