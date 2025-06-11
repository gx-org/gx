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

package builtin

import (
	"embed"
	"fmt"
	"go/ast"
	"reflect"
	"runtime"
	"strings"

	"github.com/pkg/errors"
	"github.com/gx-org/gx/build/builder"
	"github.com/gx-org/gx/build/fmterr"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/interp/compeval/cpevelements"
	"github.com/gx-org/gx/interp"
	"github.com/gx-org/gx/stdlib/impl"
)

type baseBuilder struct {
	name  string
	build func(*builder.Builder, *impl.Stdlib, *builder.FilePackage) error
}

var _ Builder = (*baseBuilder)(nil)

func (fb baseBuilder) Name() string {
	return fb.name
}

func (fb baseBuilder) Build(bld *builder.Builder, impl *impl.Stdlib, pkg *builder.FilePackage) error {
	return fb.build(bld, impl, pkg)
}

type sourceParser struct {
	fs    *embed.FS
	names []string
}

func topLevelNames(fs *embed.FS) ([]string, error) {
	entries, err := fs.ReadDir(".")
	if err != nil {
		return nil, fmt.Errorf("cannot read GX files directory: %w", err)
	}
	names := make([]string, len(entries))
	for i, entry := range entries {
		names[i] = entry.Name()
	}
	return names, nil
}

func (b *sourceParser) Name() string {
	return fmt.Sprintf("%T%v", b, b.names)
}

func (b *sourceParser) Build(bld *builder.Builder, _ *impl.Stdlib, pkg *builder.FilePackage) (err error) {
	if len(b.names) == 0 {
		if b.names, err = topLevelNames(b.fs); err != nil {
			return err
		}
	}
	return pkg.BuildFiles(b.fs, b.names)
}

// ParseSource builds a package by parsing source files.
// If names is empty, all files at the top-level will parsed.
func ParseSource(fs *embed.FS, names ...string) Builder {
	return &sourceParser{fs: fs, names: names}
}

// FuncBuilder builds a function for a package.
type FuncBuilder interface {
	BuildFuncIR(*impl.Stdlib, *ir.Package) (*ir.FuncBuiltin, error)
}

func funcName(f any) string {
	val := reflect.ValueOf(f)
	if val.Kind() != reflect.Pointer {
		return val.Type().String()
	}
	return runtime.FuncForPC(val.Pointer()).Name()
}

// BuildFunc builds a function in a package.
func BuildFunc(f FuncBuilder) Builder {
	buildFunc := func(_ *builder.Builder, impl *impl.Stdlib, pkg *builder.FilePackage) error {
		irPkg := pkg.IR()
		fn, err := f.BuildFuncIR(impl, irPkg)
		if err != nil {
			return err
		}
		return pkg.ImportIR(&ir.Declarations{
			Funcs: []ir.PkgFunc{fn},
		})
	}
	return baseBuilder{
		name:  funcName(f),
		build: buildFunc,
	}
}

type stubFunc struct {
	ftype *ir.FuncType
	impl  any
}

var _ ir.FuncImpl = (*stubFunc)(nil)

// BuildFuncType builds the type of a function given how it is called.
func (s *stubFunc) BuildFuncType(fetcher ir.Fetcher, call *ir.CallExpr) (*ir.FuncType, error) {
	return s.ftype, nil
}

// Implementation of the function, provided by the backend.
func (s *stubFunc) Implementation() any {
	return s.impl
}

func findFunc(pkg *ir.Package, name string) (*ir.FuncBuiltin, error) {
	fns := pkg.Decls.Funcs
	if splt := strings.Split(name, "."); len(splt) == 2 {
		typeName := splt[0]
		name = splt[1]
		typ := pkg.Decls.TypeByName(typeName)
		if typ == nil {
			return nil, errors.Errorf("cannot find type %s in package %s", typeName, pkg.FullName())
		}
		fns = typ.Methods
	}
	for _, fn := range fns {
		if fn.Name() != name {
			continue
		}
		builtin, ok := fn.(*ir.FuncBuiltin)
		if !ok {
			return nil, errors.Errorf("%s:%T is not a builtin function", name, fn)
		}
		return builtin, nil
	}
	return nil, nil
}

// ImplementStubFunc replaces a function declaration with a stdlib-provided implementation, while
// keeping the function's declared type.
func ImplementStubFunc(name string, slotFn func(impl *impl.Stdlib) interp.FuncBuiltin) Builder {
	return baseBuilder{
		name: name,
		build: func(bld *builder.Builder, impl *impl.Stdlib, pkg *builder.FilePackage) error {
			stub, err := findFunc(pkg.IR(), name)
			if err != nil {
				return err
			}
			if stub == nil {
				return errors.Errorf("failed to replace function stub %q: builtin function declaration not found", name)
			}
			stub.Impl = &stubFunc{ftype: stub.FuncType(), impl: slotFn(impl)}
			return nil
		},
	}
}

// ImplementBuiltin provides the implementation of a builtin function.
func ImplementBuiltin(name string, fn interp.FuncBuiltin) Builder {
	return ImplementStubFunc(name, func(*impl.Stdlib) interp.FuncBuiltin {
		return fn
	})
}

type graphFunc struct {
	ftype *ir.FuncType
	impl  interp.FuncBuiltin
}

