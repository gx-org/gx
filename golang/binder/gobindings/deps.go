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

package gobindings

import (
	"fmt"
	"maps"
	"slices"

	"github.com/gx-org/gx/build/ir"
)

type dependency struct {
	Index       int
	PackagePath string
	ImportName  string
}

func (b *binder) buildDependencies() map[string]dependency {
	deps := make(map[string]dependency)
	for file := range maps.Values(b.Package.Files) {
		for _, imp := range file.Imports {
			path := b.packageToPath(imp.Package)
			_, ok := deps[path]
			if ok {
				continue
			}
			index := len(deps)
			importName := fmt.Sprintf("gxdep%d", index)
			deps[path] = dependency{
				PackagePath: path,
				ImportName:  importName,
				Index:       index,
			}
		}
	}
	return deps
}

func (b *binder) Dependencies() []dependency {
	keys := slices.Sorted(maps.Keys(b.dependencies))
	deps := make([]dependency, len(keys))
	for i, depName := range keys {
		deps[i] = b.dependencies[depName]
	}
	return deps
}

func (b *binder) NumDeps() int {
	return len(b.dependencies)
}

func (b *binder) registerImport(path string) {
}

func (b *binder) packageToPath(pkg *ir.Package) string {
	packagePath := pkg.FullName()
	if b.stdlib.Support(packagePath) {
		return b.builder.StdlibDependencyImport(packagePath)
	}
	return b.builder.DependencyImport(pkg)
}

func (b *binder) namePackage(pkg *ir.Package) string {
	path := b.packageToPath(pkg)
	return b.dependencies[path].ImportName
}
