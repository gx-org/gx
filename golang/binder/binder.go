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
	"io"

	"github.com/gx-org/gx/build/importers/localfs/binder"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/golang/binder/gobindings"
)

// Bind writes the binder in a writer given the name of a binder and a function
// to retrieve a compiled GX package.
func Bind(w io.Writer, pkg *ir.Package) error {
	return gobindings.Generate(binder.GenImports{}, w, pkg)
}
