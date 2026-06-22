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
	"github.com/gx-org/gx/build/builder"
	"github.com/gx-org/gx/build/importers/embedpkg"
	"github.com/gx-org/gx/stdlib"

	// Packages statically loaded for tests.
	_ "github.com/gx-org/gx/tests/bindings/basic"
	_ "github.com/gx-org/gx/tests/bindings/cartpole"
	_ "github.com/gx-org/gx/tests/bindings/dtypes"
	_ "github.com/gx-org/gx/tests/bindings/encoding"
	_ "github.com/gx-org/gx/tests/bindings/generics"
	_ "github.com/gx-org/gx/tests/bindings/imports"
	_ "github.com/gx-org/gx/tests/bindings/math"
	_ "github.com/gx-org/gx/tests/bindings/parameters"
	_ "github.com/gx-org/gx/tests/bindings/pkgvars"
	_ "github.com/gx-org/gx/tests/bindings/rand"
	_ "github.com/gx-org/gx/tests/bindings/unexported"
)

// NewBuilderStaticSource returns a builder using the embedpkg importer which
// embeds GX testing source files into their corresponding Go package.
func NewBuilderStaticSource() *builder.Builder {
	return builder.New(
		stdlib.Importer(),
		embedpkg.New(),
	)
}
