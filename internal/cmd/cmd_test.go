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

package cmd

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// execute runs the gx command with some arguments and an empty standard input.
// It returns what the command printed on the standard error.
func execute(t *testing.T, args ...string) (string, error) {
	t.Helper()
	cmd := newRootCmd()
	// Always pass a non-nil slice: cobra parses os.Args if the arguments are nil.
	cmd.SetArgs(append([]string{}, args...))
	cmd.SetIn(strings.NewReader(""))
	cmd.SetOut(io.Discard)
	var stderr bytes.Buffer
	cmd.SetErr(&stderr)
	err := cmd.Execute()
	return stderr.String(), err
}

func TestFmtErrors(t *testing.T) {
	dir := t.TempDir()
	syntaxErrorPath := filepath.Join(dir, "syntax_error.gx")
	if err := os.WriteFile(syntaxErrorPath, []byte("package test\n\nfunc f( {\n"), 0644); err != nil {
		t.Fatal(err)
	}
	missingPath := filepath.Join(dir, "missing.gx")
	_, errMissing := os.Lstat(missingPath)
	if errMissing == nil {
		t.Fatalf("%s exists", missingPath)
	}
	tests := []struct {
		name       string
		args       []string
		wantStderr string
	}{
		{
			name:       "write with standard input",
			args:       []string{"fmt", "-w"},
			wantStderr: "gx fmt: cannot use -w with standard input\n",
		},
		{
			name:       "syntax error",
			args:       []string{"fmt", syntaxErrorPath},
			wantStderr: "gx fmt: " + syntaxErrorPath + ":3:9: expected ')', found '{'\n",
		},
		{
			name:       "missing path",
			args:       []string{"fmt", missingPath},
			wantStderr: "gx fmt: " + errMissing.Error() + "\n",
		},
		{
			name:       "unknown flag",
			args:       []string{"fmt", "-x"},
			wantStderr: "gx: unknown shorthand flag: 'x' in -x\n",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cmdLine := "gx " + strings.Join(test.args, " ")
			stderr, err := execute(t, test.args...)
			// An error makes the gx binary exit with a non-zero status.
			if err == nil {
				t.Errorf("%s: got no error, want an error", cmdLine)
			}
			if stderr != test.wantStderr {
				t.Errorf("%s: got standard error:\n%q\nwant:\n%q", cmdLine, stderr, test.wantStderr)
			}
		})
	}
}
