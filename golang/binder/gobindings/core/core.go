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
	"github.com/gx-org/gx/build/ir"
)

// PackageIR is the GX package intermediate representation
// built for a given runtime, but not yet for a specific device.
//
// PackageIR is constructed by loading a package from its path.
type PackageIR struct {
	runtime *api.Runtime
	pkg     *ir.Package
	tracer  trace.Callback

	deps []*PackageIR
}

// NewPackageIR returns a GX loaded package.
func NewPackageIR(rtm *api.Runtime, pkg *ir.Package, deps []*PackageIR) *PackageIR {
	return &PackageIR{runtime: rtm, pkg: pkg, deps: deps}
}

// Package returns the IR package.
func (pkg *PackageIR) Package() *ir.Package {
	return pkg.pkg
}

// Tracer returns the tracer used by the package.
func (pkg *PackageIR) Tracer() trace.Callback {
	return pkg.tracer
}

// PackageCompileSetup has all package level data to compile functions.
type PackageCompileSetup struct {
	irPkg   *PackageIR
	device  *api.Device
	options []options.PackageOption
}

// Setup returns a package handle for a device and a set of options.
func (pkg *PackageIR) Setup(dev *api.Device, opts []options.PackageOption) *PackageCompileSetup {
	return &PackageCompileSetup{
		irPkg:   pkg,
		device:  dev,
		options: opts,
	}
}

// ProcessOptions processes compiling options.
func (pkg *PackageIR) ProcessOptions(optionFactories []options.PackageOptionFactory) []options.PackageOption {
	plat := pkg.runtime.Backend().Platform()
	opts := make([]options.PackageOption, len(optionFactories))
	for i, opt := range optionFactories {
		opts[i] = opt(plat)
	}
	return opts
}

// Device returns the device of the compile setup.
func (s *PackageCompileSetup) Device() *api.Device {
	return s.device
}

// BuildDep builds a dependency.
func BuildDep[T any](setup *PackageCompileSetup, at int, builder func(*PackageIR, *api.Device, []options.PackageOption) T) T {
	return builder(setup.irPkg.deps[at], setup.device, setup.options)
}

// Tracer returns the tracer that has been set.
// Can return nil.
func (s *PackageCompileSetup) Tracer() trace.Callback {
	return s.irPkg.tracer
}

// AppendOptions returns the set of options used for compilation.
func (s *PackageCompileSetup) AppendOptions(options ...options.PackageOptionFactory) {
	s.options = append(s.options, s.irPkg.ProcessOptions(options)...)
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
	pkg := c.setup.irPkg.pkg
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
	pkg := c.setup.irPkg.pkg
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
	runner, err := tracer.Trace(c.setup.device, fn, recv, args, c.setup.options)
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
