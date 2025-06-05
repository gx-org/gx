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

package gobindings_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/gx-org/gx/build/builder"
	"github.com/gx-org/gx/golang/binder"
	gxtesting "github.com/gx-org/gx/tests/testing"
)

var tests = []struct {
	path  string
	wants []string
}{
	{
		path: "github.com/gx-org/gx/tests/bindings/basic",
		wants: []string{
			"ReturnFloat32",
			"ReturnMultiple",
			"Basic struct {",
			"AddPrivate",
			"MarshalEmpty",
			"SetFloat",
			"// New returns a new instance of the basic structure.",
		},
	},
	{
		path: "github.com/gx-org/gx/tests/bindings/encoding",
		wants: []string{
			"field0",
		},
	},
	{
		path: "github.com/gx-org/gx/tests/bindings/imports",
		wants: []string{
			"NewBasic",
			"bindings/basic/basic_go_gx",
		},
	},
	{
		path: "github.com/gx-org/gx/tests/bindings/math",
		wants: []string{
			"MaxFloat32",
			"// ReturnMaxFloat32 returns the maximum float32.",
		},
	},
	{
		path: "github.com/gx-org/gx/tests/bindings/parameters",
		wants: []string{
			"AddFloat32",
			"AddFloat32s",
			"AddInt",
			"AddInts",
		},
	},
	{
		path: "github.com/gx-org/gx/tests/bindings/pkgvars",
		wants: []string{
			"Var1Static",
			"Size",
		},
	},
	{
		path: "github.com/gx-org/gx/tests/bindings/rand",
		wants: []string{
			"Sample",
		},
	},
	{
		path: "github.com/gx-org/gx/tests/bindings/dtypes",
		wants: []string{
			"Bool",
			"Float32",
			"Float64",
			"Int32",
			"Int64",
			"Uint32",
			"Uint64",
		},
	},
}

func run(t *testing.T, bld *builder.Builder, path string, wants []string) {
	pkg, err := bld.Build(path)
	if err != nil {
		t.Errorf("cannot get package %s builder:\n%+v", path, err)
		return
	}
	out := &strings.Builder{}
	if err := binder.GoBindings(out, pkg.IR()); err != nil {
		t.Fatalf("cannot generate bindings:\n%+v\n\nOutput generated:\n%s", err, gxtesting.NumberLines(out.String()))
	}
	got := out.String()
	for _, want := range wants {
		if !strings.Contains(got, want) {
			t.Errorf("%q cannot be found in generated bindings:\n%s", want, got)
		}
	}
}

func TestGoBindings(t *testing.T) {
	builder := gxtesting.NewBuilderStaticSource(nil)
	for _, test := range tests {
		_, name := filepath.Split(test.path)
		t.Run(name, func(t *testing.T) {
			run(t, builder, test.path, test.wants)
		})
	}
}
