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

package ir_test

import (
	"fmt"
	"go/ast"
	"go/token"
	"strings"
	"testing"

	"github.com/gx-org/gx/api/options"
	"github.com/gx-org/gx/build/fmterr"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/build/ir/irhelper"
	"github.com/gx-org/gx/internal/interp/compeval"
	"github.com/gx-org/gx/interp/evaluator"
	"github.com/gx-org/gx/interp"
)

func names(vals []*ir.ValueRef) []string {
	ss := make([]string, len(vals))
	for i, val := range vals {
		ss[i] = val.Src.Name
	}
	return ss
}

type (
	rankFunc func() ir.ArrayRank
	typeFunc func(ir.Type, ir.Fetcher, ir.Type) (bool, error)
)

var (
	testFile = &ir.File{
		Package: &ir.Package{
			Name:  &ast.Ident{Name: "ir_test"},
			Decls: &ir.Declarations{},
		},
	}

	scalarRank         = &ir.Rank{}
	scalarRankFunc     = func() ir.ArrayRank { return scalarRank }
	rank2RankFunc      = func() ir.ArrayRank { return ir.NewRank([]int{2}) }
	rank4RankFunc      = func() ir.ArrayRank { return ir.NewRank([]int{4}) }
	rank2Plus2RankFunc = func() ir.ArrayRank {
		return newRank(&ir.BinaryExpr{
			Src: &ast.BinaryExpr{Op: token.ADD},
			X:   irhelper.IntNumberAs(2, ir.IntLenType()),
			Y:   irhelper.IntNumberAs(2, ir.IntLenType()),
			Typ: ir.IntLenType(),
		})
	}
	rank2SymbolicAxis = func() ir.ArrayRank {
		return newRank(newSymbol("a"), newSymbol("a"))
	}
	rank1SymbolicAddAxis = func() ir.ArrayRank {
		return newRank(&ir.BinaryExpr{
			Src: &ast.BinaryExpr{Op: token.ADD},
			X:   newSymbol("a"),
			Y:   newSymbol("a"),
		})
	}
	rank1SymbolicMulAxis = func() ir.ArrayRank {
		return newRank(&ir.BinaryExpr{
			Src: &ast.BinaryExpr{Op: token.MUL},
			X:   newSymbol("a"),
			Y:   newSymbol("a"),
		})
	}
	genericRankFunc = func() ir.ArrayRank {
		return &ir.RankInfer{Rnk: newRank(newSymbol("a"))}
	}
	exampleRanks = []rankFunc{
		scalarRankFunc,
		rank2RankFunc,
		rank2Plus2RankFunc,
		rank4RankFunc,
		rank2SymbolicAxis,
		rank1SymbolicAddAxis,
		rank1SymbolicMulAxis,
		genericRankFunc,
	}

	primitiveTypes = []ir.Type{
		ir.BoolType(),
		ir.Int32Type(),
		ir.Int64Type(),
		ir.Bfloat16Type(),
		ir.Float32Type(),
		ir.Float64Type(),
		ir.Uint32Type(),
		ir.Uint64Type(),
	}

	allTypes = append(
		primitiveTypes,
		ir.NumberIntType(), ir.NumberFloatType(), ir.StringType(),
	)

	symbolicAxisNames = make(map[string]bool)
)

func newRank(exprs ...ir.AssignableExpr) *ir.Rank {
	axes := make([]ir.AxisLengths, len(exprs))
	for i, expr := range exprs {
		axes[i] = &ir.AxisExpr{X: expr}
	}
	return &ir.Rank{Ax: axes}
}

func newSymbol(name string) *ir.ValueRef {
	vr := irhelper.LocalVar(name, ir.IntLenType())
	symbolicAxisNames[name] = true
	return irhelper.ValueRef(vr)
}

func declareVariable(file *ir.File, name string, opts []options.PackageOption) []options.PackageOption {
	decl := &ir.VarSpec{
		FFile: file,
		TypeV: ir.IntLenType(),
	}
	varExpr := &ir.VarExpr{
		Decl:  decl,
		VName: &ast.Ident{Name: name},
	}
	decl.Exprs = []*ir.VarExpr{varExpr}
	file.Package.Decls.Vars = append(file.Package.Decls.Vars, decl)
	opts = append(opts, compeval.NewOptionVariable(varExpr))
	return opts
}

