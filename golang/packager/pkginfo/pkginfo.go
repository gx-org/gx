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
	"go/parser"
	"go/token"
	"io/ioutil"
	"sort"
	"strings"

	"golang.org/x/exp/maps"
	"github.com/gx-org/gx/build/module"
)

// PkgInfo constructs information from a GX package path
// present in the Go module.
type PkgInfo struct {
	mod        *module.Module
	importPath string
	pkgPath    string
	pkgName    string
	srcs       []string
}

// Load package information from a GX path in the module.
func Load(importPath string) (*PkgInfo, error) {
	inf := &PkgInfo{importPath: importPath}
	var err error
	inf.mod, err = module.Current()
	if err != nil {
		return nil, err
	}
	inf.pkgPath, inf.pkgName, err = inf.mod.Split(inf.importPath)
	if err != nil {
		return nil, err
	}
	return inf, nil
}

// TargetFileName returns the file name to generate for the Go module.
func (inf PkgInfo) TargetFileName() string {
	return inf.GoPackageName() + ".go"
}

// TargetFolder returns the folder where to generate the Go source file packaging the GX source files.
func (inf PkgInfo) TargetFolder() string {
	return inf.mod.OSPath(inf.pkgPath)
}

// GoPackageName returns the Go package name.
func (inf PkgInfo) GoPackageName() string {
	return inf.pkgName
}

func (inf PkgInfo) osPackagePath() string {
	return inf.mod.OSPath(inf.pkgPath)
}

// SourceFiles returns the list of all the GX source files.
func (inf PkgInfo) SourceFiles() ([]string, error) {
	if len(inf.srcs) > 0 {
		return inf.srcs, nil
	}
	files, err := ioutil.ReadDir(inf.osPackagePath())
	if err != nil {
		return nil, err
	}
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		if !strings.HasSuffix(file.Name(), ".gx") {
			continue
		}
		inf.srcs = append(inf.srcs, file.Name())
	}
	return inf.srcs, nil
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
func (inf PkgInfo) Dependencies() ([]string, error) {
	srcs, err := inf.SourceFiles()
	if err != nil {
		return nil, err
	}
	depSet := map[string]bool{}
	osPkgPath := inf.osPackagePath() + "/"
	for _, src := range srcs {
		srcDeps, err := extractImports(osPkgPath + src)
		if err != nil {
			return nil, err
		}
		for _, dep := range srcDeps {
			if !strings.HasPrefix(dep, "github.com") {
				continue
			}
			depSet[dep] = true
		}
	}
	deps := maps.Keys(depSet)
	sort.Strings(deps)
	return deps, nil
}
