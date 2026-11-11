package position_test

import (
	"embed"
	"fmt"
	"strings"
	"testing"

	"github.com/gx-org/gx/build/builder"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/build/ir/position"
	"github.com/gx-org/gx/stdlib"
)

//go:embed testdata/*.gx
var testdata embed.FS

func buildPackage(t *testing.T, filenames ...string) *ir.Package {
	t.Helper()
	bld := builder.New(stdlib.Importer())
	pkg, err := bld.BuildFiles("test", "test", testdata, filenames)
	if err != nil {
		t.Fatalf("BuildFiles failed: %v", err)
	}
	return pkg.IR()
}

func cursorAt(t *testing.T, file *ir.File, lineSubstr, targetSubstr string) (int, int) {
	t.Helper()
	src, err := testdata.ReadFile(file.Name())
	if err != nil {
		t.Fatalf("ReadFile(%q) failed: %v", file.Name(), err)
	}
	lines := strings.Split(string(src), "\n")
	matchedLine := -1
	for i, line := range lines {
		if strings.Contains(line, lineSubstr) {
			if matchedLine != -1 {
				t.Fatalf("ambiguous line substring %q matched both line %d and %d", lineSubstr, matchedLine, i+1)
			}
			matchedLine = i + 1
		}
	}
	if matchedLine == -1 {
		t.Fatalf("line substring %q not found", lineSubstr)
	}
	lineText := lines[matchedLine-1]
	idx := strings.Index(lineText, targetSubstr)
	if idx < 0 {
		t.Fatalf("target substring %q not found in line %d (%q)", targetSubstr, matchedLine, lineText)
	}
	return matchedLine, idx + 1
}

func TestAtInvalidAndBoundaryPositions(t *testing.T) {
	pkg := buildPackage(t, "testdata/boundary.gx")
	file := pkg.File("testdata/boundary.gx")
	if file == nil {
		t.Fatal("missing testdata/boundary.gx")
	}

	if got := position.At(nil, 1, 1); got != nil {
		t.Errorf("At(nil, 1, 1) = %T, want nil", got)
	}

	nilPositions := []struct {
		name      string
		line, col int
	}{
		{"zero_line", 0, 1},
		{"negative_line", -1, 1},
		{"zero_column_first_line", 1, 0},
		{"zero_column", 2, 0},
		{"negative_column", 2, -3},
		{"line_past_eof", 100, 1},
		{"newline_byte", 2, 13},
		{"column_past_eol", 2, 14},
		{"column_past_blank_line", 3, 2},
		{"top_level_comment", 1, 4},
		{"blank_line", 3, 1},
		{"after_const_literal", 4, 13},
		{"column_past_eof", 4, 100},
	}
	for _, tc := range nilPositions {
		if got := position.At(file, tc.line, tc.col); got != nil {
			t.Errorf("%s: At(file, %d, %d) = %T (%v), want nil", tc.name, tc.line, tc.col, got, got)
		}
	}

	for _, col := range []int{1, 12} {
		if got := position.At(file, 2, col); got != pkg {
			t.Errorf("At(file, 2, %d) = %T, want *ir.Package", col, got)
		}
	}
	for _, col := range []int{11, 12} {
		if _, ok := position.At(file, 4, col).(*ir.NumberInt); !ok {
			t.Errorf("At(file, 4, %d) = %T, want *ir.NumberInt", col, position.At(file, 4, col))
		}
	}
}

