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

package builder_test

import (
	"testing"

	"github.com/gx-org/gx/build/builder/testbuild"
)

func TestCompeval(t *testing.T) {
	testbuild.Run(t,
		testbuild.Decl{
			Src: `
//gx:compeval
func returnTwo() intlen {
	return 2
}

func f() [returnTwo()]int32 {
	return [2]int32{1, 2}
	// Want:
	// [2]int32{1, 2}
}
`,
		},
		testbuild.Decl{
			Src: `
//gx:compeval
func returnTwo() (intlen, error) {
	return 2, nil
}

func f() [returnTwo()]int32 {
	return [2]int32{1, 2}
	// Want:
	// [2]int32{1, 2}
}
`,
		},
		testbuild.Decl{
			Src: `
//gx:compeval
func str() string

//gx:compeval
func f() string {
	// Check that str() is not executed when f is compiled.
	return str()
}
`,
		},
		testbuild.Decl{
			Src: `
//gx:compeval
func str() string

//gx:compeval
func g(string) string

//gx:compeval
func f() string {
	// Check that str() is not executed when f is compiled.
	return g(str())
}
`,
		},
	)
}

func TestCompevalError(t *testing.T) {
	testbuild.Run(t,
		testbuild.Decl{
			Src: `
import "fmt"

func returnTwo() intlen {
	return 2
}

func f() [returnTwo()]int32 {  // ERROR expect a compeval function, function returnTwo is not
	return [2]int32{1, 2}
}
`,
		},
		testbuild.Decl{
			Src: `
import "errors"

//gx:compeval
func returnTwo() (intlen, error) {
	return 2, errors.New("a compeval test error")
}

func f() [returnTwo()]int32 { 
	return [2]int32{1, 2} // ERROR a compeval test error
}
`,
		},
		testbuild.Decl{
			Src: `
import "fmt"

//gx:compeval
func returnTwo() (intlen, error) {
	return 2, fmt.Errorf("a compeval test error")
}

func f() [returnTwo()]int32 { 
	return [2]int32{1, 2} // ERROR a compeval test error
}
`,
		},
	)
}

func TestCompevalErrorWithVarargs(t *testing.T) {
	testbuild.Run(t,
		testbuild.Decl{
			Src: `
import "fmt"

//gx:compeval
func returnAnError(xs ...int32) (intlen, error) {
	return 2, fmt.Errorf("xs length: %d", len(xs))
}

func f() [returnAnError(1, 2, 3)]int32 { 
	return [2]int32{1, 2} // ERROR xs length: 3
}
`,
		},
		testbuild.Decl{
			Src: `
import "fmt"

//gx:compeval
func returnAnError(xs ...int32) (intlen, error) {
	return 2, fmt.Errorf("xs length: %d", len(xs))
}

func f() [returnAnError()]int32 { 
	return [2]int32{1, 2} // ERROR xs length: 0
}
`,
		},
	)
}
