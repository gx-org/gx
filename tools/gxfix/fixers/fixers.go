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

// Package fixers implements functions to fix GX code.
package fixers

import (
	"fmt"
	"go/ast"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/gx-org/gx/api"
	"github.com/gx-org/gx/build/ir"
	gxtesting "github.com/gx-org/gx/tests/testing"
)

// Fixers list all the fixers.
var Fixers = []Fixer{
	fixTestOutput,
}

// Fixer fixes a GX file.
type Fixer func(rtm *api.Runtime, path string, f *ast.File) (fixed bool, err error)

func commentsInFunc(file *ast.File, fun *ast.FuncDecl, prefix string) *ast.CommentGroup {
	for _, cmt := range file.Comments {
		pos := cmt.Pos()
		if pos < fun.Pos() || pos > fun.End() {
			continue
		}
		if !strings.HasPrefix(strings.TrimSpace(cmt.Text()), prefix) {
			continue
		}
		return cmt
	}
	return nil
}

const wantPrefix = "Want:"

func runTestForResult(rtm *api.Runtime, pkg *ir.Package, funcDecl *ast.FuncDecl) (string, error) {
	options, err := gxtesting.BuildCompileOptions(rtm, pkg)
	if err != nil {
		return "", err
	}
	runner, err := gxtesting.NewRunner(rtm, 0)
	if err != nil {
		return "", err
	}
	for _, irFunc := range pkg.Funcs {
		if irFunc.Name() != funcDecl.Name.Name {
			continue
		}
		_, res, err := runner.Run(irFunc.(*ir.FuncDecl), options)
		return res, err
	}
	return "", fmt.Errorf("function %s not found", funcDecl.Name.Name)
}

func findPackageFiles(path string, f *ast.File) (fs.FS, []string, error) {
	dir, _ := filepath.Split(path)
	dirFS := os.DirFS(dir)
	dirEntries, err := dirFS.(fs.ReadDirFS).ReadDir(".")
	if err != nil {
		return nil, nil, err
	}
	names := []string{}
	for _, entry := range dirEntries {
		if !isGXFile(entry) {
			continue
		}
		names = append(names, entry.Name())
	}
	return dirFS, names, nil
}

func fixTestOutput(rtm *api.Runtime, path string, f *ast.File) (fixed bool, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("cannot fix %s: %w", path, err)
		}
	}()
	if !strings.HasSuffix(path, "_test.gx") {
		return
	}
	if strings.Contains(path, "tests/errors") || strings.Contains(path, "/testfiles/imports/") {
		return
	}
	fs, packageFiles, err := findPackageFiles(path, f)
	if err != nil {
		return false, err
	}
	pkg, err := rtm.Builder().BuildFiles(path, fs, packageFiles)
	if err != nil {
		return false, err
	}
	for _, decl := range f.Decls {
		funcDecl, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}
		if !strings.HasPrefix(funcDecl.Name.Name, "Test") {
			continue
		}
		wantComment := commentsInFunc(f, funcDecl, gxtesting.WantPrefix)
		if wantComment == nil {
			continue
		}
		want, err := runTestForResult(rtm, pkg.IR(), funcDecl)
		if err != nil {
			return false, err
		}

		wantComment.List = []*ast.Comment{
			&ast.Comment{
				Slash: wantComment.List[0].Slash,
				Text:  "// " + wantPrefix,
			},
		}
		for _, line := range strings.Split(want, "\n") {
			wantComment.List = append(wantComment.List,
				&ast.Comment{
					Slash: wantComment.End() + 2,
					Text:  "// " + line,
				},
			)
		}
		funcDecl.Body.Rbrace = wantComment.End() + 1
	}
	return true, nil
}