func makeArrayTypes(types []ir.Type, ranker rankFunc) []ir.Type {
	result := make([]ir.Type, len(types))
	for i, typ := range types {
		result[i] = ir.NewArrayType(&ast.ArrayType{}, typ, ranker())
	}
	return result
}

func makeNamedType(typ ir.Type) *ir.NamedType {
	return &ir.NamedType{
		Src:        &ast.TypeSpec{Name: irhelper.Ident("named_" + typ.String())},
		File:       testFile,
		Underlying: &ir.TypeValExpr{Typ: typ},
		Methods:    []ir.PkgFunc{},
	}
}

type fetcherTesting struct {
	file *ir.File
	ctx  evaluator.Context
}

var _ ir.Fetcher = (*fetcherTesting)(nil)

func newFetcherTesting() (*fetcherTesting, error) {
	file := testFile
	var packageOptions []options.PackageOption
	for symbol := range symbolicAxisNames {
		packageOptions = declareVariable(file, symbol, packageOptions)
	}
	itp, err := interp.New(compeval.NewHostEvaluator(nil), packageOptions)
	if err != nil {
		return nil, err
	}
	ctx, err := itp.ForFile(file)
	if err != nil {
		return nil, err
	}
	return &fetcherTesting{
		file: file,
		ctx:  ctx,
	}, nil
}
func (f *fetcherTesting) File() *ir.File {
	return f.file
}

func (f *fetcherTesting) EvalExpr(expr ir.Expr) (ir.Element, error) {
	return compeval.EvalExpr(f.ctx, expr)
}

func (f *fetcherTesting) BuildExpr(src ast.Expr) (ir.Expr, bool) {
	panic("not implemented")
}

func (f *fetcherTesting) Err() *fmterr.Appender {
	return nil
}

// testTypeMatrix returns a string representation of the results of typeFunc `f` applied
// to pairs of types drawn from `a` and `b`, with elements of `a` presented as rows and `b`
// in columns.
//
// True is represented with 'X'.
func testTypeMatrix(t *testing.T, a, b []ir.Type, f typeFunc) string {
	maxLen := 0
	for _, typ := range a {
		maxLen = max(maxLen, len(typ.String()))
	}

	fetcher, err := newFetcherTesting()
	if err != nil {
		t.Fatalf("%+v", err)
	}

	var result strings.Builder
	result.WriteString("---\n")
	for _, typeA := range a {
		result.WriteString(fmt.Sprintf("%*s: ", maxLen, typeA))
		for _, typeB := range b {
			ok, err := f(typeA, fetcher, typeB)
			if err != nil {
				t.Error(err)
			}

			if ok {
				result.WriteRune('X')
			} else {
				result.WriteRune(' ')
			}
		}
		result.WriteRune('\n')
	}
	return result.String()
}

func TestTypeEquals(t *testing.T) {
	// Types are equal only to themselves:
	const want = (`---
    bool: X          
   int32:  X         
   int64:   X        
bfloat16:    X       
 float32:     X      
 float64:      X     
  uint32:       X    
  uint64:        X   
  number:         X  
  number:          X 
  string:           X
`)

	matrix := testTypeMatrix(t, allTypes, allTypes, ir.Type.Equal)
	if matrix != want {
		t.Errorf("Unexpected type Equal() matrix; got:\n%s\nwant:\n%s", matrix, want)
	}
}

func TestTypeAssignableTo(t *testing.T) {
	// Each scalar type is assignable to itself, and numbers additionally assign to concrete types.
	const want = (`---
    bool: X          
   int32:  X         
   int64:   X        
bfloat16:    X       
 float32:     X      
 float64:      X     
  uint32:       X    
  uint64:        X   
  number:  XX   XXX  
  number:    XXX   X 
  string:           X
`)

	matrix := testTypeMatrix(t, allTypes, allTypes, ir.Type.AssignableTo)
	if matrix != want {
		t.Errorf("Unexpected type AssignableTo() matrix; got:\n%s\nwant:\n%s", matrix, want)
	}
}

