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

// DeviceSetup is a device with compile option.
// All packages build for this device will share the same options.
type DeviceSetup struct {
	device *api.Device
	tracer trace.Callback
	opts   []options.PackageOption
}

// NewDeviceSetup returns a device and a set of options.
func NewDeviceSetup(dev *api.Device, optionFactories []options.PackageOptionFactory) *DeviceSetup {
	s := &DeviceSetup{
		device: dev,
	}
	s.AppendOptions(optionFactories...)
	return s
}

// AppendOptions returns the set of options used for compilation.
func (s *DeviceSetup) AppendOptions(optionFactories ...options.PackageOptionFactory) {
	plat := s.device.PlatformDevice().Platform()
	s.opts = append(s.opts, options.Process(plat, optionFactories)...)
}

// Runtime used by the device.
func (s *DeviceSetup) Runtime() *api.Runtime {
	return s.device.Runtime()
}

// PackageSetup returns a package using this device setup.
func (s *DeviceSetup) PackageSetup(pkg builder.Package) *PackageCompileSetup {
	return &PackageCompileSetup{devSetup: s, pkg: pkg}
}

// Device returns the device for which packages are compiled for.
func (s *DeviceSetup) Device() *api.Device {
	return s.device
}

// PackageCompileSetup has all package level data to compile functions.
type PackageCompileSetup struct {
	devSetup *DeviceSetup
	pkg      builder.Package
}

// Setup returns the device of the compile setup.
func (s *PackageCompileSetup) Setup() *DeviceSetup {
	return s.devSetup
}

// Package returns the builder package.
func (s *PackageCompileSetup) Package() builder.Package {
	return s.pkg
}

// IR returns the intermediate representation of the package.
func (s *PackageCompileSetup) IR() *ir.Package {
	return s.pkg.IR()
}

// Tracer returns the tracer used by the package.
func (s *PackageCompileSetup) Tracer() trace.Callback {
	return s.devSetup.tracer
}

// FuncCache provides a compilation cache for a function.
type FuncCache struct {
	pkg *PackageCompileSetup
	fn  *ir.FuncDecl

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
	return &FuncCache{pkg: s, fn: fn}
}

func (c *FuncCache) buildRunner(recv values.Value, args []values.Value) (tracer.CompiledFunc, error) {
	runner, err := tracer.Trace(c.pkg.devSetup.device, c.fn, recv, args, c.pkg.devSetup.opts)
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
func (c *FuncCache) Func() ir.PkgFunc {
	return c.fn
}
