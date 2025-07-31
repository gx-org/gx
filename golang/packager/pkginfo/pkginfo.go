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

// Package pkginfo extracts information from a gx package folder.
// This includes package path, dependencies, etc.
package pkginfo

import (
	"flag"
	"fmt"
	"go/parser"
	"go/token"
	"io/ioutil"
	"maps"
	"path/filepath"
	"slices"
	"strings"

	"github.com/gx-org/gx/build/module"
	"github.com/gx-org/gx/golang/packager/goembed"
	"github.com/gx-org/gx/stdlib"
)

var gxPackage = flag.String("gx_package", "", "GX package to embed")

// PkgInfo constructs information from a GX package path
// present in the Go module.
type PkgInfo struct {
	mod       *module.Module
	gxPackage string
	srcs      []string
	stdlib    *stdlib.Stdlib

	targetFile string
	deps       []string
}

// BuildPackageInfo builds information about a package.
func BuildPackageInfo() (goembed.PackageInfo, error) {
	mod, err := module.Current()
	if err != nil {
		return nil, err
	}
	return Load(mod, *gxPackage)
}

// Load package information from a GX path in the module.
func Load(mod *module.Module, gxPackage string) (*PkgInfo, error) {
	if gxPackage == "" {
		return nil, fmt.Errorf("no import path given")
	}
	pkgInfo := &PkgInfo{
		gxPackage: gxPackage,
		stdlib:    stdlib.Importer(nil),
		mod:       mod,
	}
	var err error
	osPath, err := pkgInfo.mod.ImportToOSPath(pkgInfo.gxPackage)
	if err != nil {
		return nil, fmt.Errorf("cannot get OS path for package %s: %v", pkgInfo.gxPackage, err)
	}
	pkgInfo.srcs, err = pkgInfo.buildSourceFiles(osPath)
	if err != nil {
		return nil, err
	}
	pkgInfo.deps, err = pkgInfo.buildDependencies()
	if err != nil {
		return nil, err
	}
	pkgInfo.targetFile = osPath + "/" + pkgInfo.GoPackageName() + ".go"

	return pkgInfo, nil
}

// TargetFile returns the folder where to generate the Go source file packaging the GX source files.
func (inf PkgInfo) TargetFile() string {
	return inf.targetFile
}

// GoPackageName returns the Go package name.
func (inf PkgInfo) GoPackageName() string {
	return inf.GXPackageName()
}

// GXPackageName is the name of the GX package.
func (inf PkgInfo) GXPackageName() string {
	paths := strings.Split(inf.gxPackage, "/")
	return paths[len(paths)-1]
}

// GXPackagePath is the name of the GX package.
func (inf PkgInfo) GXPackagePath() string {
	paths := strings.Split(inf.gxPackage, "/")
	if len(paths) == 1 {
		return ""
	}
	return strings.Join(paths[:len(paths)-1], "/")
}

// GXPackage is the full path of the GX package.
func (inf PkgInfo) GXPackage() string {
	return inf.gxPackage
}

func (inf PkgInfo) osPackagePath() string {
	return inf.mod.OSPath(inf.gxPackage)
}

// SourceFiles returns the list of all the GX source files.
func (inf PkgInfo) buildSourceFiles(osPath string) ([]string, error) {
	if len(inf.srcs) > 0 {
		return inf.srcs, nil
	}
	files, err := ioutil.ReadDir(osPath)
	if err != nil {
		return nil, fmt.Errorf("cannot list GX package source files: %v", err)
	}
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		if !strings.HasSuffix(file.Name(), ".gx") {
			continue
		}
		inf.srcs = append(inf.srcs, filepath.Join(osPath, file.Name()))
	}
	return inf.srcs, nil
}

// Embed returns the list of file to embed.
func (inf PkgInfo) Embed() []string {
	bases := make([]string, len(inf.srcs))
	for i, filename := range inf.srcs {
		bases[i] = filepath.Base(filename)
	}
	return bases
}

func extractImports(srcPath string) ([]string, error) {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, srcPath, nil, parser.ImportsOnly)
	if err != nil {
		return nil, err
	}
	var deps []string
	for _, imp := range node.Imports {
		dep := imp.Path.Value
		dep = strings.TrimPrefix(dep, `"`)
		dep = strings.TrimSuffix(dep, `"`)
		deps = append(deps, dep)
	}
	return deps, nil
}

// Dependencies returns the list of dependencies of the GX package.
func (inf PkgInfo) Dependencies() []string {
	return inf.deps
}

// SourceFiles returns the list of GX source files for the package.
func (inf PkgInfo) SourceFiles() []string {
	return inf.srcs
}

func (inf PkgInfo) buildDependencies() ([]string, error) {
	depSet := map[string]bool{}
	for _, src := range inf.srcs {
		srcDeps, err := extractImports(src)
		if err != nil {
			return nil, fmt.Errorf("cannot extract import from %s: %v", src, err)
		}
		for _, dep := range srcDeps {
			if inf.stdlib.Support(dep) {
				// Skip Dependencies to the standard libraries.
				continue
			}
			depSet[dep] = true
		}
	}
	deps := slices.Sorted(maps.Keys(depSet))
	return deps, nil
}