func TestAtDeclarationsTypesAndExpressions(t *testing.T) {
	pkg := buildPackage(t, "testdata/main.gx")
	file := pkg.File("testdata/main.gx")
	if file == nil {
		t.Fatal("missing testdata/main.gx")
	}
	if got := position.At(file, 1, 0); got != nil {
		t.Errorf("At(file, 1, 0) = %T, want nil", got)
	}

	tests := []struct {
		line  string
		token string
		check func(ir.IR) error
	}{
		{
			line:  "package test",
			token: "package",
			check: func(got ir.IR) error {
				if got != pkg {
					return fmt.Errorf("got %T, want package *ir.Package", got)
				}
				return nil
			},
		},
		{
			line:  `import "math"`,
			token: "import",
			check: wantType[*ir.ImportDecl](),
		},
		{
			line:  `import "math"`,
			token: `"math"`,
			check: func(got ir.IR) error {
				imp, ok := got.(*ir.ImportDecl)
				if !ok || imp.Path != "math" {
					return fmt.Errorf("got %T (%v), want *ir.ImportDecl(math)", got, got)
				}
				return nil
			},
		},
		{
			line:  "const UntypedConst = 7",
			token: "const",
			check: wantType[*ir.ConstSpec](),
		},
		{
			line:  "const UntypedConst = 7",
			token: "UntypedConst",
			check: func(got ir.IR) error {
				ce, ok := got.(*ir.ConstExpr)
				if !ok || ce.VName.Name != "UntypedConst" {
					return fmt.Errorf("got %T, want *ir.ConstExpr(UntypedConst)", got)
				}
				return nil
			},
		},
		{
			line:  "const UntypedConst = 7",
			token: "7",
			check: wantType[*ir.NumberInt](),
		},
		{
			line:  "const TypedConst float32 = 3.5",
			token: "float32",
			check: wantType[*ir.TypeValExpr](),
		},
		{
			line:  "const TypedConst float32 = 3.5",
			token: "3.5",
			check: wantType[*ir.NumberFloat](),
		},
		{
			line:  "const ( // multi-spec",
			token: "const",
			check: wantNil,
		},
		{
			line:  "GroupedB = 20",
			token: "= 20",
			check: wantType[*ir.ConstSpec](),
		},
		{
			line:  "const ( // single-spec",
			token: "const",
			check: wantNil,
		},
		{
			line:  "SingleParenConst = 30",
			token: "= 30",
			check: wantType[*ir.ConstSpec](),
		},
		{
			line:  "var StaticScalar int32",
			token: "var",
			check: wantType[*ir.VarSpec](),
		},
		{
			line:  "var StaticScalar int32",
			token: "StaticScalar",
			check: func(got ir.IR) error {
				ve, ok := got.(*ir.VarExpr)
				if !ok || ve.VName.Name != "StaticScalar" {
					return fmt.Errorf("got %T, want *ir.VarExpr(StaticScalar)", got)
				}
				return nil
			},
		},
		{
			line:  "var StaticScalar int32",
			token: "int32",
			check: func(got ir.IR) error {
				if got != ir.Int32Type() {
					return fmt.Errorf("got %T (%v), want Int32Type", got, got)
				}
				return nil
			},
		},
		{
			line:  "var StaticStr string",
			token: "string",
			check: func(got ir.IR) error {
				if got != ir.StringType() {
					return fmt.Errorf("got %T (%v), want StringType", got, got)
				}
				return nil
			},
		},
		{
			line:  "type Vec2 [2]float32",
			token: "[2]",
			check: wantType[*ir.Rank](),
		},
		{
			line:  "type Vec2 [2]float32",
			token: "2]",
			check: wantType[*ir.NumberInt](),
		},
		{
			line:  "type Vec2 [2]float32",
			token: "float32",
			check: func(got ir.IR) error {
				if got != ir.Float32Type() {
					return fmt.Errorf("got %T (%v), want Float32Type", got, got)
				}
				return nil
			},
		},
		{
			line:  "type Pair struct {",
			token: "Pair",
			check: func(got ir.IR) error {
				nt, ok := got.(*ir.NamedType)
				if !ok || nt.Name() != "Pair" {
					return fmt.Errorf("got %T, want *ir.NamedType(Pair)", got)
				}
				return nil
			},
		},
		{
			line:  "type Pair struct {",
			token: "struct",
			check: wantType[*ir.StructType](),
		},
		{
			line:  "A, B float32",
			token: "B",
			check: func(got ir.IR) error {
				f, ok := got.(*ir.Field)
				if !ok || f.Name.Name != "B" {
					return fmt.Errorf("got %T, want *ir.Field(B)", got)
				}
				return nil
			},
		},
		{
			line:  "type Box[T any] struct {",
			token: " any]",
			check: func(got ir.IR) error {
				grp, ok := got.(*ir.FieldGroup)
				if !ok || len(grp.Fields) != 1 || grp.Fields[0].Name.Name != "T" {
					return fmt.Errorf("got %T (%v), want *ir.FieldGroup(T)", got, got)
				}
				return nil
			},
		},
		{
			line:  "type Box[T any] struct {",
			token: "T",
			check: func(got ir.IR) error {
				f, ok := got.(*ir.Field)
				if !ok || f.Name.Name != "T" {
					return fmt.Errorf("got %T, want *ir.Field(T)", got)
				}
				return nil
			},
		},
		{
			line:  "type Stringer interface {",
			token: "interface",
			check: wantType[*ir.Interface](),
		},
		{
			line:  "Format(x int32) string",
			token: "Format",
			check: func(got ir.IR) error {
				m, ok := got.(*ir.IMethod)
				if !ok || m.Name() != "Format" {
					return fmt.Errorf("got %T, want *ir.IMethod(Format)", got)
				}
				return nil
			},
		},
		{
			line:  "Format(x int32) string",
			token: "x",
			check: func(got ir.IR) error {
				f, ok := got.(*ir.Field)
				if !ok || f.Name.Name != "x" {
					return fmt.Errorf("got %T, want *ir.Field(x)", got)
				}
				return nil
			},
		},
		{
			line:  "func (b Box[U]) Get() U {",
			token: "Get",
			check: func(got ir.IR) error {
				fn, ok := got.(*ir.FuncDecl)
				if !ok || fn.Name() != "Get" {
					return fmt.Errorf("got %T, want *ir.FuncDecl(Get)", got)
				}
				return nil
			},
		},
		{
			line:  "func (b Box[U]) Get() U {",
			token: "b ",
			check: func(got ir.IR) error {
				f, ok := got.(*ir.Field)
				if !ok || f.Name.Name != "b" {
					return fmt.Errorf("got %T, want *ir.Field(b)", got)
				}
				return nil
			},
		},
		{
			line:  "func (b Box[U]) Get() U {",
			token: "U]",
			check: func(got ir.IR) error {
				f, ok := got.(*ir.Field)
				if !ok || f.Name.Name != "U" {
					return fmt.Errorf("got %T, want *ir.Field(U)", got)
				}
				return nil
			},
		},
		{
			line:  "func unboxInt(b Box[int32]) int32 {",
			token: "int32]",
			check: func(got ir.IR) error {
				te, ok := got.(*ir.TypeValExpr)
				if !ok || te.Val() != ir.Int32Type() {
					return fmt.Errorf("got %T (%v), want *ir.TypeValExpr(int32)", got, got)
				}
				return nil
			},
		},
		{
			line:  "func pair(x float32) (float32, float32) {",
			token: "(x",
			check: wantType[*ir.FieldList](),
		},
		{
			line:  "func variadic(vals ...int32) []int32",
			token: "variadic",
			check: wantType[*ir.FuncBuiltin](),
		},
		{
			line:  "func variadic(vals ...int32) []int32",
			token: "...",
			check: wantType[*ir.VarArgsType](),
		},
		{
			line:  "func variadic(vals ...int32) []int32",
			token: "int32)",
			check: wantType[*ir.TypeValExpr](),
		},
		{
			line:  "func variadic(vals ...int32) []int32",
			token: "[]int32",
			check: wantType[*ir.SliceType](),
		},
		{
			line:  "func applyFn(fn func(y int32) float32, x int32) float32 {",
			token: "func(y",
			check: wantType[*ir.FuncType](),
		},
		{
			line:  "func applyFn(fn func(y int32) float32, x int32) float32 {",
			token: "y int32",
			check: func(got ir.IR) error {
				f, ok := got.(*ir.Field)
				if !ok || f.Name.Name != "y" {
					return fmt.Errorf("got %T, want *ir.Field(y)", got)
				}
				return nil
			},
		},
		{
			line:  "func unpackShape[S []int](x [2][S...]float32) [2][S...]float32 {",
			token: "...",
			check: wantType[*ir.Rank](),
		},
		{
			line:  "func unpackShape[S []int](x [2][S...]float32) [2][S...]float32 {",
			token: "S...",
			check: func(got ir.IR) error {
				id, ok := got.(*ir.Ident)
				if !ok || id.Src.Name != "S" {
					return fmt.Errorf("got %T, want *ir.Ident(S)", got)
				}
				return nil
			},
		},
		{
			line:  "\tnoop()",
			token: "()",
			check: wantType[*ir.FuncCallExpr](),
		},
		{
			line:  "var localVar int32",
			token: "var",
			check: wantType[*ir.DeclStmt](),
		},
		{
			line:  "var localVar int32",
			token: "localVar",
			check: func(got ir.IR) error {
				ve, ok := got.(*ir.VarExpr)
				if !ok || ve.VName.Name != "localVar" {
					return fmt.Errorf("got %T, want *ir.VarExpr(localVar)", got)
				}
				return nil
			},
		},
		{
			line:  "localVar = 1",
			token: "localVar",
			check: wantType[*ir.AssignExpr](),
		},
		{
			line:  "a, b := pair(p.A)",
			token: ":=",
			check: wantType[*ir.AssignCallStmt](),
		},
		{
			line:  "a, b := pair(p.A)",
			token: "b :=",
			check: wantType[*ir.AssignCallResult](),
		},
		{
			line:  "a, b := pair(p.A)",
			token: ".A",
			check: wantType[*ir.SelectorExpr](),
		},
		{
			line:  "a, b := pair(p.A)",
			token: "p.",
			check: func(got ir.IR) error {
				id, ok := got.(*ir.Ident)
				if !ok || id.Src.Name != "p" {
					return fmt.Errorf("got %T, want *ir.Ident(p)", got)
				}
				return nil
			},
		},
		{
			line:  "a += float32(localVar)",
			token: "+=",
			check: wantType[*ir.AssignExprStmt](),
		},
		{
			line:  "a += float32(localVar)",
			token: "float32(",
			check: func(got ir.IR) error {
				if got != ir.Float32Type() {
					return fmt.Errorf("got %T (%v), want Float32Type", got, got)
				}
				return nil
			},
		},
		{
			line:  "a += float32(localVar)",
			token: "(localVar)",
			check: wantType[*ir.CastExpr](),
		},
		{
			line:  "a += float32(localVar)",
			token: "localVar)",
			check: wantType[*ir.Ident](),
		},
		{
			line:  "lit := Pair{B: b, A: (a + -1.25)}",
			token: "Pair{",
			check: func(got ir.IR) error {
				nt, ok := got.(*ir.NamedType)
				if !ok || nt.Name() != "Pair" {
					return fmt.Errorf("got %T, want *ir.NamedType(Pair)", got)
				}
				return nil
			},
		},
		{
			line:  "lit := Pair{B: b, A: (a + -1.25)}",
			token: "{B:",
			check: wantType[*ir.StructLitExpr](),
		},
		{
			line:  "lit := Pair{B: b, A: (a + -1.25)}",
			token: "B:",
			check: func(got ir.IR) error {
				fl, ok := got.(*ir.FieldLit)
				if !ok || fl.Field.Name.Name != "B" {
					return fmt.Errorf("got %T, want *ir.FieldLit(B)", got)
				}
				return nil
			},
		},
		{
			line:  "lit := Pair{B: b, A: (a + -1.25)}",
			token: "(a",
			check: wantType[*ir.ParenExpr](),
		},
		{
			line:  "lit := Pair{B: b, A: (a + -1.25)}",
			token: "a +",
			check: func(got ir.IR) error {
				id, ok := got.(*ir.Ident)
				if !ok || id.Src.Name != "a" {
					return fmt.Errorf("got %T, want *ir.Ident(a)", got)
				}
				return nil
			},
		},
		{
			line:  "lit := Pair{B: b, A: (a + -1.25)}",
			token: "+ -1.25",
			check: wantType[*ir.BinaryExpr](),
		},
		{
			line:  "lit := Pair{B: b, A: (a + -1.25)}",
			token: "-1.25",
			check: wantType[*ir.UnaryExpr](),
		},
		{
			line:  "lit := Pair{B: b, A: (a + -1.25)}",
			token: "1.25",
			check: wantType[*ir.NumberFloat](),
		},
		{
			line:  "lit.A, lit.B = pair(p.A)",
			token: ".B",
			check: wantType[*ir.StructFieldStorage](),
		},
		{
			line:  "lit.A = arr[0][0]",
			token: "lit.",
			check: func(got ir.IR) error {
				id, ok := got.(*ir.Ident)
				if !ok || id.Src.Name != "lit" {
					return fmt.Errorf("got %T, want *ir.Ident(lit)", got)
				}
				return nil
			},
		},
		{
			line:  "lit.A = arr[0][0]",
			token: ".A",
			check: wantType[*ir.StructFieldStorage](),
		},
		{
			line:  "lit.A = arr[0][0]",
			token: "arr[",
			check: func(got ir.IR) error {
				id, ok := got.(*ir.Ident)
				if !ok || id.Src.Name != "arr" {
					return fmt.Errorf("got %T, want *ir.Ident(arr)", got)
				}
				return nil
			},
		},
		{
			line:  "lit.A = arr[0][0]",
			token: "[0]",
			check: wantType[*ir.IndexExpr](),
		},
		{
			line:  "lit.A = arr[0][0]",
			token: "0]",
			check: wantType[*ir.NumberInt](),
		},
		{
			line:  "inferred := [...]float32{1.0, 2.0}",
			token: "...",
			check: wantType[*ir.RankInfer](),
		},
		{
			line:  "inferred := [...]float32{1.0, 2.0}",
			token: "float32{",
			check: func(got ir.IR) error {
				if got != ir.Float32Type() {
					return fmt.Errorf("got %T (%v), want Float32Type", got, got)
				}
				return nil
			},
		},
		{
			line:  "inferred := [...]float32{1.0, 2.0}",
			token: "{1.0",
			check: wantType[*ir.ArrayLitExpr](),
		},
		{
			line:  "inferred := [...]float32{1.0, 2.0}",
			token: "1.0",
			check: wantType[*ir.NumberFloat](),
		},
		{
			line:  "inferred[1] = math.Abs(lit.A)",
			token: "inferred[",
			check: func(got ir.IR) error {
				id, ok := got.(*ir.Ident)
				if !ok || id.Src.Name != "inferred" {
					return fmt.Errorf("got %T, want *ir.Ident(inferred)", got)
				}
				return nil
			},
		},
		{
			line:  "inferred[1] = math.Abs(lit.A)",
			token: "[1]",
			check: wantType[*ir.IndexStorage](),
		},
		{
			line:  "inferred[1] = math.Abs(lit.A)",
			token: "1]",
			check: wantType[*ir.NumberInt](),
		},
		{
			line:  "inferred[1] = math.Abs(lit.A)",
			token: "math",
			check: func(got ir.IR) error {
				id, ok := got.(*ir.Ident)
				if !ok || id.Src.Name != "math" {
					return fmt.Errorf("got %T, want *ir.Ident(math)", got)
				}
				return nil
			},
		},
		{
			line:  "inferred[1] = math.Abs(lit.A)",
			token: "(lit.A)",
			check: wantType[*ir.FuncCallExpr](),
		},
		{
			line:  "inferred = 0",
			token: "0",
			check: wantType[*ir.NumberInt](),
		},
		{
			line:  "sl := []int32{10, 20, 30}",
			token: "{10",
			check: wantType[*ir.SliceLitExpr](),
		},
		{
			line:  "sl := []int32{10, 20, 30}",
			token: "10",
			check: wantType[*ir.NumberInt](),
		},
		{
			line:  "sl2d := [][]int32{{1}, {2}}",
			token: "[][]int32",
			check: func(got ir.IR) error {
				st, ok := got.(*ir.SliceType)
				if !ok || st.Rank != 2 {
					return fmt.Errorf("got %T (%v), want *ir.SliceType with Rank 2", got, got)
				}
				return nil
			},
		},
		{
			line:  "sl2d := [][]int32{{1}, {2}}",
			token: "int32{{",
			check: wantType[*ir.TypeValExpr](),
		},
		{
			line:  "slArr := [][2]float32{{1, 2}}",
			token: "[2]",
			check: wantType[*ir.Rank](),
		},
		{
			line:  "sub := sl[1:2]",
			token: "sl[",
			check: func(got ir.IR) error {
				id, ok := got.(*ir.Ident)
				if !ok || id.Src.Name != "sl" {
					return fmt.Errorf("got %T, want *ir.Ident(sl)", got)
				}
				return nil
			},
		},
		{
			line:  "sub := sl[1:2]",
			token: "1:",
			check: wantType[*ir.NumberInt](),
		},
		{
			line:  "sub := sl[1:2]",
			token: ":2]",
			check: wantType[*ir.SliceExpr](),
		},
		{
			line:  "sub := sl[1:2]",
			token: "2]",
			check: wantType[*ir.NumberInt](),
		},
		{
			line:  "_ = varWithPrefix(localVar, sub...)",
			token: ".)",
			check: wantType[*ir.UnpackExpr](),
		},
		{
			line:  "_ = varWithPrefix(localVar, sub...)",
			token: "sub...",
			check: func(got ir.IR) error {
				id, ok := got.(*ir.Ident)
				if !ok || id.Src.Name != "sub" {
					return fmt.Errorf("got %T, want *ir.Ident(sub)", got)
				}
				return nil
			},
		},
		{
			line:  "specVal := id[float32](a)",
			token: "id[",
			check: func(got ir.IR) error {
				id, ok := got.(*ir.Ident)
				if !ok || id.Src.Name != "id" {
					return fmt.Errorf("got %T, want *ir.Ident(id)", got)
				}
				return nil
			},
		},
		{
			line:  "specVal := id[float32](a)",
			token: "[float32]",
			check: wantType[*ir.SpecTypeExpr](),
		},
		{
			line:  "specVal := id[float32](a)",
			token: "](a)",
			check: wantType[*ir.SpecTypeExpr](),
		},
		{
			line:  "specVal := id[float32](a)",
			token: "(a)",
			check: wantType[*ir.FuncCallExpr](),
		},
		{
			line:  "specVal := id[float32](a)",
			token: "float32",
			check: func(got ir.IR) error {
				id, ok := got.(*ir.Ident)
				if !ok || id.Stor != ir.Float32Type() {
					return fmt.Errorf("got %T (%v), want *ir.Ident(float32)", got, got)
				}
				return nil
			},
		},
		{
			line:  "specTwo := firstOfTwo[float32, int32](specVal, localVar)",
			token: "int32",
			check: func(got ir.IR) error {
				id, ok := got.(*ir.Ident)
				if !ok || id.Stor != ir.Int32Type() {
					return fmt.Errorf("got %T (%v), want *ir.Ident(int32)", got, got)
				}
				return nil
			},
		},
		{
			line:  "boxed := Box[float32]{Val: specTwo}",
			token: "float32",
			check: func(got ir.IR) error {
				te, ok := got.(*ir.TypeValExpr)
				if !ok || te.Val() != ir.Float32Type() {
					return fmt.Errorf("got %T (%v), want *ir.TypeValExpr(float32)", got, got)
				}
				return nil
			},
		},
		{
			line:  "asserted := arr.([2][3]float32)",
			token: "arr.(",
			check: func(got ir.IR) error {
				id, ok := got.(*ir.Ident)
				if !ok || id.Src.Name != "arr" {
					return fmt.Errorf("got %T, want *ir.Ident(arr)", got)
				}
				return nil
			},
		},
		{
			line:  "asserted := arr.([2][3]float32)",
			token: ".([2]",
			check: wantType[*ir.TypeAssertExpr](),
		},
		{
			line:  "asserted := arr.([2][3]float32)",
			token: "[2]",
			check: wantType[*ir.Rank](),
		},
		{
			line:  "fnLit := func(v float32) float32 {",
			token: "func(",
			check: wantType[*ir.FuncLit](),
		},
		{
			line:  "fnLit := func(v float32) float32 {",
			token: "v float32",
			check: func(got ir.IR) error {
				f, ok := got.(*ir.Field)
				if !ok || f.Name.Name != "v" {
					return fmt.Errorf("got %T, want *ir.Field(v)", got)
				}
				return nil
			},
		},
		{
			line:  "return v // fnLit",
			token: "return",
			check: wantType[*ir.ReturnStmt](),
		},
		{
			line:  "if initV := fnLit(boxed.Get()); initV > 0 {",
			token: "if ",
			check: wantType[*ir.IfStmt](),
		},
		{
			line:  "if initV := fnLit(boxed.Get()); initV > 0 {",
			token: "initV :=",
			check: wantType[*ir.AssignExpr](),
		},
		{
			line:  "if initV := fnLit(boxed.Get()); initV > 0 {",
			token: "> 0",
			check: wantType[*ir.BinaryExpr](),
		},
		{
			line:  "a = asserted[0][0]",
			token: "asserted[",
			check: func(got ir.IR) error {
				id, ok := got.(*ir.Ident)
				if !ok || id.Src.Name != "asserted" {
					return fmt.Errorf("got %T, want *ir.Ident(asserted)", got)
				}
				return nil
			},
		},
		{
			line:  "} else if a < 0 {",
			token: "if a",
			check: wantType[*ir.IfStmt](),
		},
		{
			line:  "} else {",
			token: "{",
			check: wantType[*ir.BlockStmt](),
		},
		{
			line:  "for idx := range StaticScalar {",
			token: "for ",
			check: wantType[*ir.RangeStmt](),
		},
		{
			line:  "for idx := range StaticScalar {",
			token: "idx",
			check: wantType[*ir.LocalVarStorage](),
		},
		{
			line:  "for idx := range StaticScalar {",
			token: "StaticScalar",
			check: func(got ir.IR) error {
				id, ok := got.(*ir.Ident)
				if !ok || id.Src.Name != "StaticScalar" {
					return fmt.Errorf("got %T, want *ir.Ident(StaticScalar)", got)
				}
				return nil
			},
		},
		{
			line:  "localVar += int32(idx) + elem",
			token: "(idx)",
			check: wantType[*ir.CastExpr](),
		},
		{
			line:  "for idx, elem := range sl {",
			token: "elem",
			check: wantType[*ir.LocalVarStorage](),
		},
		{
			line:  "for u := range unrollIndices() {",
			token: "for ",
			check: wantType[*ir.UnrollStmt](),
		},
		{
			line:  "prod{i, k} := einsum(m1{i, j} * m2{j, k})",
			token: "einsum",
			check: wantType[*ir.EinsumExpr](),
		},
		{
			line:  "prod{i, k} := einsum(m1{i, j} * m2{j, k})",
			token: "m1",
			check: func(got ir.IR) error {
				id, ok := got.(*ir.Ident)
				if !ok || id.Src.Name != "m1" {
					return fmt.Errorf("got %T, want *ir.Ident(m1)", got)
				}
				return nil
			},
		},
		{
			line:  "prod{i, k} := einsum(m1{i, j} * m2{j, k})",
			token: "m2",
			check: func(got ir.IR) error {
				id, ok := got.(*ir.Ident)
				if !ok || id.Src.Name != "m2" {
					return fmt.Errorf("got %T, want *ir.Ident(m2)", got)
				}
				return nil
			},
		},
		{
			line:  "return a",
			token: "return",
			check: wantType[*ir.ReturnStmt](),
		},
		{
			line:  `return "hello"`,
			token: `"hello"`,
			check: wantType[*ir.StringLiteral](),
		},
	}

	for _, tc := range tests {
		t.Run(fmt.Sprintf("%s@%s", tc.line, tc.token), func(t *testing.T) {
			line, col := cursorAt(t, file, tc.line, tc.token)
			got := position.At(file, line, col)
			if err := tc.check(got); err != nil {
				t.Errorf("At(%d, %d) [%q in %q]: %v", line, col, tc.token, tc.line, err)
			}
		})
	}
}