func TestTypeConvertibleTo(t *testing.T) {
	primitiveGenericRankArrayTypes := makeArrayTypes(primitiveTypes, genericRankFunc)

	cases := []struct {
		name   string
		a, b   []ir.Type
		matrix string
	}{
		// Every type is convertible to all other types except string.
		{"scalar-all", allTypes, allTypes, (`---
    bool: XXXXXXXXXX 
   int32: XXXXXXXXXX 
   int64: XXXXXXXXXX 
bfloat16: XXXXXXXXXX 
 float32: XXXXXXXXXX 
 float64: XXXXXXXXXX 
  uint32: XXXXXXXXXX 
  uint64: XXXXXXXXXX 
  number: XXXXXXXXXX 
  number: XXXXXXXXXX 
  string:           X
`)},
		// [...]T converts to all other [...]U for primitive type T, U.
		{"generic-array-all", primitiveGenericRankArrayTypes, primitiveGenericRankArrayTypes, (`---
    [a]bool: XXXXXXXX
   [a]int32: XXXXXXXX
   [a]int64: XXXXXXXX
[a]bfloat16: XXXXXXXX
 [a]float32: XXXXXXXX
 [a]float64: XXXXXXXX
  [a]uint32: XXXXXXXX
  [a]uint64: XXXXXXXX
`)},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			matrix := testTypeMatrix(t, testCase.a, testCase.b, ir.Type.ConvertibleTo)
			if matrix != testCase.matrix {
				t.Errorf("Unexpected type ConvertibleTo() matrix; got:\n%s\nwant:\n%s", matrix, testCase.matrix)
			}
		})
	}
}

func TestArrayAtomicEqual(t *testing.T) {
	cases := []struct {
		ranker rankFunc
		equal  bool
	}{
		// Ensure that [<nil>]T{...} == T and vice versa.
		{scalarRankFunc, true},
		// However, [2]T{...} != T.
		{rank2RankFunc, false},
		// [...]T{...} != T
		{genericRankFunc, false},
	}
	for _, testCase := range cases {
		t.Run(testCase.ranker().String(), func(t *testing.T) {
			for _, typ := range primitiveTypes {
				arrayType := ir.NewArrayType(&ast.ArrayType{}, typ, testCase.ranker())
				equal, err := arrayType.Equal(nil, typ)
				if err != nil {
					t.Error(err)
				}
				if equal != testCase.equal {
					t.Errorf("Expected %s.Equal(%s) = %v, got %v", arrayType, typ, testCase.equal, equal)
				}

				equal, err = typ.Equal(nil, arrayType)
				if err != nil {
					t.Error(err)
				}
				if equal != testCase.equal {
					t.Errorf("Expected %s.Equal(%s) = %v, got %v", typ, arrayType, testCase.equal, equal)
				}
			}
		})
	}
}

func TestNamedTypes(t *testing.T) {
	for _, typ := range allTypes {
		// Named types define a new type that is distinct from the underlying type.
		namedType := makeNamedType(typ)

		if eq, err := namedType.Equal(nil, namedType); !eq || err != nil {
			t.Errorf("Expected %s.Equal(%s) = (true, nil); got (%v, %v)", namedType, namedType, eq, err)
		}
		if eq, err := namedType.Equal(nil, typ); eq || err != nil {
			t.Errorf("Expected %s.Equal(%s) = (false, nil); got (%v, %v)", namedType, typ, eq, err)
		}
		if eq, err := typ.Equal(nil, namedType); eq || err != nil {
			t.Errorf("Expected %s.Equal(%s) = (false, nil); got (%v, %v)", typ, namedType, eq, err)
		}

		if eq, err := namedType.AssignableTo(nil, namedType); !eq || err != nil {
			t.Errorf("Expected %s.AssignableTo(%s) = (true, nil); got (%v, %v)", namedType, namedType, eq, err)
		}
		if eq, err := namedType.AssignableTo(nil, typ); eq || err != nil {
			t.Errorf("Expected %s.AssignableTo(%s) = (false, nil); got (%v, %v)", namedType, typ, eq, err)
		}
		if eq, err := typ.AssignableTo(nil, namedType); eq || err != nil {
			t.Errorf("Expected %s.AssignableTo(%s) = (false, nil); got (%v, %v)", typ, namedType, eq, err)
		}

		// Named types are convertible to themselves, plus their exact underlying type and vice-versa.
		if eq, err := namedType.ConvertibleTo(nil, namedType); !eq || err != nil {
			t.Errorf("Expected %s.ConvertibleTo(%s) = (true, nil); got (%v, %v)", namedType, namedType, eq, err)
		}
		if eq, err := namedType.ConvertibleTo(nil, typ); !eq || err != nil {
			t.Errorf("Expected %s.ConvertibleTo(%s) = (true, nil); got (%v, %v)", namedType, typ, eq, err)
		}
		if eq, err := typ.ConvertibleTo(nil, namedType); !eq || err != nil {
			t.Errorf("Expected %s.ConvertibleTo(%s) = (true, nil); got (%v, %v)", typ, namedType, eq, err)
		}
	}
}

