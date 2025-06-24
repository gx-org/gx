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

// Package incbld_test tests building a GX package incrementally.
package incbld_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/gx-org/gx/build/builder"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/golang/backend"
	gxtesting "github.com/gx-org/gx/tests/testing"
	"github.com/gx-org/gx/tests"
)

const packageName = "test"

func packagePrefix(src string) string {
	return fmt.Sprintf(`
package %s

%s
`, packageName, src)
}

func checkError(t *testing.T, prefix string, err error, want string) error {
	if !strings.HasPrefix(want, prefix) {
		return err
	}
	want = want[len(prefix):]
	if err == nil {
		t.Errorf("got nil error but want an error that contains %q", want)
		return nil
	}
	got := err.Error()
	if !strings.Contains(got, want) {
		t.Errorf("got %q error but want an error that contains %q", got, want)
		return nil
	}
	return nil
}

func exportedNames(pkg *ir.Package) []string {
	var names []string
	for fn := range pkg.ExportedFuncs() {
		names = append(names, fn.Name())
	}
	return names
}

const (
	buildError = "builderror:"
	runError   = "runerror:"
)

type cellTest struct {
	runFunc string
	want    string
	src     string
}

func testCells(t *testing.T, cells []cellTest) (*gxtesting.Runner, *builder.IncrementalPackage) {
	rtm := backend.New(tests.CoreBuilder())
	pkg := rtm.Builder().NewIncrementalPackage(packageName)
	runner, err := gxtesting.NewRunner(rtm, 0)
	if err != nil {
		t.Fatal(err)
	}
	testCellsWithRunner(t, cells, pkg, runner)
	return runner, pkg
}

func testCellsWithRunner(t *testing.T, cells []cellTest, pkg *builder.IncrementalPackage, runner *gxtesting.Runner) {
	for id, cell := range cells {
		err := pkg.Build(cell.src)
		err = checkError(t, buildError, err, cell.want)
		if err != nil {
			t.Fatalf("cannot compile cell %d:\n%+v", id, err)
		}
		if cell.runFunc == "" {
			continue
		}
		fn := pkg.IR().FindFunc(cell.runFunc).(*ir.FuncDecl)
		if fn == nil {
			t.Errorf("cannot find function %s after evaluating cell %d", cell.runFunc, id)
			continue
		}
		var got string
		t.Run(fmt.Sprintf("cell[%d]", id), func(t *testing.T) {
			_, got, err = runner.Run(fn, nil)
		})
		err = checkError(t, runError, err, cell.want)
		if err != nil {
			t.Errorf("\n%+v", err)
			continue
		}
		want := strings.TrimPrefix(cell.want, "\n")
		want = strings.TrimSuffix(want, "\n")
		if got != want {
			t.Errorf("cell [%d] %s:\ngot:\n%s\nwant:\n%s", id, cell.runFunc, got, want)
		}
	}
}

func TestIncrementalPackage(t *testing.T) {
	_, pkg := testCells(t, []cellTest{
		{
			runFunc: "Func1",
			want:    "int32(5)",
			src: packagePrefix(`
func funcA() int32 {
	return 5
}

func Func1() int32 {
	return funcA()
}
`),
		},
		{
			runFunc: "Func1",
			want:    "int32(55)",
			src: packagePrefix(`
func funcA() int32 {
	return 55
}
`),
		},
	})
	want := []string{"Func1"}
	got := exportedNames(pkg.IR())
	if !cmp.Equal(got, want) {
		t.Errorf("incorrect exported names: got %v but want %v", got, want)
	}
	// Check twice to rule out symbols being accumulated in the IR.
	got = exportedNames(pkg.IR())
	if !cmp.Equal(got, want) {
		t.Errorf("incorrect exported names: got %v but want %v", got, want)
	}
}

func TestSyntaxError(t *testing.T) {
	testCells(t, []cellTest{
		{
			src: packagePrefix(`
func syntaxError() {
`),
			want: buildError + "expected '}', found 'EOF'",
		},
	})
}

func TestMissingFieldInStructInstance(t *testing.T) {
	runner, pkg := testCells(t, []cellTest{
		{
			runFunc: "",
			src: packagePrefix(`
type S struct {
	f1 float32
}

func New() S {
	return S{f1:1.0}
}
`),
		},
	})
	fNew := pkg.IR().FindFunc("New").(*ir.FuncDecl)
	if fNew == nil {
		t.Fatalf("cannot find function New after defining it")
	}
	vals, _, err := runner.Run(fNew, nil)
	if err != nil {
		t.Fatal(err)
	}
	cTest := &cellTest{
		src: packagePrefix(`
type S struct {
	f1 float32
	f2 float32
}

func F2(s S) float32 {
	return s.f2
}
`),
		want: runError + "field f2 undefined",
	}
	if err = pkg.Build(cTest.src); err != nil {
		t.Fatal(err)
	}
	fF2 := pkg.IR().FindFunc("F2").(*ir.FuncDecl)
	if fF2 == nil {
		t.Fatalf("cannot find function F2 after defining it")
	}
	_, _, err = runner.RunWithArgs(fF2, nil, vals, nil)
	if err := checkError(t, runError, err, cTest.want); err != nil {
		t.Error(err)
	}
}

func TestStaticVars(t *testing.T) {
	testCells(t, []cellTest{
		{
			runFunc: "A",
			want:    "float32(5)",
			src: packagePrefix(`
const a = 5

func A() float32 {
	return a
}
`),
		},
		{
			runFunc: "A",
			want:    "float32(4)",
			src: packagePrefix(`
const a = 4
`),
		},
	})
}

func TestStructOverCell(t *testing.T) {
	testCells(t, []cellTest{
		{
			runFunc: "New",
			want: `
test.S{
	A: float32(5),
}
`,
			src: packagePrefix(`
type S struct {
	A float32
}

func New() S {
	return S{A:5}
}
`),
		},
		{
			src: packagePrefix(`
func (s S) Add(b float32) float32 {
	return s.a+b
}
`),
			want: buildError + "not supported",
		},
	})
}
