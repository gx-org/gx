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
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"github.com/gx-org/gx/api"
	"github.com/gx-org/gx/api/options"
	"github.com/gx-org/gx/api/runner"
	"github.com/gx-org/gx/api/trace"
	"github.com/gx-org/gx/build/importers"
	"github.com/gx-org/gx/build/ir"
)

// DefaultTracer is the tracer used if the device does not define a tracer.
// Nil by default.
var DefaultTracer trace.Callback

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
	plat := s.device.Backend()
	s.opts = append(s.opts, options.Process(plat, optionFactories)...)
}

// Runtime used by the device.
func (s *DeviceSetup) Runtime() *api.Runtime {
	return s.device.Runtime()
}

// PackageSetup returns a package using this device setup.
func (s *DeviceSetup) PackageSetup(pkg importers.Package) *PackageCompileSetup {
	return &PackageCompileSetup{devSetup: s, pkg: pkg}
}

// Device returns the device for which packages are compiled for.
func (s *DeviceSetup) Device() *api.Device {
	return s.device
}

// PackageCompileSetup has all package level data to compile functions.
type PackageCompileSetup struct {
	devSetup *DeviceSetup
	pkg      importers.Package
}

func (s *PackageCompileSetup) canCompileGraph() bool {
	_, ok := s.devSetup.Runtime().Backend().(runner.EagerBackend)
	return !ok
}

// Setup returns the device of the compile setup.
func (s *PackageCompileSetup) Setup() *DeviceSetup {
	return s.devSetup
}

// Package returns the builder package.
func (s *PackageCompileSetup) Package() importers.Package {
	return s.pkg
}

// IR returns the intermediate representation of the package.
func (s *PackageCompileSetup) IR() *ir.Package {
	return s.pkg.IR()
}

// Tracer returns the tracer used by the package.
func (s *PackageCompileSetup) Tracer() trace.Callback {
	if s.devSetup.tracer != nil {
		return s.devSetup.tracer
	}
	return DefaultTracer
}

func findFunction(pkg *ir.Package, fnName string) (*ir.FuncDecl, error) {
	fn := pkg.FindFunc(fnName)
	if fn == nil {
		return nil, errors.Errorf("function %s not found in package %s", fnName, pkg.Name.Name)
	}
	fDecl, ok := fn.(*ir.FuncDecl)
	if !ok {
		return nil, errors.Errorf("cannot compile function %s in package %s: incorrect type %T", fnName, pkg.Path(), fn)
	}
	return fDecl, nil
}

func findMethod(pkg *ir.Package, recvName, fnName string) (*ir.FuncDecl, error) {
	nType := pkg.Decls.TypeByName(recvName)
	if nType == nil {
		return nil, errors.Errorf("package %s has no type %s", pkg.Name.Name, recvName)
	}
	fn := ir.MethodByName(nType, fnName)
	if fn == nil {
		return nil, errors.Errorf("type %s.%s has no method %s", pkg.Name.Name, recvName, fnName)
	}
	fDecl, ok := fn.(*ir.FuncDecl)
	if !ok {
		return nil, errors.Errorf("cannot compile method %s.%s in package %s: incorrect type %T", recvName, fnName, pkg.Name.Name, fn)
	}
	return fDecl, nil
}

// NewFunc returns a new function cache given a package compilation setup.
func (s *PackageCompileSetup) NewFunc(recvName, fnName string) (Func, error) {
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
	return newFuncCache(s, fn), nil
}

// NewFuncFromDecl returns a new function given its IR definition.
func (s *PackageCompileSetup) NewFuncFromDecl(decl *ir.FuncDecl) Func {
	return newFuncCache(s, decl)
}

func optionsToString(opts []options.PackageOption) string {
	optsS := make([]string, len(opts))
	for i, opt := range opts {
		optsS[i] = opt.String()
	}
	return fmt.Sprintf("{%s}", strings.Join(optsS, ", "))
}
