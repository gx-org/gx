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
package impgo_test

import (
	"embed"
	"strings"
	"testing"

	"github.com/gx-org/gx/build/builder"
	"github.com/gx-org/gx/build/builder/testbuild"
	"github.com/gx-org/gx/build/importers/embedpkg"
	"github.com/gx-org/gx/golang/backend"
	"github.com/gx-org/gx/internal/testing/testrtm"

	_ "github.com/gx-org/gx/internal/cmd/impgo/tests/returnatom/returnatom_importgo"
)

//go:embed tests
var fs embed.FS

var sources = &testbuild.SourceFolder{
	Name: "goimp",
	FS:   fs,
	Filter: func(name string) bool {
		return !strings.HasSuffix(name, "importgo.gx")
	},
}

func TestImportGoPackage(t *testing.T) {
	bld := builder.New(embedpkg.New())
	bck := backend.New(bld)
	testbuild.RunFactory(t, bld.Loader().Importers(), testrtm.Factory(
		bck, sources,
	))
}
