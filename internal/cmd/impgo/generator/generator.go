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

// Package generator provides abstractions for generators implementations.
package generator

import (
	"go/types"
	"path"
)

// Pkg are all the information required about a Go package.
type Pkg struct {
	Dir, Name string
	Pkg       *types.Package
}

// Path of the package being imported.
func (p Pkg) Path() string {
	return path.Join(p.Dir, p.Name)
}

// Target of the generator.
type Target struct {
	Src  Pkg
	Name string
}

// Path of the package being built for GX.
func (t Target) Path() string {
	return path.Join(t.Src.Dir, t.Name)
}

// New is a function to create a new generator.
type New func(Target) Generator

// Generator generates the source code for a file.
type Generator interface {
	// Generate generates all the source code and returns it.
	Generate() (string, error)
	// FileExtension returns the extension of the file for which the code is being generated.
	FileExtension() string
}
