// Package pkg1 encapsulates GX source files
// into a Go package.
//
// Automatically generated from google3/third_party/gxlang/gx/golang/packager/package.go.
//
// DO NOT EDIT
package pkg1

import (
	"embed"

	"github.com/gx-org/gx/build/importers"
	"github.com/gx-org/gx/build/importers/embedpkg"

)

//go:embed pkg1.gx 
var srcs embed.FS

var inputFiles = []string{
"pkg1.gx",
}

func init() {
	embedpkg.RegisterPackage("github.com/gx-org/gx/tests/testfiles/imports/pkg1", Build)
}

var _ embedpkg.BuildFunc = Build

// Build GX package.
func Build(bld importers.Builder) (importers.Package, error) {
	return bld.BuildFiles("github.com/gx-org/gx/tests/testfiles/imports", "pkg1", srcs, inputFiles)
}
