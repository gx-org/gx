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
	"sort"
	"strings"

	"golang.org/x/exp/maps"
	"github.com/gx-org/gx/build/ir"
)

type dependency struct {
	PackagePath string
	ImportName  string
}

func (b *binder) Dependencies() []dependency {
	keys := maps.Keys(b.dependencies)
	sort.Strings(keys)
	deps := make([]dependency, len(keys))
	for i, depName := range keys {
		deps[i] = b.dependencies[depName]
	}
	return deps
}

func (b *binder) GXImportDeps() string {
	var imports []string
	for _, dep := range b.Dependencies() {
		imports = append(imports, fmt.Sprintf("\t%s \"%s\"", dep.ImportName, dep.PackagePath))
	}
	sort.Slice(imports, func(i, j int) bool {
		return imports[i] < imports[j]
	})
	return strings.Join(imports, "\n")
}

func (b *binder) importPathToName(path string) string {
	dep, ok := b.dependencies[path]
	if ok {
		return dep.ImportName
	}
	importName := fmt.Sprintf("gxdep%d", len(b.dependencies))
	b.dependencies[path] = dependency{
		PackagePath: path,
		ImportName:  importName,
	}
	return importName
}

func (b *binder) namePackage(pkg *ir.Package) string {
	packagePath := pkg.FullName()
	if b.stdlib.Support(packagePath) {
		packagePath = b.builder.StdlibDependencyImport(packagePath)
	} else {
		packagePath = b.builder.DependencyImport(packagePath)
	}
	return b.importPathToName(packagePath)
}
