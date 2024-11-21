
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
package parameterstest

import (
	"testing"
	"github.com/gx-org/gx/api"
)

var tests = []struct{
  name string
  test func(*testing.T)
}{
	{name: "TestAddInt", test: TestAddInt},
	{name: "TestAddInts", test: TestAddInts},
	{name: "TestAddFloat32", test: TestAddFloat32},
	{name: "TestAddFloat32s", test: TestAddFloat32s},
	{name: "TestLen", test: TestLen},
	{name: "TestIota", test: TestIota},
	{name: "TestAddToStruct", test: TestAddToStruct},
	{name: "TestSliceSliceArg", test: TestSliceSliceArg},
	{name: "TestSliceArrayArg", test: TestSliceArrayArg},
	{name: "TestSliceArrayArgConstIndex", test: TestSliceArrayArgConstIndex},
	{name: "TestSetNotInSlice", test: TestSetNotInSlice},

}

func Run(t *testing.T, rtm *api.Runtime) {
	t.Helper()
	err := setupTest(rtm)
	if err != nil {
		t.Errorf("cannot run parameterstest tests: %+v", err)
		return
	}
	for _, test := range tests {
		t.Run("parameterstest."+test.name, test.test)
	}
}

