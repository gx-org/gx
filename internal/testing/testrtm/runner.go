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
	"github.com/gx-org/gx/api/options"
	"github.com/gx-org/gx/api/tracer"
	"github.com/gx-org/gx/api/values"
	gxfmt "github.com/gx-org/gx/base/fmt"
	"github.com/gx-org/gx/build/builder/testbuild"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/golang/backend/kernels"
)

type (
	// Runner runs test functions.
	Runner struct {
		dev *api.Device
	}

	testTracer struct {
		nTrace int
		trace  strings.Builder
	}
)

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
func (r *Runner) Run(fn *ir.FuncDecl, options []options.PackageOption) ([]values.Value, string, error) {
	return r.RunWithArgs(fn, nil, nil, options)
}

// RunWithArgs compiles a function into a XLA graph, runs it, and returns the result.
func (r *Runner) RunWithArgs(fn *ir.FuncDecl, recv values.Value, args []values.Value, options []options.PackageOption) ([]values.Value, string, error) {
	runner, err := tracer.Trace(r.dev, fn, recv, args, options)
	if err != nil {
		return nil, "", err
	}
	tracer := testTracer{}
	outs, err := runner.Run(nil, nil, &tracer)
	if err != nil {
		return nil, "", err
	}
	flattenOuts, err := values.ToHost(kernels.Allocator(), flatten(outs))
	if err != nil {
		return outs, "", err
	}
	all := testbuild.BuildGot(flattenOuts, func(val values.Value) string {
		return valueToString(fn.File(), val)
	})
	all += tracer.trace.String()
	all = strings.TrimSpace(all)
	return flattenOuts, all, nil
}

func valueToString(file *ir.File, val values.Value) string {
	if src, isSrc := val.(ir.StringDefiner); isSrc {
		return src.DefineString(file)
	}
	return val.SourceString(file)
}

func (r *testTracer) Trace(file *ir.File, call *ir.FuncCallExpr, vals []values.Value) error {
	if r.nTrace == 0 {
		r.trace.WriteString("\nTrace:\n")
	}
	vals, err := values.ToHost(kernels.Allocator(), vals)
	if err != nil {
		return err
	}
	pos := file.FileSet().Position(call.Src.Pos())
	fmt.Fprintf(&r.trace, "%s:%d", filepath.Base(pos.Filename), r.nTrace)
	r.nTrace++
	r.trace.WriteString("\n")
	for _, val := range vals {
		valS := valueToString(file, val)
		valS = gxfmt.Indent(valS)
		r.trace.WriteString(valS)
	}
	r.trace.WriteString("\n")
	return nil
}

func flatten(out []values.Value) []values.Value {
	var flat []values.Value
	for _, v := range out {
		slice, ok := v.(*values.Slice)
		if !ok {
			flat = append(flat, v)
			continue
		}
		vals := make([]values.Value, slice.Len())
		for i := 0; i < slice.Len(); i++ {
			vals[i] = slice.Element(i)
		}
		flat = append(flat, flatten(vals)...)
	}
	return flat
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
