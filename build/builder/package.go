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
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"io/fs"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/pkg/errors"
	"github.com/gx-org/gx/base/ordered"
	"github.com/gx-org/gx/base/uname"
	"github.com/gx-org/gx/build/fmterr"
	"github.com/gx-org/gx/build/ir"
)

type lastBuild struct {
	builderState *pkgState
	methods      []*processNodeT[function]
	pkg          *ir.Package
}

func (s *pkgResolveScope) lastBuild() *lastBuild {
	last := &lastBuild{
		pkg:          s.state.ibld.Pkg(),
		builderState: s.state,
	}
	for methods := range s.methods.Values() {
		for method := range methods.Values() {
			last.methods = append(last.methods, method.pNode)
		}
	}
	last.pkg.Decls = s.state.ibld.Decls()
	return last
}

func (lb *lastBuild) decls() *ordered.Map[string, processNode] {
	if lb.builderState == nil {
		return ordered.NewMap[string, processNode]()
	}
	return lb.builderState.dcls.declarations
}

func (lb *lastBuild) String() string {
	s := &strings.Builder{}
	fmt.Fprintln(s, "Declarations:")
	for k, v := range lb.decls().Iter() {
		fmt.Fprintf(s, "  %s: %T\n", k, v)
	}
	fmt.Fprintln(s, "Package:")
	for _, cDecl := range lb.pkg.Decls.Consts {
		for _, cExpr := range cDecl.Exprs {
			fmt.Fprintf(s, "  const %s\n", cExpr.VName)
		}
	}
	return s.String()
}

type basePackage struct {
	bld *Builder

	name      *ast.Ident
	path      string
	fset      *token.FileSet
	files     *ordered.Map[string, *file]
	last      *lastBuild
	unames    *uname.Unique
	macroRoot *uname.Root
}

func newBasePackage(b *Builder, path string) *basePackage {
	pkg := &basePackage{
		bld:    b,
		path:   path,
		fset:   token.NewFileSet(),
		files:  ordered.NewMap[string, *file](),
		unames: uname.New(),
	}
	pkg.macroRoot = pkg.unames.Root("_macro")
	pkg.last = &lastBuild{
		pkg: pkg.newPackageIR(),
	}
	return pkg
}

func (pkg *basePackage) base() *basePackage {
	return pkg
}

func (pkg *basePackage) setOrCheckName(name *ast.Ident) error {
	if pkg.name == nil {
		pkg.name = name
		return nil
	}
	if pkg.name.Name != name.Name {
		return errors.Errorf("package has already name %q", pkg.name)
	}
	return nil
}

func (pkg *basePackage) builder() *Builder {
	return pkg.bld
}

func (pkg *basePackage) resolveBuild(pscope *pkgProcScope) (*pkgResolveScope, bool) {
	pkgScope, ok := newPackageResolveScope(pscope)
	if !ok {
		return pkgScope, false
	}
	if ok := pscope.dcls.resolveAll(pkgScope); !ok {
		return pkgScope, false
	}
	pkg.last = pkgScope.lastBuild()
	return pkgScope, true
}

func (pkg *basePackage) newPackageIR() *ir.Package {
	return &ir.Package{
		FSet:  pkg.fset,
		Name:  pkg.name,
		Path:  pkg.path,
		Files: make(map[string]*ir.File),
		Decls: &ir.Declarations{},
	}
}

// FilePackage builds GX package from GX source files
// or programmatically build IR.
type FilePackage struct {
	*basePackage

	irImports *file
}

var _ Package = (*FilePackage)(nil)

func (b *Builder) newFilePackage(path, name string) *FilePackage {
	pkg := &FilePackage{
		basePackage: newBasePackage(b, path),
	}
	pkg.irImports = newFile(pkg.basePackage, "", &ast.File{})
	if name != "" {
		pkg.setOrCheckName(&ast.Ident{Name: name})
	}
	return pkg
}

// BuildFiles complete the package definitions from a list of source files.
func (pkg *FilePackage) BuildFiles(fs fs.FS, filenames []string) (err error) {
	if fs == nil {
		return errors.Errorf("no file system to load files from")
	}
	if len(filenames) == 0 {
		return errors.Errorf("no input files to compile")
	}
	sort.Strings(filenames)
	errs := fmterr.Errors{}
	pscope := newPackageProcScope(false, pkg.basePackage, &errs)
	ok := true
	for _, filename := range filenames {
		fileOk := pkg.buildFile(pscope, fs, filename)
		ok = ok && fileOk
	}
	if !ok {
		return &errs
	}
	pkgScope, ok := pkg.resolveBuild(pscope)
	if !ok {
		pkg.last.pkg = pkgScope.irBuilder().Pkg()
		return &errs
	}
	return nil
}

// buildFile processes a file. Returned false if the file could not be parsed.
// Process errors are accumulated in the package and functions.
func (pkg *FilePackage) buildFile(pscope *pkgProcScope, fs fs.FS, filename string) bool {
	file, err := fs.Open(filename)
	if err != nil {
		return pscope.Err().Append(err)
	}
	src, err := io.ReadAll(file)
	if err != nil {
		return pscope.Err().Append(errors.Errorf("cannot read file %q: %v", file, err))
	}

	fileDecl, err := parser.ParseFile(pkg.fset, filename, src, parser.ParseComments|parser.SkipObjectResolution)
	if err != nil {
		return pscope.Err().Append(errors.Errorf("cannot parse file %s:\n\t%v", filename, err))
	}
	_, ok := processFile(pscope, filename, fileDecl)
	return ok
}

