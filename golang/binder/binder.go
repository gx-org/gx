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

// Package binder generates bindings given a GX package.
package binder

import (
	"maps"
	"os"
	"path/filepath"
	"slices"

	"github.com/pkg/errors"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/golang/binder/bindings"
	"github.com/gx-org/gx/golang/binder/ccbindings"
	"github.com/gx-org/gx/golang/binder/gobindings"
)

// Binders available.
var Binders = map[string]bindings.New{
	"go": gobindings.New,
	"cc": ccbindings.New,
}

// Bind generates the package for a given language and file name.
func Bind(language string, targetFolder string, pkg *ir.Package) error {
	bndConstructor, ok := Binders[language]
	if !ok {
		return errors.Errorf("cannot create bindings for language %q: no binder available. Available binders are %v", language, slices.Collect(maps.Keys(Binders)))
	}
	bnd, err := bndConstructor(pkg)
	if err != nil {
		return err
	}
	for _, file := range bnd.Files() {
		filePath := file.BuildFilePath(targetFolder, pkg)
		if err := os.MkdirAll(filepath.Dir(filePath), 0750); err != nil {
			return err
		}
		f, err := os.Create(filePath)
		if err != nil {
			return errors.Errorf("cannot create target file: %v", err)
		}
		if err := file.WriteBindings(f); err != nil {
			return err
		}
		if err := f.Close(); err != nil {
			return err
		}
	}
	return nil
}
