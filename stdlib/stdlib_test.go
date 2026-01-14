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

package stdlib_test

import (
	"testing"

	"github.com/gx-org/gx/build/builder"
	"github.com/gx-org/gx/stdlib"
	gxtesting "github.com/gx-org/gx/tests/testing"
)

func TestStdlibValid(t *testing.T) {
	lib := stdlib.Importer(nil)
	bld := builder.New(lib)
	for _, path := range lib.Paths() {
		pkg, err := bld.Build(path)
		if err != nil {
			t.Fatalf("\n%+v", err)
		}
		if err := gxtesting.Validate(pkg.IR(), gxtesting.CheckSource); err != nil {
			t.Errorf("\n%s:\n%+v", path, err)
		}
	}
}
