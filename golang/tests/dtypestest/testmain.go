
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
package dtypestest

import (
	"testing"
	"github.com/gx-org/gx/api"
)

var tests = []struct{
  name string
  test func(*testing.T)
}{
	{name: "TestBool", test: TestBool},
	{name: "TestFloat32", test: TestFloat32},
	{name: "TestFloat64", test: TestFloat64},
	{name: "TestInt32", test: TestInt32},
	{name: "TestInt64", test: TestInt64},
	{name: "TestUint32", test: TestUint32},
	{name: "TestUint64", test: TestUint64},
	{name: "TestBoolArray", test: TestBoolArray},
	{name: "TestFloat32Array", test: TestFloat32Array},
	{name: "TestFloat64Array", test: TestFloat64Array},
	{name: "TestInt32Array", test: TestInt32Array},
	{name: "TestInt64Array", test: TestInt64Array},
	{name: "TestUint32Array", test: TestUint32Array},
	{name: "TestUint64Array", test: TestUint64Array},

}

func Run(t *testing.T, dev *api.Device) {
	t.Helper()
	err := setupTest(dev)
	if err != nil {
		t.Errorf("cannot run dtypestest tests: %+v", err)
		return
	}
	for _, test := range tests {
		t.Run("dtypestest."+test.name, test.test)
	}
}

