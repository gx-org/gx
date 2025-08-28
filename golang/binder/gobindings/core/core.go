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
func BuildPackage(bld builder.Builder, pkgPath string) (*Package, error) {
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
	irPkg  *ir.Package
	device *api.Device

	optionFactories []options.PackageOptionFactory
	opts            []options.PackageOption
}

// Setup returns a package handle for a device and a set of options.
func (pkg *Package) Setup(dev *api.Device, optionFactories []options.PackageOptionFactory) *PackageCompileSetup {
	s := &PackageCompileSetup{
		pkg:    pkg,
		irPkg:  pkg.pkg.IR(),
		device: dev,
	}
	s.AppendOptions(optionFactories...)
	return s
}

// Device returns the device of the compile setup.
func (s *PackageCompileSetup) Device() *api.Device {
	return s.device
}

// BuildDep builds a dependency.
func BuildDep[T any](setup *PackageCompileSetup, at int, builder func(*Package, *api.Device, []options.PackageOptionFactory) T) T {
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
	s.opts = options.Process(plat, optionFactories)
}

// FuncCache provides a compilation cache for a function.
type FuncCache struct {
	setup            *PackageCompileSetup
	recvName, fnName string

	runner   tracer.CompiledFunc
	err      error
	initOnce sync.Once
}

// NewCache returns a new function cache given a package compilation setup.
func (s *PackageCompileSetup) NewCache(recvName, fnName string) *FuncCache {
	return &FuncCache{setup: s, recvName: recvName, fnName: fnName}
}

func (c *FuncCache) findFunction() (*ir.FuncDecl, error) {
	pkg := c.setup.irPkg
	fn := pkg.FindFunc(c.fnName)
	if fn == nil {
		return nil, errors.Errorf("package %s has no function %s", pkg.Name.Name, c.fnName)
	}
	fDecl, ok := fn.(*ir.FuncDecl)
	if !ok {
		return nil, errors.Errorf("cannot compile function %s in package %s: incorrect type %T", c.fnName, pkg.FullName(), fn)
	}
	return fDecl, nil
}

func (c *FuncCache) findMethod() (*ir.FuncDecl, error) {
	pkg := c.setup.irPkg
	nType := pkg.Decls.TypeByName(c.recvName)
	if nType == nil {
		return nil, errors.Errorf("package %s has no type %s", pkg.Name.Name, c.recvName)
	}
	fn := nType.MethodByName(c.fnName)
	if fn == nil {
		return nil, errors.Errorf("type %s.%s has no method %s", pkg.Name.Name, c.recvName, c.fnName)
	}
	fDecl, ok := fn.(*ir.FuncDecl)
	if !ok {
		return nil, errors.Errorf("cannot compile method %s.%s in package %s: incorrect type %T", c.recvName, c.fnName, pkg.Name.Name, fn)
	}
	return fDecl, nil
}

func (c *FuncCache) buildRunner(recv values.Value, args []values.Value) (tracer.CompiledFunc, error) {
	var fn *ir.FuncDecl
	var err error
	if c.recvName != "" {
		fn, err = c.findMethod()
	} else {
		fn, err = c.findFunction()
	}
	if err != nil {
		return nil, err
	}
	runner, err := tracer.Trace(c.setup.device, fn, recv, args, c.setup.opts)
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