// testRankMatrix returns a string representation of the results of typeFunc `f` applied
// to pairs of arrays with ranks drawn from `ranks`.
//
// The typeFunc comparison is applied to all primitive types, which are expected to behave
// identically.
//
// True is represented with 'X'.
func testRankMatrix(t *testing.T, ranks []rankFunc, f typeFunc) string {
	maxLen := 0
	for _, rank := range ranks {
		maxLen = max(maxLen, len(rank().String()))
	}

	fetcher, err := newFetcherTesting()
	if err != nil {
		t.Fatal(err)
	}

	var result strings.Builder
	result.WriteString("---\n")
	for _, rankA := range ranks {
		result.WriteString(fmt.Sprintf("%*s: ", maxLen, rankA()))
		for _, rankB := range ranks {
			ok := true
			for _, typ := range primitiveTypes {
				typeA := ir.NewArrayType(&ast.ArrayType{}, typ, rankA())
				typeB := ir.NewArrayType(&ast.ArrayType{}, typ, rankB())
				typeOk, err := f(typeA, fetcher, typeB)
				if err != nil {
					t.Errorf("error with types %s and %s:\n%+v", typeA.String(), typeB.String(), err)
				}
				ok = ok && typeOk
			}
			if ok {
				result.WriteRune('X')
			} else {
				result.WriteRune(' ')
			}
		}
		result.WriteRune('\n')
	}
	return result.String()
}

func TestArrayRanksEqual(t *testing.T) {
	const want = `---
      : X       
   [2]:  X      
 [2+2]:   XX    
   [4]:   XX    
[a][a]:     X   
 [a+a]:      X  
 [a*a]:       X 
   [a]:        X
`

	matrix := testRankMatrix(t, exampleRanks, ir.Type.Equal)
	if matrix != want {
		t.Errorf("Unexpected type Equal() matrix; got:\n%s\nwant:\n%s", matrix, want)
	}
}

func TestArrayRanksAssignableTo(t *testing.T) {
	const want = `---
      : X       
   [2]:  X      
 [2+2]:   XX    
   [4]:   XX    
[a][a]:     X   
 [a+a]:      X  
 [a*a]:       X 
   [a]:        X
`

	matrix := testRankMatrix(t, exampleRanks, ir.Type.AssignableTo)
	if matrix != want {
		t.Errorf("Unexpected type AssignableTo() matrix; got:\n%s\nwant:\n%s", matrix, want)
	}
}

func TestArrayRanksConvertibleTo(t *testing.T) {
	const want = `---
      : X       
   [2]:  X      
 [2+2]:   XX    
   [4]:   XX    
[a][a]:     X X 
 [a+a]:      X  
 [a*a]:     X X 
   [a]:        X
`
	matrix := testRankMatrix(t, exampleRanks, ir.Type.ConvertibleTo)
	if matrix != want {
		t.Errorf("Unexpected type ConvertibleTo() matrix; got:\n%s\nwant:\n%s", matrix, want)
	}
}

func makeTypeSet(typs ...ir.Type) *ir.TypeSet {
	return &ir.TypeSet{Typs: typs}
}

var (
	emptySet = makeTypeSet()
	boolSet  = makeTypeSet(ir.BoolType())
	sintSet  = makeTypeSet(ir.Int32Type(), ir.Int64Type())
	uintSet  = makeTypeSet(ir.Uint32Type(), ir.Uint64Type())
	intSet   = makeTypeSet(ir.Int32Type(), ir.Int64Type(), ir.Uint32Type(), ir.Uint64Type())

	allSets = []ir.Type{emptySet, boolSet, sintSet, uintSet, intSet}
)

func TestTypeSetEquals(t *testing.T) {
	const want = (`---
                                    any: X    
                     interface { bool }:  X   
              interface { int32|int64 }:   X  
            interface { uint32|uint64 }:    X 
interface { int32|int64|uint32|uint64 }:     X
`)

	matrix := testTypeMatrix(t, allSets, allSets, ir.Type.Equal)
	if matrix != want {
		t.Errorf("Unexpected type Equal() matrix; got:\n%s\nwant:\n%s", matrix, want)
	}
}

