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
	"fmt"
	"testing"

	"github.com/gx-org/gx/build/builder/testbuild"
	"github.com/gx-org/gx/build/importers"
	"github.com/gx-org/gx/stdlib"
)

func TestCompEvalFuncCall(t *testing.T) {
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
		testbuild.Decl{
			Src: `
//gx:compeval
func add(a, b intlen) intlen {
	return a+b
}

func g[af, bf intlen]() [add(af,bf)]float32

func f() [5]float32 {
	return g[2][3]()
}
`,
		},
		testbuild.Decl{
			Src: `
//gx:compeval
func same(shape []intlen) ([]intlen, error) {
	return shape, nil
}

func f[S []intlen]([unpack(S)]float32) [unpack(same(S))]float32
`,
		},
		testbuild.Decl{
			Src: `

//gx:compeval
func CheckBroadcast(s1, s2 []intlen) ([]intlen, error) {
	return s2, nil
}

func Broadcast[Dst []intlen, Src []intlen](x [unpack(Src)]int32) [unpack(CheckBroadcast(Src, Dst))]int32


func OneHot[numClasses intlen](x [_axlen]int32) [axlen][numClasses]int32 {
	ax := []intlen{axlen, numClasses}
	xx := Broadcast[ax](([axlen][1]int32)(x))
	return xx
}
`,
		},
	)
}

