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

package testbuild

import (
	"fmt"
	"go/ast"
	"strings"

	"github.com/google/go-cmp/cmp"
	"github.com/gx-org/gx/build/fmterr"
	"github.com/gx-org/gx/build/ir"
)

// WantPrefix is the prefix in the comment indicating the result of a test.
const WantPrefix = "Want:"

func commentsInFunc(fn *ir.FuncDecl, prefix string) []*ast.CommentGroup {
	pkg := fn.File().Package
	startFunc := fn.Src.Pos()
	fileName := pkg.FSet.Position(startFunc).Filename
	fileDecl := pkg.File(fileName)
	endFunc := fn.Src.End()
	var cmts []*ast.CommentGroup
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

// WantOutput returns the string after the "Want:" string in the comments.
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

// BuildGot builds a string from values returned by an evaluation.
func BuildGot[T any](out []T, toString func(T) string) string {
	if len(out) == 0 {
		return ""
	}
	if len(out) == 1 {
		return toString(out[0])
	}
	bld := strings.Builder{}
	for i, s := range out {
		gotI := strings.TrimSpace(toString(s))
		fmt.Fprintf(&bld, "%d: %s\n", i, gotI)
	}
	return strings.TrimSpace(bld.String())
}

func textFromComment(cmt *ast.CommentGroup, prefix string) string {
	text := strings.TrimPrefix(cmt.Text(), prefix)
	for strings.HasSuffix(text, "\n") {
		text = strings.TrimSuffix(text, "\n")
	}
	return strings.TrimSpace(text)
}

// CheckOutput compares values returned by an evaluation and the Want
// string from the comments of a function declaration.
func CheckOutput[T any](fn *ir.FuncDecl, outs []T, got string) error {
	wantOutCmt, err := wantOutput(fn)
	if err != nil {
		return fmt.Errorf("%s: incorrect output declaration: %v",
			fmterr.At(fn.File().Package.FSet, fn.Src).String(),
			err)
	}
	if wantOutCmt == nil {
		return fmt.Errorf("%s expected a Want: directive", fmterr.At(fn.File().Package.FSet, fn.Src))
	}
	want := textFromComment(wantOutCmt, WantPrefix)
	if got != want {
		gotTypes := make([]string, len(outs))
		for i, val := range outs {
			gotTypes[i] = fmt.Sprintf("%T", val)
		}
		return fmt.Errorf("test run error:\n%s: incorrect output:\ngot (%s):\n%s\nwant:\n%s\ndiff:\n%s",
			fmterr.At(fn.File().Package.FSet, wantOutCmt), strings.Join(gotTypes, ","), got, want, cmp.Diff(got, want))
	}
	return nil
}