func TestTypeSetAssignableTo(t *testing.T) {
	const want = (`---
                                    any: XXXXX
                     interface { bool }: XX   
              interface { int32|int64 }: X X X
            interface { uint32|uint64 }: X  XX
interface { int32|int64|uint32|uint64 }: X   X
`)

	matrix := testTypeMatrix(t, allSets, allSets, ir.Type.AssignableTo)
	if matrix != want {
		t.Errorf("Unexpected type AssignableTo() matrix; got:\n%s\nwant:\n%s", matrix, want)
	}
}

func TestTypeSetAssignableToAllTypes(t *testing.T) {
	// Note that the horizontal axis here is all types (bool, int32, ..., string).
	const want = (`---
                                    any:            
                     interface { bool }: X          
              interface { int32|int64 }:            
            interface { uint32|uint64 }:            
interface { int32|int64|uint32|uint64 }:            
`)

	matrix := testTypeMatrix(t, allSets, allTypes, ir.Type.AssignableTo)
	if matrix != want {
		t.Errorf("Unexpected type AssignableTo() matrix; got:\n%s\nwant:\n%s", matrix, want)
	}
}

func TestTypeSetAssignableFromAllTypes(t *testing.T) {
	// Note that the horizontal axis here is all type sets.
	const want = (`---
    bool: XX   
   int32: X X X
   int64: X X X
bfloat16: X    
 float32: X    
 float64: X    
  uint32: X  XX
  uint64: X  XX
  number: X XXX
  number: X    
  string: X    
`)

	matrix := testTypeMatrix(t, allTypes, allSets, ir.Type.AssignableTo)
	if matrix != want {
		t.Errorf("Unexpected type AssignableTo() matrix; got:\n%s\nwant:\n%s", matrix, want)
	}
}

func TestTypeSetConvertibleTo(t *testing.T) {
	const want = (`---
                                    any: X    
                     interface { bool }:  X   
              interface { int32|int64 }:   X  
            interface { uint32|uint64 }:    X 
interface { int32|int64|uint32|uint64 }:     X
`)

	matrix := testTypeMatrix(t, allSets, allSets, ir.Type.ConvertibleTo)
	if matrix != want {
		t.Errorf("Unexpected type ConvertibleTo() matrix; got:\n%s\nwant:\n%s", matrix, want)
	}
}

func TestTypeSetConvertibleToAllTypes(t *testing.T) {
	// Note that the horizontal axis here is all types (bool, int32, ..., string). In short, the non-
	// empty type sets contain only types that can convert to every other type except string.
	const want = (`---
                                    any:            
                     interface { bool }: XXXXXXXXXX 
              interface { int32|int64 }: XXXXXXXXXX 
            interface { uint32|uint64 }: XXXXXXXXXX 
interface { int32|int64|uint32|uint64 }: XXXXXXXXXX 
`)

	matrix := testTypeMatrix(t, allSets, allTypes, ir.Type.ConvertibleTo)
	if matrix != want {
		t.Errorf("Unexpected type ConvertibleTo() matrix; got:\n%s\nwant:\n%s", matrix, want)
	}
}

func TestTypeSetCapabilities(t *testing.T) {
	cases := []struct {
		set *ir.TypeSet

		supportOperators bool
		isDataType       bool
		isInteger        bool
		isFloat          bool
	}{
		{emptySet, false, false, false, false},
		{boolSet, true, true, false, false},
		{intSet, true, true, true, false},
		{makeTypeSet(ir.DefaultFloatType), true, true, false, true},
	}
	for _, testCase := range cases {
		t.Run(testCase.set.String(), func(t *testing.T) {
			if got := ir.SupportOperators(testCase.set); got != testCase.supportOperators {
				t.Errorf("SupportsOperator(%s) returned %v; want %v", testCase.set, got, testCase.supportOperators)
			}
			if got := ir.IsDataType(testCase.set); got != testCase.isDataType {
				t.Errorf("IsDataType(%s) returned %v; want %v", testCase.set, got, testCase.isDataType)
			}
			if got := ir.IsInteger(testCase.set); got != testCase.isInteger {
				t.Errorf("IsInteger(%s) returned %v; want %v", testCase.set, got, testCase.isInteger)
			}
			if got := ir.IsFloat(testCase.set); got != testCase.isFloat {
				t.Errorf("IsFloat(%s) returned %v; want %v", testCase.set, got, testCase.isFloat)
			}
		})
	}
}
