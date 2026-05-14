// Copyright 2026 Google LLC
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
	"embed"
	"io/fs"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gx-org/gx/build/builder"
	gxtesting "github.com/gx-org/gx/tests/testing"
)

// source runs a test from a source file loaded from a file system.
type source struct {
	fs   fs.FS
	name string

	src string
}

var _ WithName = (*source)(nil)

const testdata = "testdata"

// SourcesFrom creates a set of test from a file system with a testdata folder.
func SourcesFrom(t *testing.T, fs embed.FS) []Test {
	dir, err := fs.ReadDir(testdata)
	if err != nil {
		t.Fatalf("folder %s not found in the filesystem", testdata)
		return nil
	}
	var tests []Test
	for _, entry := range dir {
		if entry.IsDir() {
			continue
		}
		if !strings.HasSuffix(entry.Name(), ".gx") {
			continue
		}
		name := path.Join(testdata, entry.Name())
		src, err := fs.ReadFile(name)
		if err != nil {
			t.Fatalf("cannot read %s: %v", entry.Name(), err)
		}
		tests = append(tests, &source{
			fs:   fs,
			name: name,
			src:  string(src),
		})
	}
	return tests
}

func (t *source) Source() string {
	return t.src
}

func (t *source) Run(b *Builder) error {
	bld := builder.NewWithLoader(&b.imp)
	pkg, err := bld.BuildFiles("", testdata, t.fs, []string{t.name})
	if _, err = gxtesting.CompareToExpectedErrors(pkg.IR(), err); err != nil {
		return &compileError{src: t.src, err: err}
	}
	return nil
}

func (t *source) Name() string {
	baseName := path.Base(t.name)
	return strings.TrimSuffix(baseName, filepath.Ext(baseName))
}
