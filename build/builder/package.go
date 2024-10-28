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

package builder

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"io/fs"
	"sort"

	"github.com/pkg/errors"
	"github.com/gx-org/gx/build/fmterr"
	"github.com/gx-org/gx/build/ir"
)

type bPackage struct {
	bld  *Builder
	repr ir.Package
	ns   *namespace

	files     []*file
	irImports *file
}

func (b *Builder) newPackage(path, name string) *bPackage {
	pkg := &bPackage{
		bld: b,
		repr: ir.Package{
			FSet: token.NewFileSet(),
			Path: path,
		},
	}
	pkg.ns = newNameSpace(nil)
	pkg.irImports = newFile(pkg, "")
	if name != "" {
		pkg.setOrCheckName(&ast.Ident{Name: name})
	}
	return pkg
}

// BuildFiles complete the package definitions from a list of source files.
func (pkg *bPackage) BuildFiles(fs fs.FS, filenames []string) (err error) {
	pkg.cleanIR()
	if fs == nil {
		return errors.Errorf("no file system to load files from")
	}
	if len(filenames) == 0 {
		return errors.Errorf("no input files to compile")
	}
	sort.Strings(filenames)
	errs := fmterr.Errors{}
	for _, filename := range filenames {
		pkg.buildFile(fs, filename, &errs)
	}
	if !errs.Empty() {
		return errs.ToError()
	}
	pkg.resolve(&errs)
	return errs.ToError()
}

// buildFile processes a file. Returned false if the file could not be parsed.
// Process errors are accumulated in the package and functions.
func (pkg *bPackage) buildFile(fs fs.FS, filename string, errs *fmterr.Errors) {
	file, err := fs.Open(filename)
	if err != nil {
		errs.Append(errors.Errorf("cannot open file %q: %v", file, err))
		return
	}
	src, err := io.ReadAll(file)
	if err != nil {
		errs.Append(errors.Errorf("cannot read file %q: %v", file, err))
		return
	}

	fileDecl, err := parser.ParseFile(pkg.repr.FSet, filename, src, parser.ParseComments)
	if err != nil {
		errs.Append(errors.Errorf("cannot parse file %s:\n\t%v", filename, err))
		return
	}
	pkg.process(filename, fileDecl, errs)
}

// ImportIR imports package definitions from a GX intermediate representation.
func (pkg *bPackage) ImportIR(repr *ir.Package) error {
	pkg.cleanIR()
	errs := &fmterr.Errors{}
	scope := newScopePackage(pkg, errs).scopeFile(pkg.irImports)
	if importNamedTypes(scope, repr.Types); !errs.Empty() {
		return errs.ToError()
	}
	if importFuncs(scope, repr.Funcs); !errs.Empty() {
		return errs.ToError()
	}
	if importConstDecls(scope, repr.Consts); !errs.Empty() {
		return errs.ToError()
	}
	return nil
}

func (pkg *bPackage) setOrCheckName(name *ast.Ident) error {
	if pkg.repr.Name == nil {
		pkg.repr.Name = name
		return nil
	}
	if pkg.repr.Name.Name != name.Name {
		return errors.Errorf("package has already name %q", pkg.repr.Name)
	}
	return nil
}

func (pkg *bPackage) process(name string, src *ast.File, errs *fmterr.Errors) {
	pkg.cleanIR()
	scope := newScopePackage(pkg, errs)
	file := processFile(scope, name, src)
	pkg.files = append(pkg.files, file)
}

func (pkg *bPackage) resolve(errs *fmterr.Errors) {
	scope := newScopePackage(pkg, errs)
	pkg.irImports.resolveAll(scope)
	for _, file := range pkg.files {
		file.resolveAll(scope)
	}
}

func (pkg *bPackage) cleanIR() {
	pkg.repr.Files = nil
	pkg.repr.Funcs = nil
	pkg.repr.Vars = nil
	pkg.repr.Consts = nil
	pkg.repr.Types = nil
}

func (pkg *bPackage) buildIR() {
	pkg.repr.Files = make(map[string]*ir.File)
	pkg.irImports.buildIRs()
	for _, file := range pkg.files {
		file.buildIRs()
	}
}

// IR returns the package GX intermediate representation.
func (pkg *bPackage) IR() *ir.Package {
	if pkg.repr.Files == nil {
		pkg.buildIR()
	}
	return &pkg.repr
}

func (pkg *bPackage) importOk() bool {
	ok := true
	for _, file := range pkg.files {
		ok = file.importOk && ok
	}
	return ok
}

func (pkg *bPackage) builder() *Builder {
	return pkg.bld
}
