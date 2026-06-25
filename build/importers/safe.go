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

package importers

import (
	"errors"
	"io/fs"
	"runtime/debug"

	"github.com/gx-org/gx/build/fmterr"
)

// SafeLoader captures panics and transform them into errors.
type SafeLoader struct {
	l Loader
}

// NewSafeLoader returns a new loader given a set of importers.
func NewSafeLoader(l Loader) *SafeLoader {
	return &SafeLoader{l: l}
}

// Load a package given its path.
func (sl *SafeLoader) Load(bld Builder, path string) (pkg Package, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmterr.Internal(errors.New(string(debug.Stack())))
		}
	}()
	return sl.l.Load(bld, path)
}

// BuildFiles builds a source for an incremental package.
func (sl *SafeLoader) BuildFiles(bld Builder, packagePath, packageName string, fs fs.FS, filenames []string) (_ Package, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmterr.Internal(errors.New(string(debug.Stack())))
		}
	}()
	return bld.BuildFiles(packagePath, packageName, fs, filenames)
}

// Importers used by the loader.
func (sl *SafeLoader) Importers() []Importer {
	return sl.l.Importers()
}
