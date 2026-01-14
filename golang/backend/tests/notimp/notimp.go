// Package notimp encapsulates GX source files
// into a Go package.
//
// Automatically generated from google3/third_party/gxlang/gx/golang/packager/package.go.
//
// DO NOT EDIT
package notimp

import (
	"embed"

	"github.com/gx-org/gx/build/importers"
	"github.com/gx-org/gx/build/importers/embedpkg"

)

//go:embed notimp.gx 
var srcs embed.FS

var inputFiles = []string{
"notimp.gx",
}

func init() {
	embedpkg.RegisterPackage("github.com/gx-org/gx/golang/backend/tests/notimp", Build)
}

var _ embedpkg.BuildFunc = Build

// Build GX package.
func Build(bld importers.Builder) (importers.Package, error) {
	return bld.BuildFiles("github.com/gx-org/gx/golang/backend/tests", "notimp", srcs, inputFiles)
}
