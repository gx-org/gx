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

package core

import (
	"log"
	"sync"
	"time"

	"github.com/gx-org/gx/api"
	"github.com/gx-org/gx/api/hostio"
	"github.com/gx-org/gx/api/runner"
	"github.com/gx-org/gx/api/trace"
	"github.com/gx-org/gx/build/ir"
)

// Func wraps a GX function, sometimes in a cache to prevent unnecessary recompilation.
type Func interface {
	Device() *api.Device
	Build(recv hostio.Value, args []hostio.Value) (runner.CompiledFunc, error)
	Func() *ir.FuncDecl
	Tracer() trace.Callback
}

type coreFunc struct {
	pkg *PackageCompileSetup
	fn  *ir.FuncDecl
}

// Func returns the function being cached.
func (c coreFunc) Func() *ir.FuncDecl {
	return c.fn
}

// Device on which the function is going to run.
func (c coreFunc) Device() *api.Device {
	return c.pkg.devSetup.Device()
}

// Tracer to use to run the function.
func (c coreFunc) Tracer() trace.Callback {
	return c.pkg.Tracer()
}

// funcCache provides a compilation cache for a function.
type funcCache struct {
	coreFunc

	runner   runner.CompiledFunc
	err      error
	initOnce sync.Once
}

// newFuncCache returns a runner cache given a function.
func newFuncCache(pkg *PackageCompileSetup, fn *ir.FuncDecl) Func {
	coreF := coreFunc{pkg: pkg, fn: fn}
	if pkg.canCompileGraph() {
		return &funcCache{coreFunc: coreF}
	}
	return &funcNoCache{coreFunc: coreF}
}

func (c *funcCache) buildRunner(recv hostio.Value, args []hostio.Value) (_ runner.CompiledFunc, err error) {
	log.Printf("GX Tracing %s.%s with options %s", c.fn.FFile.Package.Name.Name, c.fn.Name(), optionsToString(c.pkg.devSetup.opts))
	now := time.Now()
	defer func() {
		log.Printf("GX Tracing %s.%s: DONE. Time: %v.", c.fn.FFile.Package.Name.Name, c.fn.Name(), time.Since(now))
	}()
	runner, err := runner.Trace(c.pkg.devSetup.device, c.fn, recv, args, c.pkg.devSetup.opts)
	if err != nil {
		return nil, err
	}
	return runner, nil
}

// Runner compiles the function if it has not been done before.
// Returns the function compiled or an error.
func (c *funcCache) Build(recv hostio.Value, args []hostio.Value) (runner.CompiledFunc, error) {
	c.initOnce.Do(func() {
		c.runner, c.err = c.buildRunner(recv, args)
	})
	return c.runner, c.err
}

// funcNoCache runs the function with no compile cache.
type funcNoCache struct {
	coreFunc
}

func (c *funcNoCache) Build(recv hostio.Value, args []hostio.Value) (runner.CompiledFunc, error) {
	return runner.Trace(c.pkg.devSetup.device, c.fn, recv, args, c.pkg.devSetup.opts)
}
