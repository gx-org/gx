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

// Package options specifies options for packages.
package options

import (
	"github.com/gx-org/backend/platform"
	"github.com/gx-org/gx/api/values"
)

type (
	// PackageOptionFactory creates options given a backend.
	PackageOptionFactory func(platform.Platform) PackageOption

	// PackageOption is an option specific to a package.
	PackageOption interface {
		Package() string
	}

	// PackageVarSetValue sets the value of a package level static variable.
	PackageVarSetValue struct {
		// Pck is the package owning the variable.
		Pkg string
		// Index of the variable in the package definition.
		Var string
		// Value of the static variable for the compiler.
		Value values.Value
	}
)

// Package for which the option has been built.
func (p PackageVarSetValue) Package() string {
	return p.Pkg
}
