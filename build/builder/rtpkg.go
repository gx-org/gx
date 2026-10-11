// Copyright 2026 Google LLC
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

package builder

import (
	"path"
	"strings"

	"github.com/gx-org/gx/build/importers"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/interp/elements"
)

const runtimePrefix = "runtime" + ir.PackageTypeSeparator

type runtimePackage struct {
	*basePackage
}

// loadRuntimePackage loads a runtime package, that is a package that only be known at runtime,
// as opposed to compile time.
func (b *Builder) loadRuntimePackage(pkgpath string) (bPackage, error) {
	pkgpath = strings.TrimPrefix(pkgpath, runtimePrefix)
	var loaderPkg importers.Package
	var err error
	dir, name := path.Split(pkgpath)
	pkg := &runtimePackage{
		basePackage: newBasePackage(b, dir, name),
	}
	if b.loader.Support(pkgpath) {
		loaderPkg, err = b.loader.Load(b, pkgpath)
	}
	if loaderPkg != nil {
		pkg.last.pkg = loaderPkg.IR()
		return pkg, err
	}
	runtimeFunc := func() (ir.PackageElement, error) {
		return elements.RuntimePackage(pkg.last.pkg)
	}
	pkg.last.pkg.Runtime = runtimeFunc
	pkg.last.pkg.Decls = nil
	return pkg, err
}

func (pkg *runtimePackage) IR() *ir.Package {
	return pkg.last.pkg
}
