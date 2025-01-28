
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

// Package main generates a main function to run all the GX tests of a file.
// Automatically generated from google3/third_party/gxlang/gx/golang/tools/testsmain.go.
//
// DO NOT EDIT
package randtest

import (
	"testing"
	"github.com/gx-org/gx/api"
)

var tests = []struct{
  name string
  test func(*testing.T)
}{
	{name: "TestSample", test: TestSample},
	{name: "TestSampleBool", test: TestSampleBool},
	{name: "TestDeviceValueAsSeed", test: TestDeviceValueAsSeed},

}

func Run(t *testing.T, dev *api.Device) {
	t.Helper()
	err := setupTest(dev)
	if err != nil {
		t.Errorf("cannot run randtest tests: %+v", err)
		return
	}
	for _, test := range tests {
		t.Run("randtest."+test.name, test.test)
	}
}

