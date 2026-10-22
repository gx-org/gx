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

package gxrun_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gx-org/gx/internal/cmd/gxrun"
)

// run executes the run command with some arguments.
// It returns what the command printed on the standard output.
func run(t *testing.T, args ...string) (string, error) {
	t.Helper()
	cmd := gxrun.Cmd()
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true
	// Always pass a non-nil slice: cobra parses os.Args if the arguments are nil.
	cmd.SetArgs(append([]string{}, args...))
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	err := cmd.Execute()
	return stdout.String(), err
}

// writeFile writes a GX source file named main.gx in a new temporary directory
// and returns its path.
func writeFile(t *testing.T, src string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "main.gx")
	if err := os.WriteFile(path, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestRun(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want string
	}{
		{
			name: "scalar",
			src: `package main

func Main() int32 {
	return 2 + 3
}
`,
			want: "int32(5)\n",
		},
		{
			name: "array",
			src: `package main

func Main() [3]float32 {
	return [3]float32{1, 2, 3} * 2
}
`,
			want: "[3]float32{2, 4, 6}\n",
		},
		{
			name: "multiple results",
			src: `package main

func Main() (int64, [2]bool, string) {
	return 4, [2]bool{true, false}, "Hello, GX"
}
`,
			want: "int64(4)\n[2]bool{true, false}\nHello, GX\n",
		},
		{
			name: "struct",
			src: `package main

type Point struct {
	X, Y float32
}

func Main() Point {
	return Point{X: 1, Y: 2}
}
`,
			want: "Point{\n\tX: float32(1),\n\tY: float32(2),\n}\n",
		},
		{
			name: "slice",
			src: `package main

func Main() []int32 {
	return []int32{int32(1), int32(2)}
}
`,
			want: "int32(1)\nint32(2)\n",
		},
		{
			name: "standard library",
			src: `package main

import "num"

func Main() [2][2]float32 {
	x := [2][2]float32{{1, 2}, {3, 4}}
	return num.MatMul(x, x)
}
`,
			want: "[2][2]float32{\n\t{7, 10},\n\t{15, 22},\n}\n",
		},
		{
			name: "trace",
			src: `package main

func Main() int64 {
	a := int64(2)
	trace(a)
	a = a + 1
	trace(a, [2]int64{a, a})
	return a
}
`,
			want: "int64(3)\nint64(2)\nint64(3)\n[2]int64{3, 3}\n",
		},
		{
			name: "no result",
			src: `package main

func Main() {
	trace(int32(1))
}
`,
			want: "int32(1)\n",
		},
		{
			name: "compeval",
			src: `package main

//gx:compeval
func Main() (int, string) {
	return 2 + 3, "compeval"
}
`,
			want: "5\n\"compeval\"\n",
		},
		{
			name: "compeval with parameters",
			src: `package main

//gx:compeval
func Main(a, b int) (int, int) {
	return a + b, a * b
}
`,
			want: "a+b\na*b\n",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := run(t, writeFile(t, test.src))
			if err != nil {
				t.Fatalf("gx run: %v", err)
			}
			if got != test.want {
				t.Errorf("gx run output:\ngot:\n%s\nwant:\n%s", got, test.want)
			}
		})
	}
}

func TestErrors(t *testing.T) {
	dir := t.TempDir()
	tests := []struct {
		name string
		args []string
		src  string
		want string
	}{
		{
			name: "no argument",
			want: "accepts 1 arg(s), received 0",
		},
		{
			name: "too many arguments",
			args: []string{"a.gx", "b.gx"},
			want: "accepts 1 arg(s), received 2",
		},
		{
			name: "missing file",
			args: []string{filepath.Join(dir, "missing.gx")},
			want: filepath.Join(dir, "missing.gx") + ": no such file or directory",
		},
		{
			name: "directory",
			args: []string{dir},
			want: dir + " is a directory",
		},
		{
			name: "syntax error",
			src:  "package main\n\nfunc Main( {\n",
			want: "main.gx:3:12: expected ')', found '{'",
		},
		{
			name: "compilation error",
			src:  "package main\n\nfunc Main() int32 {\n\treturn float32(1)\n}\n",
			want: "main.gx:4:9: cannot use float32 as int32 value in return statement",
		},
		{
			name: "no Main",
			src:  "package main\n\nfunc main() int32 {\n\treturn 1\n}\n",
			want: "function Main is undeclared in package main",
		},
		{
			name: "Main without a body",
			src:  "package main\n\nfunc Main() int32\n",
			want: "main.gx:3:1: func Main must have a body",
		},
		{
			name: "non-compeval Main with parameters",
			src:  "package main\n\nfunc Main(x int32) int32 {\n\treturn x\n}\n",
			want: "main.gx:3:1: func Main must have no parameters",
		},
		{
			name: "non-compeval Main with type parameters",
			src:  "package main\n\nfunc Main[T interface{ float32 }]() T {\n\treturn T(1)\n}\n",
			want: "main.gx:3:1: func Main must have no type parameters",
		},
		{
			name: "evaluation error",
			src:  "package main\n\nvar Size int\n\nfunc Main() [Size]float32 {\n\treturn [Size]float32{}\n}\n",
			want: "main.Main evaluation error",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			args := test.args
			if test.src != "" {
				args = []string{writeFile(t, test.src)}
			}
			stdout, err := run(t, args...)
			if err == nil {
				t.Fatalf("gx run %s: got no error, want an error containing %q", strings.Join(args, " "), test.want)
			}
			if !strings.Contains(err.Error(), test.want) {
				t.Errorf("gx run %s: got error:\n%s\nwant an error containing %q", strings.Join(args, " "), err, test.want)
			}
			if stdout != "" {
				t.Errorf("gx run %s printed %q on the standard output, want nothing", strings.Join(args, " "), stdout)
			}
		})
	}
}
