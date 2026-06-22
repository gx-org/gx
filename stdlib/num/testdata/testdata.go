// Copyright 2026 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package testdata stores the GX test files for the num package.
package testdata

import (
	"embed"

	"github.com/gx-org/gx/build/builder/testbuild"
)

//go:embed *.gx
var fs embed.FS

// Sources contains all the tests to run.
var Sources = testbuild.SourceFolder{
	Name: "num",
	FS:   fs,
}
