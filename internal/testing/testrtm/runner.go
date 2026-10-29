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

package testrtm

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gx-org/gx/api"
	"github.com/gx-org/gx/api/hostio"
	"github.com/gx-org/gx/api/options"
	"github.com/gx-org/gx/api/runner"
	"github.com/gx-org/gx/api/trace"
	gxfmt "github.com/gx-org/gx/base/fmt"
	"github.com/gx-org/gx/build/builder/testbuild"
	"github.com/gx-org/gx/build/ir"
)

// Runner runs test functions.
type Runner struct {
	dev *api.Device
}

// NewRunner returns a test runner given a device.
func NewRunner(rtm *api.Runtime, devID int) (*Runner, error) {
	dev, err := rtm.Device(devID)
	if err != nil {
		return nil, err
	}
	return NewRunnerDevice(dev), nil
}

// NewRunnerDevice returns a new runner given a device.
func NewRunnerDevice(dev *api.Device) *Runner {
	return &Runner{dev: dev}
}

// Run compiles a function into a XLA graph, runs it, and returns the result.
func (r *Runner) Run(fn *ir.FuncDecl, options []options.PackageOption) ([]hostio.Value, string, error) {
	return r.RunWithArgs(fn, nil, nil, options)
}

// RunWithArgs compiles a function into a XLA graph, runs it, and returns the result.
func (r *Runner) RunWithArgs(fn *ir.FuncDecl, recv hostio.Value, args []hostio.Value, options []options.PackageOption) ([]hostio.Value, string, error) {
	var rec trace.Recorder
	outs, err := runner.TraceAndRun(r.dev, fn, recv, args, options, &rec)
	if err != nil {
		return nil, "", err
	}
	all := testbuild.BuildGot(outs, func(val hostio.Value) string {
		return ir.DefineString(fn.File(), val)
	})
	all += formatTraces(rec.Calls)
	all = strings.TrimSpace(all)
	return outs, all, nil
}

func formatTraces(calls []trace.Call) string {
	if len(calls) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("\nTrace:\n")
	for i, call := range calls {
		pos := call.File.FileSet().Position(call.Expr.Src.Pos())
		fmt.Fprintf(&b, "%s:%d\n", filepath.Base(pos.Filename), i)
		for _, val := range call.Values {
			valS := ir.DefineString(call.File, val)
			valS = gxfmt.Indent(valS)
			b.WriteString(valS)
			b.WriteString("\n")
		}
	}
	return b.String()
}

// Test runs a GX test and returns an error if the test fails.
func (r *Runner) Test(fn *ir.FuncDecl, options []options.PackageOption) error {
	values, got, err := r.Run(fn, options)
	if err != nil {
		return fmt.Errorf("runner.Run error:\n%+v", err)
	}
	return testbuild.CheckOutput(fn, values, got)
}

// RunAll compiles and runs all the test at a specified path.
// Returns the number of tests that have been run.
func RunAll(t *testing.T, rtm *api.Runtime, pkg *ir.Package) {
	fns, err := testbuild.MustFindTests(pkg, false)
	if err != nil {
		t.Errorf("\n%+v", err)
		return
	}
	options, err := BuildCompileOptions(rtm, pkg)
	if err != nil {
		t.Errorf("\n%+v", err)
		return
	}
	tRunner, err := NewRunner(rtm, 0)
	if err != nil {
		t.Errorf("\n%+v", err)
		return
	}
	numTests := 0
	for _, fn := range fns {
		name := fn.File().Package.Name.Name + "." + fn.Name()
		t.Run(name, func(t *testing.T) {
			numTests++
			if err := tRunner.Test(fn, options); err != nil {
				t.Error(err)
			}
		})
	}
	if numTests == 0 {
		t.Errorf("no test found in %s", pkg.Path())
	}
}
