// Copyright 2025 Google LLC
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

// Package core provides helpers for Go bindings.
package core

import (
	"sync"

	"github.com/pkg/errors"
	"github.com/gx-org/gx/api"
	"github.com/gx-org/gx/api/options"
	"github.com/gx-org/gx/api/trace"
	"github.com/gx-org/gx/api/tracer"
	"github.com/gx-org/gx/api/values"
	"github.com/gx-org/gx/build/builder"
	"github.com/gx-org/gx/build/ir"
)

// Package is a handler to a package and its dependencies.
//
// Package is constructed by loading a package from its path.
type Package struct {
	pkg    builder.Package
	tracer trace.Callback

	deps []*Package
}

// NewPackage returns a GX loaded package.
func NewPackage(pkg builder.Package, deps []*Package) *Package {
	return &Package{pkg: pkg, deps: deps}
}

// BuildPackage builds a package given its path and a runtime.
func BuildPackage(bld *builder.Builder, pkgPath string) (*Package, error) {
	pkg, err := bld.Build(pkgPath)
	if err != nil {
		return nil, err
	}
	return NewPackage(pkg, nil), nil
}

// Package returns the IR package.
func (pkg *Package) Package() *ir.Package {
	return pkg.pkg.IR()
}

// Tracer returns the tracer used by the package.
func (pkg *Package) Tracer() trace.Callback {
	return pkg.tracer
}

// PackageCompileSetup has all package level data to compile functions.
type PackageCompileSetup struct {
	pkg    *Package
	device *api.Device

	optionFactories []options.PackageOptionFactory
	opts            []options.PackageOption
}

// Setup returns a package handle for a device and a set of options.
func (pkg *Package) Setup(dev *api.Device, optionFactories []options.PackageOptionFactory) *PackageCompileSetup {
	s := &PackageCompileSetup{
		pkg:    pkg,
		device: dev,
	}
	s.AppendOptions(optionFactories...)
	return s
}

// Device returns the device of the compile setup.
func (s *PackageCompileSetup) Device() *api.Device {
	return s.device
}

// IR returns the intermediate representation of the package.
func (s *PackageCompileSetup) IR() *ir.Package {
	return s.pkg.pkg.IR()
}

// Package used by the setup.
func (s *PackageCompileSetup) Package() *Package {
	return s.pkg
}

// BuildDep builds a dependency.
func BuildDep[T any](setup *PackageCompileSetup, at int, builder func(*Package, *api.Device, []options.PackageOptionFactory) (T, error)) (T, error) {
	return builder(setup.pkg.deps[at], setup.device, setup.optionFactories)
}

// Tracer returns the tracer that has been set.
// Can return nil.
func (s *PackageCompileSetup) Tracer() trace.Callback {
	return s.pkg.tracer
}

// AppendOptions returns the set of options used for compilation.
func (s *PackageCompileSetup) AppendOptions(optionFactories ...options.PackageOptionFactory) {
	plat := s.device.PlatformDevice().Platform()
	s.optionFactories = append(s.optionFactories, optionFactories...)
	s.opts = append(s.opts, options.Process(plat, optionFactories)...)
}

// FuncCache provides a compilation cache for a function.
type FuncCache struct {
	setup *PackageCompileSetup
	fn    *ir.FuncDecl

	runner   tracer.CompiledFunc
	err      error
	initOnce sync.Once
}

func findFunction(pkg *ir.Package, fnName string) (*ir.FuncDecl, error) {
	fn := pkg.FindFunc(fnName)
	if fn == nil {
		return nil, errors.Errorf("function %s not found in package %s", fnName, pkg.Name.Name)
	}
	fDecl, ok := fn.(*ir.FuncDecl)
	if !ok {
		return nil, errors.Errorf("cannot compile function %s in package %s: incorrect type %T", fnName, pkg.FullName(), fn)
	}
	return fDecl, nil
}

func findMethod(pkg *ir.Package, recvName, fnName string) (*ir.FuncDecl, error) {
	nType := pkg.Decls.TypeByName(recvName)
	if nType == nil {
		return nil, errors.Errorf("package %s has no type %s", pkg.Name.Name, recvName)
	}
	fn := nType.MethodByName(fnName)
	if fn == nil {
		return nil, errors.Errorf("type %s.%s has no method %s", pkg.Name.Name, recvName, fnName)
	}
	fDecl, ok := fn.(*ir.FuncDecl)
	if !ok {
		return nil, errors.Errorf("cannot compile method %s.%s in package %s: incorrect type %T", recvName, fnName, pkg.Name.Name, fn)
	}
	return fDecl, nil
}

// NewCache returns a new function cache given a package compilation setup.
func (s *PackageCompileSetup) NewCache(recvName, fnName string) (*FuncCache, error) {
	var fn *ir.FuncDecl
	var err error
	if recvName != "" {
		fn, err = findMethod(s.IR(), recvName, fnName)
	} else {
		fn, err = findFunction(s.IR(), fnName)
	}
	if err != nil {
		return nil, err
	}
	return s.NewCacheFromFunc(fn), nil
}

// NewCacheFromFunc returns a runner cache given a function.
func (s *PackageCompileSetup) NewCacheFromFunc(fn *ir.FuncDecl) *FuncCache {
	return &FuncCache{setup: s, fn: fn}
}

func (c *FuncCache) buildRunner(recv values.Value, args []values.Value) (tracer.CompiledFunc, error) {
	runner, err := tracer.Trace(c.setup.device, c.fn, recv, args, c.setup.opts)
	if err != nil {
		return nil, err
	}
	return runner, nil
}

// Runner compiles the function if it has not been done before.
// Returns the function compiled or an error.
func (c *FuncCache) Runner(recv values.Value, args []values.Value) (tracer.CompiledFunc, error) {
	c.initOnce.Do(func() {
		c.runner, c.err = c.buildRunner(recv, args)
	})
	return c.runner, c.err
}

// Func returns the function being cached.
func (c *FuncCache) Func() ir.Func {
	return c.fn
}
