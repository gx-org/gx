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
	"maps"
	"slices"
	"sort"
	"strconv"

	"github.com/pkg/errors"
	"github.com/gx-org/gx/build/fmterr"
	"github.com/gx-org/gx/build/ir"
)

type (
	topLevelIdent struct {
		node    *identNode
		builder irBuilder
	}

	packageNamespace struct {
		idents map[string]*topLevelIdent
		pkg    *basePackage
	}

	irBuilder interface {
		buildIR(pkg *ir.Package)
	}
)

var _ namespaceFetcher = (*packageNamespace)(nil)

func (ns *packageNamespace) fetchIdent(name string) (*identNode, bool) {
	tpIdent, ok := ns.idents[name]
	if !ok {
		return nil, false
	}
	return tpIdent.node, true
}

func (ns *packageNamespace) fetch(name string) *identNode {
	return fetch(ns.fetchIdent, nil, name)
}

func (ns *packageNamespace) assign(node *identNode, b irBuilder) {
	ns.idents[node.ident.Name] = &topLevelIdent{
		node:    node,
		builder: b,
	}
}

type basePackage struct {
	override bool
	bld      *Builder
	repr     ir.Package
	ns       *packageNamespace
}

func newBasePackage(b *Builder, path string, override bool) *basePackage {
	pkg := &basePackage{
		bld:      b,
		override: override,
		ns: &packageNamespace{
			idents: make(map[string]*topLevelIdent),
		},
		repr: ir.Package{
			FSet: token.NewFileSet(),
			Path: path,
		},
	}
	pkg.ns.pkg = pkg
	return pkg
}

func (pkg *basePackage) base() *basePackage {
	return pkg
}

func (pkg *basePackage) setOrCheckName(name *ast.Ident) error {
	if pkg.repr.Name == nil {
		pkg.repr.Name = name
		return nil
	}
	if pkg.repr.Name.Name != name.Name {
		return errors.Errorf("package has already name %q", pkg.repr.Name)
	}
	return nil
}

func (pkg *basePackage) cleanIR() {
	pkg.repr.Files = nil
	pkg.repr.Funcs = nil
	pkg.repr.Vars = nil
	pkg.repr.Consts = nil
	pkg.repr.Types = nil
}

func (pkg *basePackage) builder() *Builder {
	return pkg.bld
}

func (pkg *basePackage) buildIdentIRs() {
	for _, name := range slices.Sorted(maps.Keys(pkg.ns.idents)) {
		node := pkg.ns.idents[name]
		node.builder.buildIR(&pkg.repr)
	}
}

func (pkg *basePackage) checkIfDefined(block *scopeFile, ident *ast.Ident) bool {
	if pkg.override {
		return true
	}
	if prev := pkg.ns.fetch(ident.Name); prev != nil {
		appendRedeclaredError(block.err(), ident, prev)
		return false
	}
	return true
}

// FilePackage builds GX package from GX source files
// or programmatically build IR.
type FilePackage struct {
	*basePackage

	files     []*file
	irImports *file
}

var _ Package = (*FilePackage)(nil)

func (b *Builder) newFilePackage(path, name string) *FilePackage {
	pkg := &FilePackage{
		basePackage: newBasePackage(b, path, false),
	}
	pkg.irImports = newFile(pkg.basePackage, "")
	if name != "" {
		pkg.setOrCheckName(&ast.Ident{Name: name})
	}
	return pkg
}

// BuildFiles complete the package definitions from a list of source files.
func (pkg *FilePackage) BuildFiles(fs fs.FS, filenames []string) (err error) {
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
func (pkg *FilePackage) buildFile(fs fs.FS, filename string, errs *fmterr.Errors) {
	file, err := fs.Open(filename)
	if err != nil {
		errs.Append(err)
		return
	}
	src, err := io.ReadAll(file)
	if err != nil {
		errs.Append(errors.Errorf("cannot read file %q: %v", file, err))
		return
	}

	fileDecl, err := parser.ParseFile(pkg.repr.FSet, filename, src, parser.ParseComments|parser.SkipObjectResolution)
	if err != nil {
		errs.Append(errors.Errorf("cannot parse file %s:\n\t%v", filename, err))
		return
	}
	pkg.process(filename, fileDecl, errs)
}

// ImportIR imports package definitions from a GX intermediate representation.
func (pkg *FilePackage) ImportIR(repr *ir.Package) error {
	pkg.cleanIR()
	errs := &fmterr.Errors{}
	scope := newScopePackage(pkg.basePackage, errs).scopeFile(pkg.irImports)
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

func (pkg *FilePackage) process(name string, src *ast.File, errs *fmterr.Errors) {
	pkg.cleanIR()
	scope := newScopePackage(pkg.basePackage, errs)
	file := processFile(scope, name, src)
	pkg.files = append(pkg.files, file)
}

func (pkg *FilePackage) resolve(errs *fmterr.Errors) {
	scope := newScopePackage(pkg.basePackage, errs)
	pkg.irImports.resolveAll(scope)
	for _, file := range pkg.files {
		file.resolveAll(scope)
	}
}

func (pkg *FilePackage) buildIR() {
	pkg.repr.Files = make(map[string]*ir.File)
	pkg.irImports.buildIR(&pkg.repr)
	for _, file := range pkg.files {
		file.buildIR(&pkg.repr)
	}
	pkg.buildIdentIRs()
}

// IR returns the package GX intermediate representation.
func (pkg *FilePackage) IR() *ir.Package {
	if pkg.repr.Files == nil {
		pkg.buildIR()
	}
	return &pkg.repr
}

// IncrementalPackage builds GX package from an AST.
// It uses a single file in which everything is defined.
// Any name in the package can be reassigned without triggering an error.
// The main use case is for GX in a notebook.
type IncrementalPackage struct {
	*basePackage

	next  int
	files []*file
}

var _ Package = (*IncrementalPackage)(nil)

// NewIncrementalPackage creates a new incremental package.
func (b *Builder) NewIncrementalPackage(name string) *IncrementalPackage {
	pkg := &IncrementalPackage{
		basePackage: newBasePackage(b, "", true),
	}
	pkg.basePackage.repr.Name = &ast.Ident{
		Name: name,
	}
	return pkg
}

// Build a AST source file. Definitions are added to the package or replace existing definitions.
func (pkg *IncrementalPackage) Build(src string) error {
	name := strconv.Itoa(pkg.next)
	astFile, err := parser.ParseFile(pkg.repr.FSet, name, src, parser.ParseComments)
	if err != nil {
		return err
	}
	errs := &fmterr.Errors{}
	scope := newScopePackage(pkg.basePackage, errs)
	file := processFile(scope, name, astFile)
	pkg.files = append(pkg.files, file)
	file.resolveAll(scope)
	return errs.ToError()
}

// IR returns the package GX intermediate representation.
func (pkg *IncrementalPackage) IR() *ir.Package {
	pkg.cleanIR()
	pkg.repr.Files = make(map[string]*ir.File)
	for _, file := range pkg.files {
		file.buildIR(&pkg.repr)
	}
	pkg.buildIdentIRs()
	return &pkg.repr
}
