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

// Package importers provide a package loader implementation.
package importers

import (
	"github.com/gx-org/gx/build/builder"
)

// Importer loads a package given its path.
type Importer interface {
	// Support checks if the importer supports the import path given its prefix.
	Support(path string) bool
	// Import a path.
	Import(bld *builder.Builder, path string) (builder.Package, error)
}
