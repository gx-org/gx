// Copyright 2026 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package testrtm provides utilities to test a runtime with GX code.
package testrtm

import (
	"fmt"
	"sort"
	"strings"

	"github.com/gx-org/gx/build/ir"
)

func noTestFuncOk(pkg *ir.Package) bool {
	for _, file := range pkg.Files {
		for _, grp := range file.Src.Comments {
			if strings.HasPrefix(grp.Text(), "No test function") {
				return true
			}
		}
	}
	return false
}

func findTests(pkg *ir.Package) []*ir.FuncDecl {
	var funs []*ir.FuncDecl
	for fn := range pkg.ExportedFuncs() {
		if !strings.HasPrefix(fn.Name(), "Test") {
			continue
		}
		funcDecl, ok := fn.(*ir.FuncDecl)
		if !ok {
			continue
		}
		funs = append(funs, funcDecl)
	}
	sort.Slice(funs, func(i, j int) bool {
		return funs[i].Name() < funs[j].Name()
	})
	return funs
}

// FindTests finds all the tests at the top-level of a filesystem.
func FindTests(pkg *ir.Package) ([]*ir.FuncDecl, error) {
	funs := findTests(pkg)
	if len(funs) == 0 && !noTestFuncOk(pkg) {
		return nil, fmt.Errorf("no test found in package %s. Add to the source file:\n\t// No test function\nif no test functions are expected", pkg.Path())
	}
	return funs, nil
}
