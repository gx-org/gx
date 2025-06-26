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

package testing

import (
	"io/fs"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gx-org/gx/api"
	"github.com/gx-org/gx/build/importers/localfs"
)

// Session is a test session to a set of tests
// given a filesystem, a builder, and a backend.
type Session struct {
	rtm *api.Runtime
	fs  fs.ReadDirFS
}

// NewSession returns a new testing session given a runtime.
func NewSession(rtm *api.Runtime, fs fs.ReadDirFS) *Session {
	return &Session{rtm: rtm, fs: fs}
}

// TestFolder run the tests in a folder.
func (s *Session) TestFolder(t *testing.T, path string) {
	bld := s.rtm.Builder()
	pkg, err := localfs.ImportAt(bld, s.fs, path, path)
	if err != nil {
		t.Fatalf("\n%s:\n%+v", path, err)
	}
	if err := Validate(pkg.IR(), CheckSource); err != nil {
		t.Errorf("\n%s:\n%+v", path, err)
	}
	RunAll(t, s.rtm, pkg.IR(), err)
}

// UnitSession is a test session that reads files from
// a file system and, for each file:
//
//  1. Create a new builder.
//  2. Read and compile the file as its own package.
//     The package name needs to match the file name.
//  3. Run the tests present in the file.
//
// This is different to a Session where all packages
// will be built using the same builder.
type UnitSession struct {
	rtmF func() (*api.Runtime, error)
	fs   fs.ReadDirFS
}

// NewUnitSession returns a new testing session given a runtime.
func NewUnitSession(rtmF func() (*api.Runtime, error), fs fs.ReadDirFS) *UnitSession {
	return &UnitSession{rtmF: rtmF, fs: fs}
}

func (s *UnitSession) runFileTest(t *testing.T, path, fileName, packageName string) (int, error) {
	rtm, err := s.rtmF()
	if err != nil {
		return 0, err
	}
	pkg, err := rtm.Builder().BuildFiles(path, packageName, s.fs, []string{
		filepath.Join(path, fileName),
	})
	if err != nil {
		return 0, err
	}
	if err := Validate(pkg.IR(), CheckSource); err != nil {
		return 0, err
	}
	numTests := RunAll(t, rtm, pkg.IR(), err)
	return numTests, nil
}

// TestFolder run the tests in a folder.
func (s *UnitSession) TestFolder(t *testing.T, path string) {
	entries, err := s.fs.ReadDir(path)
	if err != nil {
		t.Fatal(err)
	}
	totalNumTests := 0
	errs := false
	for _, entry := range entries {
		if !localfs.IsGXFile(entry) {
			continue
		}
		fileName := entry.Name()
		packageName := strings.TrimSuffix(fileName, filepath.Ext(fileName))
		t.Run(packageName, func(t *testing.T) {
			numTests, err := s.runFileTest(t, path, fileName, packageName)
			if err != nil {
				errs = true
				t.Errorf("\n%s:\n%+v", path, err)
				return
			}
			if numTests == 0 {
				errs = true
				t.Errorf("no test found in %s", entry.Name())
				return
			}
			totalNumTests += numTests
		})
	}
	if !errs && totalNumTests == 0 {
		t.Errorf("no unit test run in %s", path)
	}
}
