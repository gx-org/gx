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

package testing

import (
	"fmt"
	"go/ast"
	"go/token"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/gx-org/gx/api"
	"github.com/gx-org/gx/api/values"
	"github.com/gx-org/gx/build/builder"
	"github.com/gx-org/gx/build/fmterr"
	"github.com/gx-org/gx/build/importers/embedpkg"
	"github.com/gx-org/gx/build/importers"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/golang/backend/kernels"
	"github.com/gx-org/gx/interp"
	"github.com/gx-org/gx/stdlib/impl"
	"github.com/gx-org/gx/stdlib"

	// Packages statically loaded for tests.
	_ "github.com/gx-org/gx/tests/bindings/basic"
	_ "github.com/gx-org/gx/tests/bindings/cartpole"
	_ "github.com/gx-org/gx/tests/bindings/dtypes"
	_ "github.com/gx-org/gx/tests/bindings/encoding"
	_ "github.com/gx-org/gx/tests/bindings/imports"
	_ "github.com/gx-org/gx/tests/bindings/math"
	_ "github.com/gx-org/gx/tests/bindings/parameters"
	_ "github.com/gx-org/gx/tests/bindings/pkgvars"
	_ "github.com/gx-org/gx/tests/bindings/rand"
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

// NewBuilderStaticSource returns a builder using the embedpkg importer which
// embeds GX testing source files into their corresponding Go package.
func NewBuilderStaticSource(stdlibImpl *impl.Stdlib) *builder.Builder {
	return builder.New(importers.NewCacheLoader(
		stdlib.Importer(stdlibImpl),
		embedpkg.New(),
	))
}

// NewRunner returns a test runner given a device.
func NewRunner(rtm *api.Runtime, devID int) (*Runner, error) {
	dev, err := rtm.Device(devID)
	if err != nil {
		return nil, err
	}
	return &Runner{dev: dev}, nil
}

// Run compiles a function into a XLA graph, runs it, and returns the result.
func (r *Runner) Run(fn *ir.FuncDecl, options []interp.PackageOption) ([]values.Value, string, error) {
	return r.RunWithArgs(fn, nil, nil, options)
}

// RunWithArgs compiles a function into a XLA graph, runs it, and returns the result.
func (r *Runner) RunWithArgs(fn *ir.FuncDecl, recv values.Value, args []values.Value, options []interp.PackageOption) ([]values.Value, string, error) {
	runner, err := interp.Compile(r.dev, fn, recv, args, options)
	if err != nil {
		return nil, "", err
	}
	tracer := testTracer{}
	values, err := runner.Run(nil, nil, &tracer)
	if err != nil {
		return nil, "", err
	}
	all := buildGot(values) + tracer.trace.String()
	all = strings.TrimSpace(all)
	return values, all, nil
}

func (r *testTracer) Trace(fset *token.FileSet, call *ir.CallExpr, vals []values.Value) error {
	if r.nTrace == 0 {
		r.trace.WriteString("\nTrace:\n")
	}
	vals, err := values.ToHost(kernels.Allocator(), vals)
	if err != nil {
		return err
	}
	pos := fset.Position(call.Src.Pos())
	r.trace.WriteString(fmt.Sprintf("%s:%d", filepath.Base(pos.Filename), r.nTrace))
	r.nTrace++
	const indent = "  "
	r.trace.WriteString("\n" + indent)
	for _, val := range vals {
		valS := fmt.Sprint(val)
		valS = strings.ReplaceAll(valS, "\n", "\n"+indent)
		r.trace.WriteString(valS)
	}
	r.trace.WriteString("\n")
	return nil
}

// WantPrefix is the prefix in the comment indicating the result of a test.
const WantPrefix = "Want:"

func wantOutput(fn *ir.FuncDecl) (*ast.CommentGroup, error) {
	cmts := commentsInFunc(fn, WantPrefix)
	if len(cmts) == 0 {
		return nil, nil
	}
	if len(cmts) > 1 {
		return nil, fmterr.Errorf(
			fn.File().Package.FSet,
			cmts[1],
			"function %s declares more than one Want",
			fn.Name())
	}
	return cmts[0], nil
}

func flatten(out []values.Value) []values.Value {
	flat := []values.Value{}
	for _, v := range out {
		slice, ok := v.(*values.Slice)
		if !ok {
			flat = append(flat, v)
			continue
		}
		vals := make([]values.Value, slice.Size())
		for i := 0; i < slice.Size(); i++ {
			vals[i] = slice.Element(i)
		}
		flat = append(flat, flatten(vals)...)
	}
	return flat
}

func buildGot(out []values.Value) string {
	out, err := values.ToHost(kernels.Allocator(), flatten(out))
	if err != nil {
		return err.Error()
	}
	if len(out) == 0 {
		return ""
	}
	if len(out) == 1 {
		return fmt.Sprint(out[0])
	}
	bld := strings.Builder{}
	for i, s := range out {
		bld.WriteString(fmt.Sprintf("%d: %v\n", i, s))
	}
	return strings.TrimSpace(bld.String())
}

func commentsInFunc(fn *ir.FuncDecl, prefix string) []*ast.CommentGroup {
	pkg := fn.File().Package
	startFunc := fn.Src.Pos()
	fileName := pkg.FSet.Position(startFunc).Filename
	fileDecl := pkg.File(fileName)
	endFunc := fn.Src.End()
	cmts := []*ast.CommentGroup{}
	for _, cmt := range fileDecl.Src.Comments {
		pos := cmt.Pos()
		if pos < startFunc || pos > endFunc {
			continue
		}
		if !strings.HasPrefix(strings.TrimSpace(cmt.Text()), prefix) {
			continue
		}
		cmts = append(cmts, cmt)
	}
	return cmts
}

func textFromComment(cmt *ast.CommentGroup, prefix string) string {
	text := strings.TrimPrefix(cmt.Text(), prefix)
	for strings.HasSuffix(text, "\n") {
		text = strings.TrimSuffix(text, "\n")
	}
	return strings.TrimSpace(text)
}

func (r *Runner) run(t *testing.T, fn *ir.FuncDecl, options []interp.PackageOption) {
	t.Parallel()
	values, got, err := r.Run(fn, options)
	if err != nil {
		t.Errorf("runner.Run error:\n%+v", err)
		return
	}
	wantOutCmt, err := wantOutput(fn)
	if err != nil {
		t.Errorf("%s: incorrect output declaration: %v",
			fmterr.PosString(fn.File().Package.FSet, fn.Src.Pos()), err)
		return
	}
	if wantOutCmt == nil {
		t.Errorf("%s expected a Want: directive", fmterr.PosString(fn.File().Package.FSet, fn.Src.Pos()))
		return
	}
	want := textFromComment(wantOutCmt, WantPrefix)
	if got != want {
		gotTypes := make([]string, len(values))
		for i, val := range values {
			gotTypes[i] = fmt.Sprintf("%T", val)
		}
		t.Errorf("test run error:\n%s: incorrect output:\ngot (%s):\n%s\nwant:\n%s\ndiff:\n%s\n",
			fmterr.PosString(fn.File().Package.FSet, wantOutCmt.Pos()), strings.Join(gotTypes, ","), got, want, cmp.Diff(got, want))
	}
}
