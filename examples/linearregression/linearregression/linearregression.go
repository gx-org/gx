// Package linearregression encapsulates GX source files
// into a Go package.
//
// Automatically generated from google3/third_party/gxlang/gx/golang/packager/package.go.
//
// DO NOT EDIT
package linearregression

import (
	"embed"

	"github.com/gx-org/gx/build/importers"
	"github.com/gx-org/gx/build/importers/embedpkg"

)

//go:embed learner.gx target.gx 
var srcs embed.FS

var inputFiles = []string{
"learner.gx","target.gx",
}

func init() {
	embedpkg.RegisterPackage("github.com/gx-org/gx/examples/linearregression/linearregression", Build)
}

var _ embedpkg.BuildFunc = Build

// Build GX package.
func Build(bld importers.Builder) (importers.Package, error) {
	return bld.BuildFiles("github.com/gx-org/gx/examples/linearregression", "linearregression", srcs, inputFiles)
}
