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
	"github.com/gx-org/gx/api/trace"
	"github.com/gx-org/gx/api/tracer"
	"github.com/gx-org/gx/build/ir"
)

// funcCache provides a compilation cache for a function.
type funcCache struct {
	pkg *PackageCompileSetup
	fn  *ir.FuncDecl

	runner   tracer.CompiledFunc
	err      error
	initOnce sync.Once
}

// newFuncCache returns a runner cache given a function.
func newFuncCache(pkg *PackageCompileSetup, fn *ir.FuncDecl) Func {
	return &funcCache{pkg: pkg, fn: fn}
}

func (c *funcCache) buildRunner(recv hostio.Value, args []hostio.Value) (_ tracer.CompiledFunc, err error) {
	log.Printf("GX Tracing %s.%s with options %s", c.fn.FFile.Package.Name.Name, c.fn.Name(), optionsToString(c.pkg.devSetup.opts))
	now := time.Now()
	defer func() {
		log.Printf("GX Tracing %s.%s: DONE. Time: %v.", c.fn.FFile.Package.Name.Name, c.fn.Name(), time.Since(now))
	}()
	runner, err := tracer.Trace(c.pkg.devSetup.device, c.fn, recv, args, c.pkg.devSetup.opts)
	if err != nil {
		return nil, err
	}
	return runner, nil
}

// Runner compiles the function if it has not been done before.
// Returns the function compiled or an error.
func (c *funcCache) Runner(recv hostio.Value, args []hostio.Value) (tracer.CompiledFunc, error) {
	c.initOnce.Do(func() {
		c.runner, c.err = c.buildRunner(recv, args)
	})
	return c.runner, c.err
}

// Func returns the function being cached.
func (c *funcCache) Func() *ir.FuncDecl {
	return c.fn
}

// Device on which the function is going to run.
func (c *funcCache) Device() *api.Device {
	return c.pkg.devSetup.Device()
}

// Tracer to use to run the function.
func (c *funcCache) Tracer() trace.Callback {
	return c.pkg.Tracer()
}
