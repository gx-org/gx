// Copyright 2026 Google LLC
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

// Package gxmain compiles GX packages to run a Main function.
package gxmain

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"strings"

	"github.com/gx-org/gx/api"
	"github.com/gx-org/gx/api/hostio"
	"github.com/gx-org/gx/api/runner"
	"github.com/gx-org/gx/api/values"
	"github.com/gx-org/gx/build/builder"
	"github.com/gx-org/gx/build/fmterr"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/golang/backend/kernels"
)

// FromFile runs a main function from a single file path.
func FromFile(rtm *api.Runtime, fs fs.FS, filenames []string) (ir.PkgFunc, []hostio.Value, error) {
	pkg, err := rtm.Builder().BuildFiles("", "", fs, filenames)
	if err != nil {
		// Build errors start with a new line: remove it to print the first error after the command prefix.
		return nil, nil, errors.New(strings.TrimPrefix(err.Error(), "\n"))
	}
	return Run(rtm, pkg.IR())
}

const mainFuncName = "Main"

// Run the Main function of a given package.
func Run(rtm *api.Runtime, pkg *ir.Package) (ir.PkgFunc, []hostio.Value, error) {
	// Find the function.
	pkgFn := pkg.FindFunc(mainFuncName)
	if pkgFn == nil {
		return nil, nil, fmt.Errorf("function %s is undeclared in package %s", mainFuncName, pkg.Path())
	}
	fn, ok := pkgFn.(*ir.FuncDecl)
	if !ok {
		return pkgFn, nil, fmterr.Errorf(pkg.FSet, pkgFn.Node(), "func %s must have a body", mainFuncName)
	}
	var outs []hostio.Value
	var err error
	if fn.FuncType().Nature.CompEval {
		outs, err = runCompEvalMain(rtm.Builder(), fn)
	} else {
		outs, err = runGraphMain(rtm, fn)
	}
	return pkgFn, outs, err
}

func runGraphMain(rtm *api.Runtime, fn *ir.FuncDecl) ([]hostio.Value, error) {
	fType := fn.FuncType()
	if fType.GenParams.Fields.Len() > 0 {
		return nil, fmterr.Errorf(fn.File().FileSet(), fn.Node(), "func %s must have no type parameters", mainFuncName)
	}
	if fType.Params.Len() > 0 {
		return nil, fmterr.Errorf(fn.File().FileSet(), fn.Node(), "func %s must have no parameters", mainFuncName)
	}
	dev, err := rtm.Device(0)
	if err != nil {
		return nil, fmt.Errorf("cannot create device: %w", err)
	}
	compiled, err := runner.Trace(dev, fn, nil, nil, nil)
	if err != nil {
		return nil, err
	}
	tr := &tracer{}
	outs, err := compiled.Run(nil, nil, tr)
	if err != nil {
		return nil, err
	}
	hostOuts, err := hostio.ToHost(kernels.DefaultAllocator(), flatten(outs))
	if err != nil {
		return nil, err
	}
	// Append the values passed to the trace builtin after the values returned by Main.
	return append(hostOuts, tr.tracedOut...), nil
}

// runCompEvalMain evaluates a compeval Main function at compile time and prints the values it returns.
// The parameters of Main, if any, are symbolic: the values returned by Main are then expressions
// of these parameters.
func runCompEvalMain(bld *builder.Builder, fn *ir.FuncDecl) ([]hostio.Value, error) {
	_, outs, err := builder.CompEvalFunc(bld, fn)
	if err != nil {
		return nil, err
	}
	vals := make([]hostio.Value, len(outs))
	for i, out := range outs {
		vals[i] = values.NewIRNode(out)
	}
	return vals, nil
}

// flatten replaces the slices in a list of values by their elements, recursively.
func flatten(vals []hostio.Value) []hostio.Value {
	var flat []hostio.Value
	for _, val := range vals {
		slice, ok := val.(*values.Slice)
		if !ok {
			flat = append(flat, val)
			continue
		}
		elements := make([]hostio.Value, slice.Len())
		for i := range elements {
			elements[i] = slice.Element(i)
		}
		flat = append(flat, flatten(elements)...)
	}
	return flat
}

// valueString returns the GX source code of a value.
func valueString(file *ir.File, val hostio.Value) string {
	if definer, ok := val.(ir.StringDefiner); ok {
		return definer.DefineString(file)
	}
	return val.SourceString(file)
}

// Printf a string representation of the values.
func Printf(w io.Writer, fn ir.PkgFunc, outs []hostio.Value) error {
	for _, out := range outs {
		if _, err := fmt.Fprintln(w, valueString(fn.File(), out)); err != nil {
			return err
		}
	}
	return nil
}
