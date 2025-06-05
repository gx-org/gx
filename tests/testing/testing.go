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

// Package testing provides functions to run gx tests.
package testing

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/pkg/errors"
	"github.com/gx-org/backend/dtype"
	"github.com/gx-org/gx/api"
	"github.com/gx-org/gx/api/options"
	"github.com/gx-org/gx/build/fmterr"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/golang/binder/gobindings/types"
)

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
	if len(funs) == 0 {
		return nil, fmt.Errorf("no test found")
	}
	return funs, nil
}

const (
	setStatic = "setStatic"
)

func findVarDecl(pkg *ir.Package, name string) (*ir.VarExpr, error) {
	for _, decl := range pkg.Decls.Vars {
		for _, vr := range decl.Exprs {
			if vr.VName.Name == name {
				return vr, nil
			}
		}
	}
	return nil, errors.Errorf("cannot find variable %s in package %s", name, pkg.FullName())
}

func buildSetStaticOption(rtm *api.Runtime, pkg *ir.Package, cmdS []string) (options.PackageOption, error) {
	cmdS = cmdS[1:]
	const numArgs = 3
	if len(cmdS) != numArgs {
		return nil, errors.Errorf("%s: invalid parameters: got %d but want %d", setStatic, len(cmdS), numArgs)
	}
	valName, valType, valS := cmdS[0], cmdS[1], cmdS[2]
	varDecl, err := findVarDecl(pkg, valName)
	if err != nil {
		return nil, err
	}
	typeWant := varDecl.Type().Kind().String()
	if valType != typeWant {
		return nil, errors.Errorf("cannot use a value of type %s to set variable %s (type %s)", valType, valName, typeWant)
	}
	var val types.Bridger
	switch valType {
	case "int32":
		valInt, err := strconv.Atoi(valS)
		if err != nil {
			return nil, err
		}
		val = types.Int32(int32(valInt))
	case "int64":
		valInt, err := strconv.Atoi(valS)
		if err != nil {
			return nil, err
		}
		val = types.Int64(int64(valInt))
	case "intlen":
		valInt, err := strconv.Atoi(valS)
		if err != nil {
			return nil, err
		}
		val = types.DefaultInt(ir.Int(valInt))
	default:
		return nil, errors.Errorf("type %q not supported", valType)
	}
	return options.PackageVarSetValue{
		Pkg:   pkg.FullName(),
		Var:   valName,
		Value: val.Bridge().GXValue(),
	}, nil
}

func buildOption(rtm *api.Runtime, pkg *ir.Package, cmd string) (options.PackageOption, error) {
	cmdS := strings.Split(cmd, " ")
	if len(cmdS) == 0 {
		return nil, nil
	}
	switch cmdS[0] {
	case setStatic:
		return buildSetStaticOption(rtm, pkg, cmdS)
	default:
		return nil, errors.Errorf("compiler option command %q not supported", cmdS[0])
	}
}

// BuildCompileOptions from the source code of the package.
func BuildCompileOptions(rtm *api.Runtime, pkg *ir.Package) ([]options.PackageOption, error) {
	var options []options.PackageOption
	for _, file := range pkg.Files {
		for _, grp := range file.Src.Comments {
			if !strings.HasPrefix(grp.Text(), "Test options:") {
				continue
			}
			for _, cmt := range grp.List[1:] {
				text := strings.TrimSpace(cmt.Text)
				text = strings.TrimPrefix(text, "//")
				text = strings.TrimSpace(text)
				opt, err := buildOption(rtm, pkg, text)
				if err != nil {
					return nil, fmterr.Errorf(file.Package.FSet, cmt, "cannot build option %q: %v", text, err)
				}
				if opt == nil {
					continue
				}
				options = append(options, opt)
			}
		}
	}
	return options, nil
}

// RunAll compiles and runs all the test at a specified path.
// Returns the number of tests that have been run.
func RunAll(t *testing.T, rtm *api.Runtime, pkg *ir.Package, err error) (numTests int) {
	var errs *fmterr.Errors
	if err != nil {
		var ok bool
		errs, ok = err.(*fmterr.Errors)
		if !ok {
			t.Errorf("%+v", err)
			return
		}
	}
	hasExpectedErrors, err := CompareToExpectedErrors(pkg, errs)
	if err != nil {
		t.Errorf("\n%+v", err)
		return
	}
	if hasExpectedErrors {
		return
	}
	fns, err := FindTests(pkg)
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
		t.Error(err)
		return
	}
	for _, fn := range fns {
		name := fn.File().Package.Name.Name + "." + fn.Name()
		t.Run(name, func(t *testing.T) {
			numTests++
			tRunner.run(t, fn, options)
		})
	}
	return
}

// NumberLines returns a string where lines are prefixed by their number.
func NumberLines(s string) string {
	lines := strings.Split(s, "\n")
	if len(lines) < 2 {
		return s
	}
	padding := int(math.Ceil(math.Log10(float64(len(lines)))))
	paddingS := fmt.Sprintf("%%0%dd ", padding)
	for i, line := range lines {
		lines[i] = fmt.Sprintf(paddingS, i+1) + line
	}
	return strings.Join(lines, "\n")
}

// FetchAtom fetches an atomic value from a device.
func FetchAtom[T dtype.GoDataType](t *testing.T, atom types.Atom[T]) T {
	if atom == nil {
		t.Fatalf("cannot fetch value from a nil atom")
	}
	value, err := atom.Fetch()
	if err != nil {
		t.Fatalf("cannot fetch value:\n%+v", err)
	}
	return value.Value()
}

// FetchArray fetches an array from a device.
func FetchArray[T dtype.GoDataType](t *testing.T, array types.Array[T]) []T {
	if array == nil {
		t.Fatalf("cannot fetch value from a nil array")
	}
	value, err := array.Fetch()
	if err != nil {
		t.Fatalf("cannot fetch value:\n%+v", err)
	}
	return value.CopyFlat()
}
