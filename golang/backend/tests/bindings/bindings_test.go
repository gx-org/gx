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

package bindings_test

import (
	"testing"

	"github.com/gx-org/gx/api"
	"github.com/gx-org/gx/golang/backend"
	"github.com/gx-org/gx/golang/tests/basictest"
	"github.com/gx-org/gx/golang/tests/dtypestest"
	"github.com/gx-org/gx/golang/tests/importtest"
	"github.com/gx-org/gx/golang/tests/pkgvarstest"
	bindingstests "github.com/gx-org/gx/golang/tests"
	gxtesting "github.com/gx-org/gx/tests/testing"
)

var tests = []func(t *testing.T, dev *api.Device){
	basictest.Run,
	importtest.Run,
	pkgvarstest.Run,
	dtypestest.Run,
}

func TestGoBindings(t *testing.T) {
	bld := gxtesting.NewBuilderStaticSource(nil)
	rtm := backend.New(bld)
	dev, err := rtm.Device(0)
	if err != nil {
		t.Fatal(err)
	}
	bindingstests.Run(t, dev, tests)
}
