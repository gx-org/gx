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

// Package builtin defines interfaces and helper methods to provide builtin functions.
package builtin

import (
	"fmt"
	"go/ast"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/gx-org/gx/build/importers"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/interp/elements"
	"github.com/gx-org/gx/interp/engine"
	"github.com/gx-org/gx/interp"
	"github.com/gx-org/gx/interp/materialise"
)

type (
	// BuilderParam groups all the parameters to build the IR of builtin packages.
	BuilderParam struct {
		Builder importers.Builder
		FS      fs.FS
	}

	// Builder a builtin package.
	Builder interface {
		// Name of the builder (for debugging purpose only).
		Name() string
		// Build the given package.
		Build(*BuilderParam, importers.FilePackage) error
	}

	// PackageBuilder builds a builtin package, that is a package composed of GX and Go code.
	PackageBuilder struct {
		// FullPath is the full path to the package, including its path and its name.
		FullPath string
		// Builders are the steps required to build the package.
		Builders []Builder
	}
)

// NilShape is a nil value for a slice of axis length.
var NilShape = elements.NilFromType(ir.IntSliceType())

// ToShapeString represents elements into an array shape.
func ToShapeString(els []ir.Element) string {
	ss := make([]string, len(els))
	for i, el := range els {
		ss[i] = fmt.Sprintf("[%s]", el)
	}
	return strings.Join(ss, "")
}

// ToShapeResult converts elements into a slice of axis lengths and an error.
func ToShapeResult(els ...ir.Element) ([]ir.Element, error) {
	shape, err := elements.NewSlice(ir.IntSliceType(), els)
	if err != nil {
		return nil, err
	}
	return []ir.Element{shape, elements.NilError()}, nil
}

// Build a package from its description.
func (param BuilderParam) Build(pkgBuilder PackageBuilder) (importers.Package, error) {
	path, name := filepath.Split(pkgBuilder.FullPath)
	if len(path) > 0 {
		path = path[:len(path)-1]
	}
	pkg, err := param.Builder.NewPackage(path, name)
	if err != nil {
		return nil, err
	}
	for _, builder := range pkgBuilder.Builders {
		if err := builder.Build(&param, pkg); err != nil {
			return nil, err
		}
	}
	return pkg, nil
}

// Func is a standard library builtin function.
type Func struct {
	Func *ir.FuncBuiltin
	Impl interp.FuncBuiltin
}

// Implementation of the builtin function.
func (f Func) Implementation() any {
	return f.Impl
}

// Name of the builtin function.
func (f Func) Name() string {
	return f.Func.Name()
}

type builtinFuncImpl interface {
	ir.FuncImpl
	~struct{ Func }
}

// IRFuncBuiltin returns an *ir.FuncBuiltin for a function with the given name, implementation
// (contained in type T), interpreter slot, and package. Satisfies FuncBuilder.BuildFuncIR.
//
// The type of the function is computed at compile time from the call to the function.
func IRFuncBuiltin[T builtinFuncImpl](name string, fnBuiltin interp.FuncBuiltin, pkg *ir.Package) *ir.FuncBuiltin {
	fn := &ir.FuncBuiltin{
		Src:   &ast.FuncDecl{Name: &ast.Ident{Name: name}},
		FFile: &ir.File{Package: pkg},
	}
	fn.Impl = T{Func: Func{
		Func: fn,
		Impl: fnBuiltin,
	}}
	return fn
}

// Materialiser returns the materialiser from the evaluation environment.
func Materialiser(env engine.Env) materialise.Materialiser {
	return env.Engine().ArrayOps().(materialise.Materialiser)
}