var _ ir.FuncImpl = (*stubFunc)(nil)

// BuildFuncType builds the type of a function given how it is called.
func (s *graphFunc) BuildFuncType(fetcher ir.Fetcher, call *ir.CallExpr) (*ir.FuncType, error) {
	return s.ftype, nil
}

// Implementation of the function, provided by the backend.
func (s *graphFunc) Implementation() any {
	return s.impl
}

// ImplementGraphFunc sets the implementation in a function declaration.
// The function declared type does not change.
func ImplementGraphFunc(name string, slotFn interp.FuncBuiltin) Builder {
	return baseBuilder{
		name: name,
		build: func(bld *builder.Builder, impl *impl.Stdlib, pkg *builder.FilePackage) error {
			for _, fn := range pkg.IR().Decls.Funcs {
				if fn.Name() == name {
					stub := fn.(*ir.FuncBuiltin)
					stub.Impl = &stubFunc{ftype: fn.FuncType(), impl: slotFn}
					return nil
				}
			}
			return fmt.Errorf("cannot set function implementation: cannot find function %q in package %s", name, pkg.IR().Name)
		},
	}
}

// MethodBuilder builds a method for a package given its named type.
type MethodBuilder interface {
	BuildMethodIR(*impl.Stdlib, builder.Package, *ir.NamedType) (*ir.FuncBuiltin, error)
}

// BuildMethod builds a method for a named type in a package.
func BuildMethod(name string, f MethodBuilder) Builder {
	buildMethod := func(bld *builder.Builder, impl *impl.Stdlib, pkg *builder.FilePackage) error {
		irPkg := pkg.IR()
		var namedType *ir.NamedType
		for _, named := range irPkg.Decls.Types {
			if named.Name() == name {
				namedType = named
				break
			}
		}
		if namedType == nil {
			return errors.Errorf("type %s undefined", name)
		}
		fn, err := f.BuildMethodIR(impl, pkg, namedType)
		if err != nil {
			return err
		}
		fn.FFile = &ir.File{
			Package: pkg.IR(),
		}
		return pkg.ImportIR(&ir.Declarations{
			Types: []*ir.NamedType{
				&ir.NamedType{
					Src:     namedType.Src,
					Methods: []ir.PkgFunc{fn},
				},
			},
		})
	}
	return baseBuilder{
		name:  "MethodBuilder:" + name,
		build: buildMethod,
	}
}

// TypeBuilder builds a type for a package.
type TypeBuilder interface {
	BuildNamedType(*impl.Stdlib, *ir.Package) (*ir.NamedType, error)
}

// BuildType builds a function in a package.
func BuildType(f TypeBuilder) Builder {
	buildType := func(bld *builder.Builder, impl *impl.Stdlib, pkg *builder.FilePackage) error {
		irPkg := pkg.IR()
		tp, err := f.BuildNamedType(impl, irPkg)
		if err != nil {
			return err
		}
		return pkg.ImportIR(&ir.Declarations{
			Types: []*ir.NamedType{tp},
		})
	}
	return baseBuilder{
		name:  fmt.Sprintf("TypeBuilder:%T", f),
		build: buildType,
	}
}

// ConstBuilder builds a type for a package.
type ConstBuilder func(*ir.Package) (string, ir.AssignableExpr, ir.Type, error)

// BuildConst builds a function in a package.
func BuildConst(f ConstBuilder) Builder {
	buildConst := func(bld *builder.Builder, _ *impl.Stdlib, pkg *builder.FilePackage) error {
		irPkg := pkg.IR()
		name, expr, typ, err := f(irPkg)
		if err != nil {
			return err
		}
		constDecl := ir.ConstDecl{Type: &ir.TypeValExpr{Typ: typ}}
		constDecl.Exprs = []*ir.ConstExpr{
			&ir.ConstExpr{
				Decl:  &constDecl,
				VName: &ast.Ident{Name: name},
				Val:   expr,
			},
		}
		return pkg.ImportIR(&ir.Declarations{
			Consts: []*ir.ConstDecl{&constDecl},
		})
	}
	return baseBuilder{
		name:  "ConstBuilder:" + funcName(f),
		build: buildConst,
	}
}

type registerMacro struct {
	name string
	impl cpevelements.MacroImpl
}

// RegisterMacro registers the implementation of a meta function.
func RegisterMacro(name string, impl cpevelements.MacroImpl) Builder {
	return &registerMacro{
		name: name,
		impl: impl,
	}
}

func (b *registerMacro) Build(bld *builder.Builder, _ *impl.Stdlib, pkg *builder.FilePackage) (err error) {
	pkgIR := pkg.IR()
	defer func() {
		if err != nil {
			err = fmterr.Internal(errors.Errorf("cannot set the implementation of %s.%s: %v", pkgIR.Name, b.name, err))
		}
	}()
	fun := pkgIR.FindFunc(b.name)
	if fun == nil {
		return errors.Errorf("cannot find the function in the package IR")
	}
	macro, ok := fun.(*ir.Macro)
	if !ok {
		return errors.Errorf("type %T is not %s", fun, reflect.TypeFor[*ir.Macro]())
	}
	macro.BuildSynthetic = b.impl
	return nil
}

func (b *registerMacro) Name() string {
	return fmt.Sprintf("%T:%s", b, b.name)
}
