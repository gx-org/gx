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
	"testing"

	"github.com/gx-org/gx/api"
	"github.com/gx-org/gx/build/importers/fsimporter"
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
	pkg, err := fsimporter.CompileDir(bld, s.fs, path)
	if err != nil {
		t.Fatalf("\n%s:\n%+v", path, err)
	}
	if err := Validate(pkg.IR(), CheckSource); err != nil {
		t.Errorf("\n%s:\n%+v", path, err)
	}
	RunAll(t, s.rtm, pkg.IR(), err)
}
