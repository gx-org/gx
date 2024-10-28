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
	"path/filepath"

	"github.com/gx-org/gx/build/builder"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/interp"
	"github.com/gx-org/gx/stdlib/impl"
)

type (
	// Builder a builtin package.
	Builder interface {
		// Name of the builder (for debugging purpose only).
		Name() string
		// Build the given package.
		Build(*builder.Builder, *impl.Stdlib, builder.Package) error
	}

	// PackageBuilder builds a builtin package, that is a package composed of GX and Go code.
	PackageBuilder struct {
		// FullPath is the full path to the package, including its path and its name.
		FullPath string
		// Builders are the steps required to build the package.
		Builders []Builder
	}
)

// Build a package from its description.
func Build(bld *builder.Builder, impl *impl.Stdlib, pkgBuilder PackageBuilder) (builder.Package, error) {
	path, name := filepath.Split(pkgBuilder.FullPath)
	if len(path) > 0 {
		path = path[:len(path)-1]
	}
	pkg, err := bld.NewPackage(path, name)
	if err != nil {
		return nil, err
	}
	for _, builder := range pkgBuilder.Builders {
		if err := builder.Build(bld, impl, pkg); err != nil {
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
		FName: name,
	}
	fn.Impl = T{Func: Func{
		Func: fn,
		Impl: fnBuiltin,
	}}
	return fn
}
