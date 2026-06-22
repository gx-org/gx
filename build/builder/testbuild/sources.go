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
	"fmt"
	"path"
	"strings"

	"github.com/gx-org/gx/api"
	"github.com/gx-org/gx/build/builder"
	"github.com/gx-org/gx/internal/testing/cmperr"
)

// SourceFolder is a folder with a name and a filesystem with the source files.
type SourceFolder struct {
	// Name of the folder.
	Name string
	// FS is the filesystem with the source files.
	FS embed.FS
}

var _ TestFactory = SourceFolder{}

// BuildTests creates a set of test from a file system with a testdata folder.
func (sf SourceFolder) BuildTests(rtm *api.Runtime) ([]Test, error) {
	dir, err := sf.FS.ReadDir(".")
	if err != nil {
		return nil, fmt.Errorf("cannot read filesystem: %w", err)
	}
	var tests []Test
	for _, entry := range dir {
		if entry.IsDir() {
			continue
		}
		if !strings.HasSuffix(entry.Name(), ".gx") {
			continue
		}
		name := entry.Name()
		src, err := sf.FS.ReadFile(name)
		if err != nil {
			return nil, fmt.Errorf("cannot read %s: %v", entry.Name(), err)
		}
		tests = append(tests, &source{
			folder: sf,
			name:   name,
			rtm:    rtm,
			src:    string(src),
		})
	}
	return tests, nil
}

// source runs a test from a source file loaded from a file system.
type source struct {
	folder SourceFolder
	name   string
	rtm    *api.Runtime
	src    string
}

var _ WithName = (*source)(nil)

func (t *source) Source() string {
	return t.src
}

func (t *source) Run(b *Builder) error {
	bld := builder.NewWithLoader(&b.imp)
	pkg, err := bld.BuildFiles("", "testdata", t.folder.FS, []string{t.name})
	if _, err = cmperr.Compare(pkg.IR(), err); err != nil {
		return &compileError{src: t.src, err: err}
	}
	return nil
}

func (t *source) Name() string {
	return path.Base(t.name)
}
