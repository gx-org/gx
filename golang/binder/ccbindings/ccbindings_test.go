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

package ccbindings_test

import (
	"io"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gx-org/gx/build/builder"
	"github.com/gx-org/gx/golang/binder/ccbindings"
	gxtesting "github.com/gx-org/gx/tests/testing"
)

var tests = []struct {
	path    string
	hWants  []string
	ccWants []string
}{
	{
		path: "github.com/gx-org/gx/tests/bindings/dtypes",
		ccWants: []string{
			"DtypesIR::Load",
			"DtypesIR::BuildFor",
		},
		hWants: []string{
			"class DtypesIR",
			"class Dtypes",
		},
	},
}

func checkGenerate(t *testing.T, gen func(io.Writer) error, wants []string) {
	out := &strings.Builder{}
	if err := gen(out); err != nil {
		t.Fatalf("cannot generate bindings:\n%+v\n\nOutput generated:\n%s", err, gxtesting.NumberLines(out.String()))
	}
	got := out.String()
	for _, want := range wants {
		if !strings.Contains(got, want) {
			t.Errorf("%q cannot be found in generated bindings:\n%s", want, got)
		}
	}
}

func run(t *testing.T, bld *builder.Builder, path string, hWants, ccWants []string) {
	pkg, err := bld.Build(path)
	if err != nil {
		t.Errorf("cannot get package %s builder: %v", path, err)
		return
	}
	bnd, err := ccbindings.New(pkg.IR())
	if err != nil {
		t.Fatal(err)
	}
	files := bnd.Files()
	hFile, ccFile := files[0], files[1]
	checkGenerate(t, hFile.WriteBindings, hWants)
	checkGenerate(t, ccFile.WriteBindings, ccWants)
}

func TestCCBindings(t *testing.T) {
	builder := gxtesting.NewBuilderStaticSource(nil)
	for _, test := range tests {
		_, name := filepath.Split(test.path)
		t.Run(name, func(t *testing.T) {
			run(t, builder, test.path, test.hWants, test.ccWants)
		})
	}
}
