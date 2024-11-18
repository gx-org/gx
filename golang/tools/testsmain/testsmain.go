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

// Package main generates a main function to run all the GX tests of a file.
package main

import (
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"github.com/gx-org/gx/golang/template"
	"github.com/gx-org/gx/tools/gxflag"
)

var (
	targetName   = flag.String("target_name", "testmain.go", "target filename")
	targetFolder = flag.String("target_folder", "", "target folder location")

	goPackageName  = flag.String("go_package_name", "", "name of the Go package")
	goFiles        = gxflag.StringList("go_files", "list of GX files to package")
	gxSourceFolder = flag.String("gx_test_folder", "", "folder with the GX files")
)

const mainSource = `
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

// Package main generates a main function to run all the GX tests of a file.
// Automatically generated from google3/third_party/gxlang/gx/golang/tools/testsmain.go.
//
// DO NOT EDIT
package {{.GoPackageName}}

import (
	"testing"
	"github.com/gx-org/gx/api"
)

var tests = []struct{
  name string
  test func(*testing.T)
}{
{{.Tests}}
}

func Run(t *testing.T, rtm *api.Runtime) {
	t.Helper()
	err := setupTest(rtm)
	if err != nil {
		t.Errorf("cannot run {{.GoPackageName}} tests: %+v", err)
		return
	}
	for _, test := range tests {
		t.Run("{{.GoPackageName}}."+test.name, test.test)
	}
}

`

type testmain struct {
	GoPackageName string
	Tests         string
}

func processDecl(file *ast.File) ([]string, error) {
	return nil, nil
}

func processSrcFile(fset *token.FileSet, name string) ([]string, error) {
	src, err := os.ReadFile(name)
	if err != nil {
		return nil, errors.Errorf("cannot read file: %v", err)
	}
	file, err := parser.ParseFile(fset, name, src, 0)
	if err != nil {
		return nil, errors.Errorf("cannot parser file: %v", err)
	}
	funcs := []string{}
	for _, decl := range file.Decls {
		funcDecl, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}
		funcName := funcDecl.Name.Name
		if !strings.HasPrefix(funcName, "Test") {
			continue
		}
		funcs = append(funcs, funcName)
	}
	return funcs, nil
}

func funcsToString(funcs []string) string {
	if len(funcs) == 0 {
		return ""
	}
	s := strings.Builder{}
	for _, fun := range funcs {
		s.WriteString(fmt.Sprintf("\t{name: \"%s\", test: %s},\n", fun, fun))
	}
	return s.String()
}

func processGoFiles(files []string) ([]string, error) {
	var testFuncs []string
	fset := token.NewFileSet()
	for _, fileName := range files {
		fileName = strings.TrimSpace(fileName)
		funcs, err := processSrcFile(fset, fileName)
		if fileName == "" {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("cannot process %s: %v", fileName, err)
		}
		testFuncs = append(testFuncs, funcs...)
	}
	if len(testFuncs) == 0 {
		return nil, fmt.Errorf("no test functions found")
	}
	return testFuncs, nil
}

func listGXSourceFiles(path string) ([]string, error) {
	files, err := ioutil.ReadDir(path)
	if err != nil {
		return nil, fmt.Errorf("cannot read directory %q: %v", path, err)
	}
	var gxSourceFiles []string
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		gxSourceFiles = append(gxSourceFiles, filepath.Join(path, file.Name()))
	}
	return gxSourceFiles, nil
}

func main() {
	flag.Parse()
	if *targetFolder == "" {
		*targetFolder = *gxSourceFolder
	}
	if *goPackageName == "" {
		*goPackageName = filepath.Base(*gxSourceFolder)
	}
	if len(*goFiles) == 0 {
		var err error
		*goFiles, err = listGXSourceFiles(*gxSourceFolder)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
	testFuncs, err := processGoFiles(*goFiles)
	if err != nil {
		fmt.Fprint(os.Stderr, err)
		os.Exit(1)
	}
	tests := funcsToString(testFuncs)
	pkg := testmain{
		GoPackageName: *goPackageName,
		Tests:         tests,
	}
	goTarget := filepath.Join(*targetFolder, *targetName)
	if err := template.Exec(mainSource, goTarget, pkg); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