func TestAtMultiFileMethodsAndIsolation(t *testing.T) {
	pkg := buildPackage(t, "testdata/multifile_a.gx", "testdata/multifile_b.gx")
	irA := pkg.File("testdata/multifile_a.gx")
	if irA == nil {
		t.Fatal("missing testdata/multifile_a.gx")
	}
	irB := pkg.File("testdata/multifile_b.gx")
	if irB == nil {
		t.Fatal("missing testdata/multifile_b.gx")
	}

	mLine, mCol := cursorAt(t, irB, "func (c Counter) Next() int32 {", "Next")
	gotMeth := position.At(irB, mLine, mCol)
	fn, ok := gotMeth.(*ir.FuncDecl)
	if !ok || fn.Name() != "Next" {
		t.Fatalf("At(multifile_b.gx, Next) = %T (%v), want *ir.FuncDecl(Next)", gotMeth, gotMeth)
	}

	recvLine, recvCol := cursorAt(t, irB, "func (c Counter) Next() int32 {", "Counter")
	gotRecvType := position.At(irB, recvLine, recvCol)
	if te, ok := gotRecvType.(*ir.TypeValExpr); !ok || te.Val().(*ir.NamedType).Name() != "Counter" {
		t.Fatalf("At(multifile_b.gx, Counter) = %T (%v), want *ir.TypeValExpr(Counter)", gotRecvType, gotRecvType)
	}

	callLine, callCol := cursorAt(t, irB, "return c.N + HelperA()", "HelperA")
	gotIdent := position.At(irB, callLine, callCol)
	id, ok := gotIdent.(*ir.Ident)
	if !ok || id.Src.Name != "HelperA" {
		t.Fatalf("At(multifile_b.gx, HelperA) = %T (%v), want *ir.Ident(HelperA)", gotIdent, gotIdent)
	}

	typeLine, typeCol := cursorAt(t, irA, "type Counter struct {", "Counter")
	gotType := position.At(irA, typeLine, typeCol)
	if nt, ok := gotType.(*ir.NamedType); !ok || nt.Name() != "Counter" {
		t.Fatalf("At(multifile_a.gx, Counter) = %T (%v), want *ir.NamedType(Counter)", gotType, gotType)
	}
}

func wantNil(got ir.IR) error {
	if got != nil {
		return fmt.Errorf("got %T (%v), want nil", got, got)
	}
	return nil
}

func wantType[T ir.IR]() func(ir.IR) error {
	return func(got ir.IR) error {
		if _, ok := got.(T); !ok {
			var zero T
			return fmt.Errorf("got %T (%v), want %T", got, got, zero)
		}
		return nil
	}
}
