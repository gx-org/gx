// Copyright 2025 Google LLC
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

// Package fmtpath formats paths for C++ generated files.
package fmtpath

import (
	"path/filepath"
	"strings"

	"github.com/gx-org/gx/build/module"
)

var setModName string

// SetModuleName sets the module name so that go.mod is not used.
// Should be used only for testing.
func SetModuleName(name string) {
	setModName = name
}

func modName() (string, error) {
	if setModName != "" {
		return setModName, nil
	}
	mod, err := module.Current()
	if err != nil {
		return "", err
	}
	return mod.Name(), nil
}

func trimPrefix(path string) (string, error) {
	mName, err := modName()
	if err != nil {
		return "", err
	}
	if mName == path {
		return mName, nil
	}
	path = strings.TrimPrefix(path, mName)
	path = strings.TrimPrefix(path, "/")
	return path, nil
}

// Functions provided by the package.
type Functions struct{}

// HeaderPath formats the string for header guards.
func (Functions) HeaderPath(path string) string {
	return filepath.Base(path) + ".h"
}

// HeaderGuard formats the string for header guards.
func (Functions) HeaderGuard(guard string) string {
	guard, err := trimPrefix(guard)
	if err != nil {
		return ""
	}
	guard = strings.ReplaceAll(guard, "/", "_")
	guard = strings.ReplaceAll(guard, "-", "_")
	guard = strings.ReplaceAll(guard, ".", "_")
	guard = strings.ToUpper(guard)
	return guard + "_H"
}

// Namespace returns the C++ namespace.
func (Functions) Namespace(path string) string {
	namespace, err := trimPrefix(path)
	if err != nil {
		return ""
	}
	namespace = strings.ReplaceAll(namespace, "/", "::")
	return namespace
}

// PackagePath prepares a package path.
func (Functions) PackagePath(path string) string {
	return path
}
