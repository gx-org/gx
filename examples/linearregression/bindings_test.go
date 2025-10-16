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

package linearregressionexperiment_test

import (
	"strings"
	"testing"

	"github.com/gx-org/gx/golang/binder/gobindings"
	gxtesting "github.com/gx-org/gx/tests/testing"

	_ "github.com/gx-org/gx/examples/linearregression"
)

func TestLinearRegressionBindings(t *testing.T) {
	bld := gxtesting.NewBuilderStaticSource(nil)
	out := &strings.Builder{}
	pkg, err := bld.Build("github.com/gx-org/gx/examples/linearregression")
	if err != nil {
		t.Fatalf("cannot generate bindings:\n%+v", err)
	}
	if err := gobindings.Write(out, pkg.IR()); err != nil {
		t.Fatalf("cannot generate bindings:\n%+v", err)
	}
	got := out.String()
	for _, want := range []string{
		"NewTarget",
		"NewLearner",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("%q cannot be found in generated bindings:\n%s", want, got)
		}
	}
}
