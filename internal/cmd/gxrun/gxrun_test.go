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
	"embed"
	"fmt"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gx-org/gx/internal/cmd/gxrun"
	"github.com/gx-org/gx/src/gx/parser"
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

// wantPrefix starts the comment declaring the output expected from running a GX test file.
const wantPrefix = "Want:"

// wantOutput returns the output expected from running a GX test file.
// The file declares it with a comment starting with wantPrefix followed
// by the lines expected on the standard output.
func wantOutput(path string) (string, error) {
	file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ParseComments)
	if err != nil {
		return "", err
	}
	var wants []string
	for _, cmt := range file.Comments {
		if want, ok := strings.CutPrefix(cmt.Text(), wantPrefix); ok {
			wants = append(wants, want)
		}
	}
	if len(wants) != 1 {
		return "", fmt.Errorf("%s: found %d comments starting with %q, want 1", path, len(wants), wantPrefix)
	}
	want := strings.TrimSpace(wants[0])
	if want == "" {
		return "", nil
	}
	return want + "\n", nil
}

// testdata stores the GX test files run by TestRun.
//
//go:embed testdata/*.gx
var testdata embed.FS

// TestRun runs the Main function of each GX file in testdata and checks
// the standard output against the output declared in the file.
func TestRun(t *testing.T) {
	// The run command reads its source file from the disk:
	// copy the embedded GX test files into a temporary directory first.
	dir := t.TempDir()
	if err := os.CopyFS(dir, testdata); err != nil {
		t.Fatal(err)
	}
	paths, err := filepath.Glob(filepath.Join(dir, "testdata", "*.gx"))
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) == 0 {
		t.Fatal("no GX test file found in testdata")
	}
	for _, path := range paths {
		t.Run(strings.TrimSuffix(filepath.Base(path), ".gx"), func(t *testing.T) {
			want, err := wantOutput(path)
			if err != nil {
				t.Fatal(err)
			}
			got, err := run(t, path)
			if err != nil {
				t.Fatalf("gx run %s: %v", path, err)
			}
			if got != want {
				t.Errorf("gx run %s output:\ngot:\n%s\nwant:\n%s", path, got, want)
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