func TestCompevalError(t *testing.T) {
	testbuild.RunWith(t,
		[]importers.Importer{stdlib.Importer()},
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

func f() [returnTwo()]int32 { // ERROR a compeval test error
	return [2]int32{1, 2}
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

func f() [returnTwo()]int32 { // ERROR a compeval test error
	return [2]int32{1, 2}
}
`,
		},
	)
}

func TestCompevalErrorWithVarargs(t *testing.T) {
	testbuild.RunWith(t,
		[]importers.Importer{stdlib.Importer()},
		testbuild.Decl{
			Src: `
import "fmt"

//gx:compeval
func returnAnError(xs ...int32) (intlen, error) {
	return 2, fmt.Errorf("xs length: %d", len(xs))
}

func f() [returnAnError(1, 2, 3)]int32 { // ERROR xs length: 3
	return [2]int32{1, 2}
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

func f() [returnAnError()]int32 { // ERROR xs length: 0 
	return [2]int32{1, 2}
}
`,
		},
	)
}

func TestCompevalEval(t *testing.T) {
	testbuild.Run(t,
		testbuild.CompEval{
			Src: `
//gx:compeval
func test() int32 {
	return 2
}
`,
			Wants: []string{"2"},
		},
	)
}

func TestCompevalBigFloatEqNeq(t *testing.T) {
	testbuild.Run(t,
		testbuild.CompEval{
			EvalCanonical: true,
			Src: `
//gx:compeval
func test() bool {
	return 2 == 2
}
`,
			Wants: []string{"true"},
		},
		testbuild.CompEval{
			EvalCanonical: true,
			Src: `
//gx:compeval
func test() bool {
	return 2 == 3
}
`,
			Wants: []string{"false"},
		},
		testbuild.CompEval{
			EvalCanonical: true,
			Src: `
//gx:compeval
func test() bool {
	return 2 != 2
}
`,
			Wants: []string{"false"},
		},
		testbuild.CompEval{
			EvalCanonical: true,
			Src: `
//gx:compeval
func test() bool {
	return 2 != 3
}
`,
			Wants: []string{"true"},
		},
	)
}

func TestCompevalBigFloatLSSGTR(t *testing.T) {
	testbuild.Run(t,
		testbuild.CompEval{
			EvalCanonical: true,
			Src: `
//gx:compeval
func test() bool {
	return 2 > 3
}
`,
			Wants: []string{"false"},
		},
		testbuild.CompEval{
			EvalCanonical: true,
			Src: `
//gx:compeval
func test() bool {
	return 3 > 3
}
`,
			Wants: []string{"false"},
		},
		testbuild.CompEval{
			EvalCanonical: true,
			Src: `
//gx:compeval
func test() bool {
	return 3 > 2
}
`,
			Wants: []string{"true"},
		},
		testbuild.CompEval{
			EvalCanonical: true,
			Src: `
//gx:compeval
func test() bool {
	return 2 < 3
}
`,
			Wants: []string{"true"},
		},
		testbuild.CompEval{
			EvalCanonical: true,
			Src: `
//gx:compeval
func test() bool {
	return 3 < 3
}
`,
			Wants: []string{"false"},
		},
		testbuild.CompEval{
			EvalCanonical: true,
			Src: `
//gx:compeval
func test() bool {
	return 3 < 2
}
`,
			Wants: []string{"false"},
		},
	)
}

func TestCompevalBigFloatLEQGEQ(t *testing.T) {
	testbuild.Run(t,
		testbuild.CompEval{
			EvalCanonical: true,
			Src: `
//gx:compeval
func test() bool {
	return 2 >= 3
}
`,
			Wants: []string{"false"},
		},
		testbuild.CompEval{
			EvalCanonical: true,
			Src: `
//gx:compeval
func test() bool {
	return 3 >= 3
}
`,
			Wants: []string{"true"},
		},
		testbuild.CompEval{
			EvalCanonical: true,
			Src: `
//gx:compeval
func test() bool {
	return 3 >= 2
}
`,
			Wants: []string{"true"},
		},
		testbuild.CompEval{
			EvalCanonical: true,
			Src: `
//gx:compeval
func test() bool {
	return 2 <= 3
}
`,
			Wants: []string{"true"},
		},
		testbuild.CompEval{
			EvalCanonical: true,
			Src: `
//gx:compeval
func test() bool {
	return 3 <= 3
}
`,
			Wants: []string{"true"},
		},
		testbuild.CompEval{
			EvalCanonical: true,
			Src: `
//gx:compeval
func test() bool {
	return 3 <= 2
}
`,
			Wants: []string{"false"},
		},
	)
}

func TestCompevalSliceLen(t *testing.T) {
	testbuild.Run(t,
		testbuild.CompEval{
			EvalCanonical: true,
			Src: `
//gx:compeval
func test() bool {
	a := []intlen{2, 3}
	return len(a) != len(a) 
}
`,
			Wants: []string{"false"},
		},
		testbuild.CompEval{
			EvalCanonical: true,
			Src: `
//gx:compeval
func test() bool {
	a := []intlen{2, 3}
	b := []intlen{2}
	return len(a) != len(b) 
}
`,
			Wants: []string{"true"},
		},
	)
}

func atomicAtomicSource(tok string, a, b int, want bool) string {
	out := "1"
	if want {
		out = "2"
	}
	return fmt.Sprintf(`
//gx:compeval
func cmp(a, b intlen) intlen {
	if a %s b {
		return 2
	}
	return 1
}

func f[a, b intlen](x [a]float32, y [b]float32) [cmp(a, b)]float32

func test() [%s]float32 {
	return f([%d]float32{}, [%d]float32{})
}
`, tok, out, a, b)
}

func TestCompevalAtomicAtomic(t *testing.T) {
	testbuild.Run(t,
		testbuild.Decl{
			Src: atomicAtomicSource("==", 2, 2, true),
		},
		testbuild.Decl{
			Src: atomicAtomicSource("==", 2, 3, false),
		},
		testbuild.Decl{
			Src: atomicAtomicSource("!=", 2, 2, false),
		},
		testbuild.Decl{
			Src: atomicAtomicSource("!=", 2, 3, true),
		},
		testbuild.Decl{
			Src: atomicAtomicSource("<", 3, 2, false),
		},
		testbuild.Decl{
			Src: atomicAtomicSource("<", 2, 3, true),
		},
		testbuild.Decl{
			Src: atomicAtomicSource("<", 3, 3, false),
		},
		testbuild.Decl{
			Src: atomicAtomicSource(">", 3, 2, true),
		},
		testbuild.Decl{
			Src: atomicAtomicSource(">", 2, 3, false),
		},
		testbuild.Decl{
			Src: atomicAtomicSource(">", 3, 3, false),
		},
		testbuild.Decl{
			Src: atomicAtomicSource("<=", 3, 2, false),
		},
		testbuild.Decl{
			Src: atomicAtomicSource("<=", 2, 3, true),
		},
		testbuild.Decl{
			Src: atomicAtomicSource("<=", 3, 3, true),
		},
		testbuild.Decl{
			Src: atomicAtomicSource(">=", 3, 2, true),
		},
		testbuild.Decl{
			Src: atomicAtomicSource(">=", 2, 3, false),
		},
		testbuild.Decl{
			Src: atomicAtomicSource(">=", 3, 3, true),
		},
	)
}

func TestCompevalAcrossPackages(t *testing.T) {
	testbuild.Run(t,
		testbuild.DeclarePackage{
			Src: `
package cp

const c = 6

//gx:compeval
func add(a, b intlen) intlen {
	return a+b
}

func F([_a]int32, [_b]int32) [add(a, b)+c]int32
`,
		},
		testbuild.Decl{
			Src: `
import "cp"

func f() [11]int32 {
	return cp.F([2]int32{}, [3]int32{})
}
`,
		},
	)
}