// ImportIR imports package definitions from a GX intermediate representation.
func (pkg *FilePackage) ImportIR(decls *ir.Declarations) error {
	errs := &fmterr.Errors{}
	pscope := newPackageProcScope(false, pkg.basePackage, errs)
	if !importNamedTypes(pscope, pkg.irImports, decls.Types) {
		return errs
	}
	if !importFuncs(pscope, pkg.irImports, decls.Funcs) {
		return errs
	}
	if !importConstDecls(pscope, pkg.irImports, decls.Consts) {
		return errs
	}
	if _, ok := pkg.resolveBuild(pscope); !ok {
		return errs
	}
	return nil
}

// IR returns the package GX intermediate representation.
func (pkg *FilePackage) IR() *ir.Package {
	return pkg.last.pkg
}

// IncrementalPackage builds GX package from an AST.
// It uses a single file in which everything is defined.
// Any name in the package can be reassigned without triggering an error.
// The main use case is for GX in a notebook.
type IncrementalPackage struct {
	mut sync.Mutex
	*basePackage

	next int
}

var _ Package = (*IncrementalPackage)(nil)

// NewIncrementalPackage creates a new incremental package.
func (b *Builder) NewIncrementalPackage(fullname string) *IncrementalPackage {
	paths := strings.Split(fullname, "/")
	name := paths[len(paths)-1]
	path := strings.Join(paths[:len(paths)-1], "/")
	pkg := &IncrementalPackage{
		basePackage: newBasePackage(b, path),
	}
	if name == "" {
		return pkg
	}
	pkg.basePackage.name = &ast.Ident{
		Name: name,
	}
	return pkg
}

// Build a AST source file. Definitions are added to the package or replace existing definitions.
func (pkg *IncrementalPackage) Build(src string) error {
	pkg.mut.Lock()
	defer pkg.mut.Unlock()

	name := strconv.Itoa(pkg.next)
	astFile, err := parser.ParseFile(pkg.fset, name, src, parser.ParseComments)
	if err != nil {
		return err
	}
	errs := &fmterr.Errors{}
	pscope := newPackageProcScope(true, pkg.basePackage, errs)
	if _, ok := processFile(pscope, name, astFile); !ok {
		return errs
	}
	if _, ok := pkg.resolveBuild(pscope); !ok {
		return errs
	}
	return nil
}

// BuildExpr builds an expression.
func (pkg *IncrementalPackage) BuildExpr(src string, imports ...*ast.ImportSpec) (ir.Expr, error) {
	const fileName = "expression"
	fset := &token.FileSet{}
	fset.AddFile(fileName, -1, len(src))
	astExpr, err := parser.ParseExprFrom(fset, fileName, src, parser.SkipObjectResolution)
	if err != nil {
		return nil, err
	}
	errs := &fmterr.Errors{}
	pkgScope := newPackageProcScope(false, pkg.basePackage, errs)
	file := newFile(pkg.basePackage, fileName, &ast.File{
		Imports: imports,
	})
	pscope := pkgScope.newFilePScope(file)
	if ok := file.processDecls(pscope, nil); !ok {
		return nil, errs
	}
	bFType, ok := processFuncType(pscope, &ast.FuncType{}, nil, false)
	if !ok {
		return nil, errs
	}
	expr, _ := processExpr(pscope, astExpr)
	if !errs.Empty() {
		return nil, errs
	}
	pkgRScope, ok := pkg.resolveBuild(pkgScope)
	if !ok {
		return nil, errs
	}
	rScope, ok := pkgRScope.newFileRScope(file)
	if !ok {
		return nil, errs
	}
	_, funRScope, ok := bFType.buildFuncType(rScope)
	if !ok {
		return nil, errs
	}
	blockScope, ok := newBlockScope(funRScope, &blockStmt{
		src: &ast.BlockStmt{},
	})
	if !ok {
		return nil, errs
	}
	irExpr, ok := expr.buildExpr(blockScope)
	if !ok {
		return nil, errs
	}
	return irExpr, nil
}

// Fetcher builds a fetcher to evaluate expressions.
func (pkg *IncrementalPackage) Fetcher() (ir.Fetcher, error) {
	errs := &fmterr.Errors{}
	pkgScope := newPackageProcScope(false, pkg.basePackage, errs)
	pkgRScope, ok := pkg.resolveBuild(pkgScope)
	if !ok {
		return nil, errs.ToError()
	}
	file := newFile(pkg.basePackage, "", &ast.File{})
	fScope, ok := pkgRScope.newFileRScope(file)
	if !ok {
		return nil, errs.ToError()
	}
	cp, ok := fScope.compEval()
	if !ok {
		return nil, errs.ToError()
	}
	return cp, nil
}

// IR returns the package GX intermediate representation.
func (pkg *IncrementalPackage) IR() *ir.Package {
	pkg.mut.Lock()
	defer pkg.mut.Unlock()

	return pkg.last.pkg
}
