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

package gxfmt_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gx-org/gx/internal/cmd/gxfmt"
)

const (
	unformatted = `package test

func  f( x int32 ) int32 {
return x
}
`
	formatted = `package test

func f(x int32) int32 {
	return x
}
`
)

// runFmt runs the fmt command given a standard input and some arguments.
// It returns what the command printed on the standard output.
func runFmt(t *testing.T, stdin string, args ...string) (string, error) {
	t.Helper()
	cmd := gxfmt.Cmd()
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true
	// Always pass a non-nil slice: cobra parses os.Args if the arguments are nil.
	cmd.SetArgs(append([]string{}, args...))
	cmd.SetIn(strings.NewReader(stdin))
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	err := cmd.Execute()
	return stdout.String(), err
}

// writeFiles writes files in a new temporary directory and returns that directory.
func writeFiles(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for name, content := range files {
		path := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

// checkFiles checks the content of files in a directory.
func checkFiles(t *testing.T, dir string, want map[string]string) {
	t.Helper()
	for name, wantContent := range want {
		got, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != wantContent {
			t.Errorf("content of %s:\ngot:\n%s\nwant:\n%s", name, got, wantContent)
		}
	}
}

func TestStdin(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want string
	}{
		{
			name: "file",
			src:  unformatted,
			want: formatted,
		},
		{
			name: "declarations",
			src:  "func  f( x int32 ) int32 {\nreturn x\n}\n",
			want: "func f(x int32) int32 {\n\treturn x\n}\n",
		},
		{
			name: "statements",
			src:  "x:=f( 1 )\nreturn  x\n",
			want: "x := f(1)\nreturn x\n",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := runFmt(t, test.src)
			if err != nil {
				t.Fatalf("gx fmt: %v", err)
			}
			if got != test.want {
				t.Errorf("gx fmt output:\ngot:\n%s\nwant:\n%s", got, test.want)
			}
		})
	}
}

func TestStdinWriteError(t *testing.T) {
	_, err := runFmt(t, unformatted, "-w")
	if err == nil {
		t.Fatal("gx fmt -w with no path: got no error, want an error")
	}
	if want := "standard input"; !strings.Contains(err.Error(), want) {
		t.Errorf("gx fmt -w with no path: got error %q, want an error containing %q", err, want)
	}
}

func TestPrintFiles(t *testing.T) {
	files := map[string]string{
		"a.gx": unformatted,
		"b.gx": formatted,
	}
	dir := writeFiles(t, files)
	got, err := runFmt(t, "", filepath.Join(dir, "a.gx"), filepath.Join(dir, "b.gx"))
	if err != nil {
		t.Fatalf("gx fmt: %v", err)
	}
	if want := formatted + formatted; got != want {
		t.Errorf("gx fmt output:\ngot:\n%s\nwant:\n%s", got, want)
	}
	// Without -w, files are left untouched.
	checkFiles(t, dir, files)
}

func TestWriteDirectory(t *testing.T) {
	dir := writeFiles(t, map[string]string{
		"a.gx":         unformatted,
		"formatted.gx": formatted,
		"sub/b.gx":     unformatted,
		".hidden.gx":   unformatted,
		"notgx.txt":    unformatted,
	})
	stdout, err := runFmt(t, "", "-w", dir)
	if err != nil {
		t.Fatalf("gx fmt -w: %v", err)
	}
	if stdout != "" {
		t.Errorf("gx fmt -w printed %q on the standard output, want nothing", stdout)
	}
	checkFiles(t, dir, map[string]string{
		"a.gx":         formatted,
		"formatted.gx": formatted,
		"sub/b.gx":     formatted,
		".hidden.gx":   unformatted,
		"notgx.txt":    unformatted,
	})
}

func TestWriteExplicitFile(t *testing.T) {
	dir := writeFiles(t, map[string]string{
		"notgx.txt": unformatted,
	})
	// Files given as arguments are always formatted, whatever their extension.
	if _, err := runFmt(t, "", "--write", filepath.Join(dir, "notgx.txt")); err != nil {
		t.Fatalf("gx fmt --write: %v", err)
	}
	checkFiles(t, dir, map[string]string{
		"notgx.txt": formatted,
	})
}

func TestErrors(t *testing.T) {
	dir := writeFiles(t, map[string]string{
		"bad.gx":      "package test\n\nfunc f( {\n",
		"good.gx":     unformatted,
		"sub/bad.gx":  "func f() {}\n",
		"sub/good.gx": unformatted,
	})
	_, err := runFmt(t, "", "-w",
		filepath.Join(dir, "bad.gx"),
		filepath.Join(dir, "missing.gx"),
		filepath.Join(dir, "good.gx"),
		filepath.Join(dir, "sub"),
	)
	if err == nil {
		t.Fatal("gx fmt -w: got no error, want an error")
	}
	for _, want := range []string{
		filepath.Join(dir, "bad.gx") + ":3:",
		filepath.Join(dir, "missing.gx"),
		filepath.Join(dir, "sub", "bad.gx") + ":1:",
	} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("gx fmt -w: got error %q, want an error containing %q", err, want)
		}
	}
	// Errors do not prevent the other files from being formatted.
	checkFiles(t, dir, map[string]string{
		"bad.gx":      "package test\n\nfunc f( {\n",
		"good.gx":     formatted,
		"sub/bad.gx":  "func f() {}\n",
		"sub/good.gx": formatted,
	})
}
