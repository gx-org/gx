package hover_test

import (
	"strings"
	"testing"
	"testing/fstest"

	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/lsp/hover"
	"github.com/gx-org/gx/lsp/testdata"
	lsptesting "github.com/gx-org/gx/lsp/testing"
)

func hoverAtToken(t *testing.T, file *ir.File, lineSubstr, targetSubstr string) hover.Info {
	t.Helper()
	line, col := lsptesting.CursorAt(t, testdata.FS, file, lineSubstr, targetSubstr)
	return hover.At(file, line, col)
}

func TestAtInvalidAndBoundaryPositions(t *testing.T) {
	pkg := lsptesting.BuildPackage(t, testdata.FS, "boundary.gx")
	file := pkg.File("boundary.gx")

	if got := hover.At(nil, 1, 1); got != (hover.Info{}) {
		t.Errorf("At(nil, 1, 1) = %+v, want empty Info", got)
	}
	emptyPositions := []struct{ line, col int }{
		{0, 1}, {-1, 1}, {2, 0}, {2, -3}, {100, 1}, {2, 13}, {2, 14}, {1, 4}, {3, 1}, {4, 13},
	}
	for _, tc := range emptyPositions {
		if got := hover.At(file, tc.line, tc.col); got != (hover.Info{}) {
			t.Errorf("At(file, %d, %d) = %+v, want empty Info", tc.line, tc.col, got)
		}
	}
	for _, col := range []int{1, 12} {
		want := hover.Info{IR: pkg, SymbolInfo: "package test", Documentation: "header comment", Definition: pkg}
		if got := hover.At(file, 2, col); got != want {
			t.Errorf("At(file, 2, %d) = %+v, want %+v", col, got, want)
		}
	}
	got42 := hover.At(file, 4, 11)
	if _, ok := got42.IR.(*ir.NumberInt); !ok || got42.SymbolInfo != "" || got42.Definition != nil {
		t.Errorf("At(42) = %+v, want *ir.NumberInt with empty hover metadata", got42)
	}
}

func TestAtPackageAndImport(t *testing.T) {
	pkg := lsptesting.BuildPackage(t, testdata.FS, "main.gx")
	file := pkg.File("main.gx")

	gotPkg := hoverAtToken(t, file, "package test", "package")
	if want := (hover.Info{IR: pkg, SymbolInfo: "package test", Definition: pkg}); gotPkg != want {
		t.Errorf("package hover = %+v, want %+v", gotPkg, want)
	}

	decl := hoverAtToken(t, file, `import "math"`, `"math"`)
	if _, ok := decl.IR.(*ir.ImportDecl); !ok || decl.Definition != decl.IR || decl.SymbolInfo != `import "math"` {
		t.Errorf("import decl hover = %+v, want *ir.ImportDecl for math", decl)
	}

	ref := hoverAtToken(t, file, "inferred[1] = math.Abs(lit.A)", "math")
	if _, ok := ref.IR.(*ir.Ident); !ok || ref.Definition != decl.Definition || ref.SymbolInfo != `import "math"` {
		t.Errorf("import ref hover = %+v, want *ir.Ident resolving to math import", ref)
	}
}

func TestAtAliasedImportAndDocComment(t *testing.T) {
	file := lsptesting.BuildPackage(t, testdata.FS, "docs.gx").File("docs.gx")

	decl := hoverAtToken(t, file, `import m "math"`, `"math"`)
	ref := hoverAtToken(t, file, "return m.Abs(x)", "m.")
	for name, got := range map[string]hover.Info{"decl": decl, "ref": ref} {
		if got.SymbolInfo != `import m "math"` || got.Documentation != "Aliased math doc." || got.Definition != decl.IR {
			t.Errorf("%s hover = %+v, want aliased math import with doc and matching definition", name, got)
		}
	}
}

func TestAtConstsAndVars(t *testing.T) {
	mainFile := lsptesting.BuildPackage(t, testdata.FS, "main.gx").File("main.gx")
	docsFile := lsptesting.BuildPackage(t, testdata.FS, "docs.gx").File("docs.gx")
	tests := []struct {
		file                                     *ir.File
		line, token, wantInfo, wantDecl, wantDoc string
	}{
		{mainFile, "const UntypedConst = 7", "UntypedConst", "const UntypedConst = 7", "test/test", ""},
		{mainFile, "const TypedConst float32 = 3.5", "TypedConst", "const TypedConst float32 = 3.5", "test/test", ""},
		{mainFile, "GroupedA = 10", "GroupedA", "const GroupedA = 10", "test/test", ""},
		{mainFile, "SingleParenConst = 30", "SingleParenConst", "const SingleParenConst = 30", "test/test", ""},
		{mainFile, "var StaticScalar int32", "StaticScalar", "var StaticScalar int32", "test/test", ""},
		{mainFile, "var StaticStr string", "StaticStr", "var StaticStr string", "test/test", ""},
		{mainFile, "var localVar int32", "localVar", "var localVar int32", "", ""},
		{docsFile, "Scale float32 = 2.0", "Scale", "const Scale float32 = 2.0", "test/test", "Scale factor doc."},
		{docsFile, "Uncommented = 1", "Uncommented", "const Uncommented = 1", "test/test", ""},
		{docsFile, "var GlobalCount int32", "GlobalCount", "var GlobalCount int32", "test/test", "Global counter doc."},
		{docsFile, "var offset float32", "offset", "var offset float32", "", "Local offset doc."},
	}
	for _, tc := range tests {
		got := hoverAtToken(t, tc.file, tc.line, tc.token)
		want := hover.Info{IR: got.IR, SymbolInfo: tc.wantInfo, DeclaredBy: tc.wantDecl, Documentation: tc.wantDoc, Definition: got.IR}
		if got != want {
			t.Errorf("At(%q in %q) = %+v, want %+v", tc.token, tc.line, got, want)
		}
	}
	for _, tc := range []struct {
		file                               *ir.File
		declLine, declTok, refLine, refTok string
	}{
		{mainFile, "var StaticScalar int32", "StaticScalar", "for idx := range StaticScalar {", "StaticScalar"},
		{docsFile, "Scale float32 = 2.0", "Scale", "_ = Scale + offset", "Scale"},
		{docsFile, "var offset float32", "offset", "_ = Scale + offset", "offset"},
	} {
		decl, ref := hoverAtToken(t, tc.file, tc.declLine, tc.declTok), hoverAtToken(t, tc.file, tc.refLine, tc.refTok)
		if ref.Definition != decl.IR || ref.SymbolInfo != decl.SymbolInfo || ref.Documentation != decl.Documentation {
			t.Errorf("ref %q = %+v, want definition matching %+v", tc.refTok, ref, decl)
		}
	}
}

func TestAtLocalAssignmentsAndRangeVars(t *testing.T) {
	file := lsptesting.BuildPackage(t, testdata.FS, "main.gx").File("main.gx")
	defs := []struct {
		line, token, wantInfo, wantSnippet string
		wantIRType                         func(ir.IR) bool
	}{
		{"a, b := pair(p.A)", "a,", "var a float32", "", isType[*ir.AssignCallResult]},
		{"a, b := pair(p.A)", "b :=", "var b float32", "", isType[*ir.AssignCallResult]},
		{"inferred := [...]float32{1.0, 2.0}", "inferred", "var inferred [2]float32", "", isType[*ir.AssignExpr]},
		{"fnLit := func(v float32) float32 {", "fnLit", "var fnLit func(v float32) float32", "fnLit := func(v float32) float32 {\n\treturn v\n}", isType[*ir.AssignExpr]},
		{"if initV := fnLit(boxed.Get()); initV > 0 {", "initV :=", "var initV float32", "", isType[*ir.AssignExpr]},
		{"for idx := range StaticScalar {", "idx", "var idx int32", "", isType[*ir.LocalVarStorage]},
		{"for idx, elem := range sl {", "idx,", "var idx int", "", isType[*ir.LocalVarStorage]},
		{"for idx, elem := range sl {", "elem", "var elem int32", "", isType[*ir.LocalVarStorage]},
		{"for _, rv := range inferred {", "rv", "var rv float32", "", isType[*ir.LocalVarStorage]},
		{"for u := range unrollIndices() {", "u", "var u int", "", isType[*ir.LocalVarStorage]},
	}
	for _, tc := range defs {
		got := hoverAtToken(t, file, tc.line, tc.token)
		want := hover.Info{IR: got.IR, SymbolInfo: tc.wantInfo, Definition: got.IR, Snippet: tc.wantSnippet}
		if !tc.wantIRType(got.IR) || got != want {
			t.Errorf("At(%q in %q) = %+v (IR %T), want %+v", tc.token, tc.line, got, got.IR, want)
		}
	}
	refs := []struct {
		declLine, declTok, refLine, refTok string
		wantIRType                         func(ir.IR) bool
	}{
		{"var localVar int32", "localVar", "localVar = 1", "localVar", isType[*ir.AssignExpr]},
		{"var localVar int32", "localVar", "a += float32(localVar)", "localVar)", isType[*ir.Ident]},
		{"var localVar int32", "localVar", "localVar += int32(idx) + elem", "localVar", isType[*ir.AssignExpr]},
		{"a, b := pair(p.A)", "a,", "a, b = pair(b)", "a,", isType[*ir.AssignCallResult]},
		{"a, b := pair(p.A)", "b :=", "a, b = pair(b)", "b =", isType[*ir.AssignCallResult]},
		{"a, b := pair(p.A)", "a,", "a += float32(localVar)", "a ", isType[*ir.AssignExpr]},
		{"a, b := pair(p.A)", "a,", "a = asserted[0][0]", "a =", isType[*ir.AssignExpr]},
		{"a, b := pair(p.A)", "a,", "return a", "a", isType[*ir.Ident]},
		{"inferred := [...]float32{1.0, 2.0}", "inferred", "inferred[1] = math.Abs(lit.A)", "inferred[", isType[*ir.Ident]},
		{"inferred := [...]float32{1.0, 2.0}", "inferred", "inferred = 0", "inferred", isType[*ir.AssignExpr]},
		{"fnLit := func(v float32) float32 {", "fnLit", "if initV := fnLit(boxed.Get()); initV > 0 {", "fnLit(", isType[*ir.Ident]},
		{"for idx := range StaticScalar {", "idx", "idx = idx + 1", "idx =", isType[*ir.AssignExpr]},
		{"for idx := range StaticScalar {", "idx", "localVar += idx", "idx", isType[*ir.Ident]},
		{"for idx, elem := range sl {", "idx,", "localVar += int32(idx) + elem", "idx)", isType[*ir.Ident]},
		{"for idx, elem := range sl {", "elem", "localVar += int32(idx) + elem", "elem", isType[*ir.Ident]},
		{"for _, rv := range inferred {", "rv", "a, rv = pair(rv)", "rv =", isType[*ir.AssignCallResult]},
		{"for _, rv := range inferred {", "rv", "a += rv", "rv", isType[*ir.Ident]},
		{"for u := range unrollIndices() {", "u", "localVar += int32(u)", "u)", isType[*ir.Ident]},
	}
	for _, tc := range refs {
		decl, ref := hoverAtToken(t, file, tc.declLine, tc.declTok), hoverAtToken(t, file, tc.refLine, tc.refTok)
		if !tc.wantIRType(ref.IR) || ref.Definition != decl.IR || ref.SymbolInfo != decl.SymbolInfo || ref.Snippet != decl.Snippet {
			t.Errorf("ref %q in %q = %+v (IR %T), want definition matching %+v", tc.refTok, tc.refLine, ref, ref.IR, decl)
		}
	}
}

func isType[T any](v ir.IR) bool {
	_, ok := v.(T)
	return ok
}

func TestAtNamedTypes(t *testing.T) {
	mainFile := lsptesting.BuildPackage(t, testdata.FS, "main.gx").File("main.gx")
	docsFile := lsptesting.BuildPackage(t, testdata.FS, "docs.gx").File("docs.gx")
	multiPkg := lsptesting.BuildPackage(t, testdata.FS, "multifile_a.gx", "multifile_b.gx")
	multiA, multiB := multiPkg.File("multifile_a.gx"), multiPkg.File("multifile_b.gx")
	decls := []struct {
		file                                        *ir.File
		line, token, wantInfo, wantDoc, wantSnippet string
	}{
		{mainFile, "type Vec2 [2]float32", "Vec2", "type Vec2 [2]float32", "", ""},
		{mainFile, "type Pair struct {", "Pair", "type Pair struct", "", "type Pair struct {\n\tA, B float32\n}"},
		{mainFile, "type Box[T any] struct {", "Box", "type Box[T any] struct", "", "type Box[T any] struct {\n\tVal T\n}\n\nfunc (b Box[V]) Early[V any]() V\nfunc (b Box[U]) Get[U any]() U"},
		{mainFile, "type Stringer interface {", "Stringer", "type Stringer interface", "", "type Stringer interface {\n\tFormat(x int32) string\n}"},
		{docsFile, "type Point struct {", "Point", "type Point struct", "Point doc.", "type Point struct {\n\tX, Y float32\n}"},
		{docsFile, "Scalar = float32", "Scalar", "type Scalar = float32", "Scalar alias doc.", ""},
		{docsFile, "Floats interface {", "Floats", "type Floats interface", "Floats doc.", "type Floats interface {\n\tfloat32 | float64\n}"},
		{docsFile, "EmptyStruct struct{}", "EmptyStruct", "type EmptyStruct struct{}", "", ""},
		{docsFile, "EmptyIface  interface{}", "EmptyIface", "type EmptyIface interface{}", "", ""},
		{docsFile, "MyAny       any", "MyAny", "type MyAny any", "", ""},
		{docsFile, "type EmptyWithMethod struct{}", "EmptyWithMethod", "type EmptyWithMethod struct{}", "", "func (EmptyWithMethod) Ping() int32"},
		{docsFile, "type Meter float32", "Meter", "type Meter float32", "", "func (m Meter) Scale[T Floats](k T) out Meter"},
		{docsFile, "type TagBox[A, B any] struct {", "TagBox", "type TagBox[A, B any] struct", "", "type TagBox[A, B any] struct {\n\tFirst  A\n\tSecond B\n}\n\nfunc (b TagBox[X, Y]) With[X any, Y any, T Floats](k T) (X, T)"},
		{multiA, "type Counter struct {", "Counter", "type Counter struct", "", "type Counter struct {\n\tN int32\n}\n\nfunc (c Counter) Next() int32"},
	}
	for _, tc := range decls {
		got := hoverAtToken(t, tc.file, tc.line, tc.token)
		want := hover.Info{IR: got.IR, SymbolInfo: tc.wantInfo, DeclaredBy: "test/test", Documentation: tc.wantDoc, Definition: got.IR, Snippet: tc.wantSnippet}
		if !isType[*ir.NamedType](got.IR) || got != want {
			t.Errorf("At(%q in %q) = %+v (IR %T), want %+v", tc.token, tc.line, got, got.IR, want)
		}
	}
	pairDecl := hoverAtToken(t, mainFile, "type Pair struct {", "Pair")
	boxDecl := hoverAtToken(t, mainFile, "type Box[T any] struct {", "Box")
	counterDecl := hoverAtToken(t, multiA, "type Counter struct {", "Counter")
	refs := []struct {
		file                  *ir.File
		line, token, wantInfo string
		decl                  hover.Info
		wantIRType            func(ir.IR) bool
	}{
		{mainFile, "func Exercise(p Pair, arr [2][3]float32) float32 {", "Pair", "type Pair struct", pairDecl, isType[*ir.TypeValExpr]},
		{mainFile, "lit := Pair{B: b, A: (a + -1.25)}", "Pair{", "type Pair struct", pairDecl, isType[*ir.NamedType]},
		{mainFile, "func (b Box[V]) Early() V {", "Box", "type Box[V] struct", boxDecl, isType[*ir.TypeValExpr]},
		{mainFile, "func (b Box[U]) Get() U {", "Box", "type Box[U] struct", boxDecl, isType[*ir.TypeValExpr]},
		{mainFile, "func unboxInt(b Box[int32]) int32 {", "Box", "type Box[int32] struct", boxDecl, isType[*ir.TypeValExpr]},
		{mainFile, "boxed := Box[float32]{Val: specTwo}", "Box", "type Box[float32] struct", boxDecl, isType[*ir.NamedType]},
		{multiB, "func (c Counter) Next() int32 {", "Counter", "type Counter struct", counterDecl, isType[*ir.TypeValExpr]},
	}
	for _, tc := range refs {
		got := hoverAtToken(t, tc.file, tc.line, tc.token)
		want := hover.Info{IR: got.IR, SymbolInfo: tc.wantInfo, DeclaredBy: tc.decl.DeclaredBy, Definition: tc.decl.IR, Snippet: tc.decl.Snippet}
		if !tc.wantIRType(got.IR) || got != want {
			t.Errorf("At(%q in %q) = %+v (IR %T), want %+v", tc.token, tc.line, got, got.IR, want)
		}
	}
}

func TestAtFunctionsAndMethods(t *testing.T) {
	mainFile := lsptesting.BuildPackage(t, testdata.FS, "main.gx").File("main.gx")
	docsFile := lsptesting.BuildPackage(t, testdata.FS, "docs.gx").File("docs.gx")
	multiPkg := lsptesting.BuildPackage(t, testdata.FS, "multifile_a.gx", "multifile_b.gx")
	multiA, multiB := multiPkg.File("multifile_a.gx"), multiPkg.File("multifile_b.gx")
	stdlibFile := lsptesting.BuildPackage(t, testdata.FS, "stdlib.gx").File("stdlib.gx")

	decls := []struct {
		file                                         *ir.File
		line, token, wantInfo, wantDecl, wantSnippet string
	}{
		{mainFile, "func noop() {}", "noop", "func noop", "test/test", "func noop() {}"},
		{mainFile, "func pair(x float32) (float32, float32) {", "pair", "func pair(x float32) (float32, float32)", "test/test", "func pair(x float32) (float32, float32) {\n\treturn x, x\n}"},
		{mainFile, "func id[T any](v T) T {", "id", "func id[T any](v T) T", "test/test", "func id[T any](v T) T {\n\treturn v\n}"},
		{mainFile, "func unrollIndices() []int {", "unrollIndices", "func unrollIndices() []int", "test/test", "func unrollIndices() []int {\n\treturn []int{0, 1}\n}"},
		{mainFile, "func (b Box[U]) Get() U {", "Get", "func (b Box[U]) Get[U any]() U", "test/test.Box", "func (b Box[U]) Get() U {\n\treturn b.Val\n}"},
		{docsFile, "func (EmptyWithMethod) Ping() int32 {", "Ping", "func (EmptyWithMethod) Ping() int32", "test/test.EmptyWithMethod", "func (EmptyWithMethod) Ping() int32 {\n\treturn 1\n}"},
		{multiA, "func HelperA() int32 {", "HelperA", "func HelperA() int32", "test/test", "func HelperA() int32 {\n\treturn 1\n}"},
		{multiB, "func (c Counter) Next() int32 {", "Next", "func (c Counter) Next() int32", "test/test.Counter", "func (c Counter) Next() int32 {\n\treturn c.N + HelperA()\n}"},
	}
	for _, tc := range decls {
		got := hoverAtToken(t, tc.file, tc.line, tc.token)
		want := hover.Info{IR: got.IR, SymbolInfo: tc.wantInfo, DeclaredBy: tc.wantDecl, Definition: got.IR, Snippet: tc.wantSnippet}
		if !isType[*ir.FuncDecl](got.IR) || got != want {
			t.Errorf("At(%q in %q) = %+v (IR %T), want %+v", tc.token, tc.line, got, got.IR, want)
		}
	}

	pairDecl := hoverAtToken(t, mainFile, "func pair(x float32) (float32, float32) {", "pair")
	idDecl := hoverAtToken(t, mainFile, "func id[T any](v T) T {", "id")
	firstDecl := hoverAtToken(t, mainFile, "func firstOfTwo[T any, U any](v T, _ U) T {", "firstOfTwo")
	boxGetDecl := hoverAtToken(t, mainFile, "func (b Box[U]) Get() U {", "Get")
	helperADecl := hoverAtToken(t, multiA, "func HelperA() int32 {", "HelperA")
	refs := []struct {
		file                  *ir.File
		line, token, wantInfo string
		decl                  hover.Info
		wantIRType            func(ir.IR) bool
	}{
		{mainFile, "a, b := pair(p.A)", "pair", pairDecl.SymbolInfo, pairDecl, isType[*ir.Ident]},
		{mainFile, "specVal := id[float32](a)", "id[", idDecl.SymbolInfo, idDecl, isType[*ir.Ident]},
		{mainFile, "specVal := id[float32](a)", "[float32]", "func id[float32](v float32) float32", idDecl, isType[*ir.SpecTypeExpr]},
		{mainFile, "specTwo := firstOfTwo[float32, int32](specVal, localVar)", "](specVal", "func firstOfTwo[float32][int32](v float32, _ int32) float32", firstDecl, isType[*ir.SpecTypeExpr]},
		{mainFile, "if initV := fnLit(boxed.Get()); initV > 0 {", "Get", boxGetDecl.SymbolInfo, boxGetDecl, isType[*ir.SelectorExpr]},
		{multiB, "return c.N + HelperA()", "HelperA", helperADecl.SymbolInfo, helperADecl, isType[*ir.Ident]},
	}
	for _, tc := range refs {
		got := hoverAtToken(t, tc.file, tc.line, tc.token)
		want := hover.Info{IR: got.IR, SymbolInfo: tc.wantInfo, DeclaredBy: tc.decl.DeclaredBy, Documentation: tc.decl.Documentation, Definition: tc.decl.IR, Snippet: tc.decl.Snippet}
		if !tc.wantIRType(got.IR) || got != want {
			t.Errorf("At(%q in %q) = %+v (IR %T), want %+v", tc.token, tc.line, got, got.IR, want)
		}
	}

	gotStd := hoverAtToken(t, stdlibFile, "src := rand.NewSource(42)", "NewSource")
	if !isType[*ir.FuncDecl](gotStd.Definition) || gotStd.SymbolInfo != "func NewSource(seed int64) rand.Philox" || gotStd.DeclaredBy != "rand" || gotStd.Documentation != "NewSource returns a new source for a random generator using a seed." || !strings.HasPrefix(gotStd.Snippet, "func NewSource(seed int64) Philox {") {
		t.Errorf("At(rand.NewSource) = %+v (Def %T)", gotStd, gotStd.Definition)
	}
}

func TestAtBuiltinScalarAndSliceTypes(t *testing.T) {
	mainFile := lsptesting.BuildPackage(t, testdata.FS, "main.gx").File("main.gx")
	docsFile := lsptesting.BuildPackage(t, testdata.FS, "docs.gx").File("docs.gx")
	stdlibFile := lsptesting.BuildPackage(t, testdata.FS, "stdlib.gx").File("stdlib.gx")

	tests := []struct {
		file                  *ir.File
		line, token, wantInfo string
		wantIR                func(ir.IR) bool
	}{
		{mainFile, "var StaticScalar int32", "int32", "type int32", func(v ir.IR) bool { return v == ir.Int32Type() }},
		{mainFile, "var StaticStr string", "string", "type string", func(v ir.IR) bool { return v == ir.StringType() }},
		{mainFile, "type Vec2 [2]float32", "float32", "type float32", func(v ir.IR) bool { return v == ir.Float32Type() }},
		{mainFile, "a += float32(localVar)", "float32(", "type float32", func(v ir.IR) bool { return v == ir.Float32Type() }},
		{mainFile, "inferred := [...]float32{1.0, 2.0}", "float32{", "type float32", func(v ir.IR) bool { return v == ir.Float32Type() }},
		{mainFile, "const TypedConst float32 = 3.5", "float32", "type float32", isType[*ir.TypeValExpr]},
		{mainFile, "func unboxInt(b Box[int32]) int32 {", "int32]", "type int32", isType[*ir.TypeValExpr]},
		{mainFile, "func variadic(vals ...int32) []int32", "int32)", "type int32", isType[*ir.TypeValExpr]},
		{mainFile, "sl2d := [][]int32{{1}, {2}}", "int32{{", "type int32", isType[*ir.TypeValExpr]},
		{mainFile, "func unrollIndices() []int {", "int {", "type int", isType[*ir.TypeValExpr]},
		{docsFile, "MyAny       any", "any", "type any", isType[*ir.TypeValExpr]},
		{stdlibFile, "func(s counter) bool { return s.n < 10 },", "bool", "type bool", isType[*ir.TypeValExpr]},
		{mainFile, "specVal := id[float32](a)", "float32", "type float32", isType[*ir.Ident]},
		{mainFile, "specTwo := firstOfTwo[float32, int32](specVal, localVar)", "float32", "type float32", isType[*ir.Ident]},
		{mainFile, "specTwo := firstOfTwo[float32, int32](specVal, localVar)", "int32", "type int32", isType[*ir.Ident]},
		{mainFile, "func variadic(vals ...int32) []int32", "[]int32", "type []int32", isType[*ir.SliceType]},
		{mainFile, "func unrollIndices() []int {", "[]int", "type []int", isType[*ir.SliceType]},
		{mainFile, "sl := []int32{10, 20, 30}", "[]int32", "type []int32", isType[*ir.SliceType]},
		{mainFile, "sl2d := [][]int32{{1}, {2}}", "[][]int32", "type [][]int32", isType[*ir.SliceType]},
		{mainFile, "slArr := [][2]float32{{1, 2}}", "[][2]", "type [][2]float32", isType[*ir.SliceType]},
		{mainFile, "type Pair struct {", "struct", "", isType[*ir.StructType]},
		{mainFile, "type Stringer interface {", "interface", "", isType[*ir.Interface]},
		{docsFile, "EmptyIface  interface{}", "interface", "", isType[*ir.Interface]},
		{mainFile, "func applyFn(fn func(y int32) float32, x int32) float32 {", "func(y", "", isType[*ir.FuncType]},
		{mainFile, "func variadic(vals ...int32) []int32", "...", "", isType[*ir.VarArgsType]},
	}
	for _, tc := range tests {
		got := hoverAtToken(t, tc.file, tc.line, tc.token)
		want := hover.Info{IR: got.IR, SymbolInfo: tc.wantInfo}
		if !tc.wantIR(got.IR) || got != want {
			t.Errorf("At(%q in %q) = %+v (IR %T), want %+v", tc.token, tc.line, got, got.IR, want)
		}
	}

	varPkg := lsptesting.BuildPackage(t, testdata.FS, "main.gx")
	varFile := varPkg.File("main.gx")
	varSpec := varPkg.Decls.Vars[0]
	for _, tc := range []struct {
		typ      ir.Type
		wantInfo string
	}{
		{ir.BoolType(), "type bool"},
		{ir.Bfloat16Type(), "type bfloat16"},
		{ir.Float32Type(), "type float32"},
		{ir.Float64Type(), "type float64"},
		{ir.IntType(), "type int"},
		{ir.Int32Type(), "type int32"},
		{ir.Int64Type(), "type int64"},
		{ir.Uint32Type(), "type uint32"},
		{ir.Uint64Type(), "type uint64"},
		{ir.StringType(), "type string"},
		{ir.AnyType(), "type any"},
		{&ir.BuiltinType{Impl: "opaque"}, "type opaque"},
	} {
		varSpec.TypeV = tc.typ
		got := hoverAtToken(t, varFile, "var StaticScalar int32", "int32")
		want := hover.Info{IR: tc.typ, SymbolInfo: tc.wantInfo}
		if got != want {
			t.Errorf("At(%T) = %+v, want %+v", tc.typ, got, want)
		}
	}
}

func TestAtStructFields(t *testing.T) {
	mainFile := lsptesting.BuildPackage(t, testdata.FS, "main.gx").File("main.gx")
	docsFile := lsptesting.BuildPackage(t, testdata.FS, "docs.gx").File("docs.gx")
	multiPkg := lsptesting.BuildPackage(t, testdata.FS, "multifile_a.gx", "multifile_b.gx")
	multiA, multiB := multiPkg.File("multifile_a.gx"), multiPkg.File("multifile_b.gx")
	const docSnippet = "type DocStruct struct {\n\t// Dim doc.\n\tWidth, Height float32\n\tDepth         float32\n}"
	decls := []struct {
		file                                                  *ir.File
		line, token, wantInfo, wantDecl, wantDoc, wantSnippet string
	}{
		{mainFile, "A, B float32", "A", "A float32", "test/test.Pair", "", "type Pair struct {\n\tA, B float32\n}"},
		{mainFile, "A, B float32", "B", "B float32", "test/test.Pair", "", "type Pair struct {\n\tA, B float32\n}"},
		{mainFile, "Val T", "Val", "Val T", "test/test.Box", "", "type Box[T any] struct {\n\tVal T\n}"},
		{docsFile, "Width, Height float32", "Width", "Width float32", "test/test.DocStruct", "Dim doc.", docSnippet},
		{docsFile, "Width, Height float32", "Height", "Height float32", "test/test.DocStruct", "Dim doc.", docSnippet},
		{docsFile, "Depth         float32", "Depth", "Depth float32", "test/test.DocStruct", "", docSnippet},
		{multiA, "N int32", "N", "N int32", "test/test.Counter", "", "type Counter struct {\n\tN int32\n}"},
	}
	for _, tc := range decls {
		got := hoverAtToken(t, tc.file, tc.line, tc.token)
		want := hover.Info{IR: got.IR, SymbolInfo: tc.wantInfo, DeclaredBy: tc.wantDecl, Documentation: tc.wantDoc, Definition: got.IR, Snippet: tc.wantSnippet}
		if !isType[*ir.Field](got.IR) || got != want {
			t.Errorf("At(%q in %q) = %+v (IR %T), want %+v", tc.token, tc.line, got, got.IR, want)
		}
	}
	pairA, pairB := hoverAtToken(t, mainFile, "A, B float32", "A"), hoverAtToken(t, mainFile, "A, B float32", "B")
	boxVal, counterN := hoverAtToken(t, mainFile, "Val T", "Val"), hoverAtToken(t, multiA, "N int32", "N")
	refs := []struct {
		file                  *ir.File
		line, token, wantInfo string
		decl                  hover.Info
		wantIRType            func(ir.IR) bool
	}{
		{mainFile, "a, b := pair(p.A)", ".A", "A float32", pairA, isType[*ir.SelectorExpr]},
		{mainFile, "lit := Pair{B: b, A: (a + -1.25)}", "B:", "B float32", pairB, isType[*ir.FieldLit]},
		{mainFile, "lit := Pair{B: b, A: (a + -1.25)}", "A:", "A float32", pairA, isType[*ir.FieldLit]},
		{mainFile, "lit.A, lit.B = pair(p.A)", ".A,", "A float32", pairA, isType[*ir.StructFieldStorage]},
		{mainFile, "lit.A, lit.B = pair(p.A)", ".B", "B float32", pairB, isType[*ir.StructFieldStorage]},
		{mainFile, "lit.A = arr[0][0]", ".A", "A float32", pairA, isType[*ir.StructFieldStorage]},
		{mainFile, "boxed := Box[float32]{Val: specTwo}", "Val:", "Val float32", boxVal, isType[*ir.FieldLit]},
		{multiB, "return c.N + HelperA()", ".N", "N int32", counterN, isType[*ir.SelectorExpr]},
	}
	for _, tc := range refs {
		got := hoverAtToken(t, tc.file, tc.line, tc.token)
		want := hover.Info{IR: got.IR, SymbolInfo: tc.wantInfo, DeclaredBy: tc.decl.DeclaredBy, Definition: tc.decl.IR, Snippet: tc.decl.Snippet}
		if !tc.wantIRType(got.IR) || got != want {
			t.Errorf("At(%q in %q) = %+v (IR %T), want %+v", tc.token, tc.line, got, got.IR, want)
		}
	}
	getLine, _ := lsptesting.CursorAt(t, testdata.FS, mainFile, "func (b Box[U]) Get() U {", "Get")
	unboxLine, _ := lsptesting.CursorAt(t, testdata.FS, mainFile, "func unboxInt(b Box[int32]) int32 {", "unboxInt")
	for _, tc := range []struct {
		line     int
		wantInfo string
	}{{getLine + 1, "Val U"}, {unboxLine + 1, "Val int32"}} {
		got := hover.At(mainFile, tc.line, len("\treturn b.")+1)
		want := hover.Info{IR: got.IR, SymbolInfo: tc.wantInfo, DeclaredBy: boxVal.DeclaredBy, Definition: boxVal.IR, Snippet: boxVal.Snippet}
		if !isType[*ir.SelectorExpr](got.IR) || got != want {
			t.Errorf("At(b.Val line %d) = %+v (IR %T), want %+v", tc.line, got, got.IR, want)
		}
	}
	importedFS := fstest.MapFS{
		"imported.gx": {Data: []byte("package test\n\nimport (\n\t\"runtime:ext\"\n\t\"shape\"\n)\n\nfunc ReadDType(s shape.Shape) shape.DType {\n\treturn s.DType\n}\n")},
	}
	importedFile := lsptesting.BuildPackage(t, importedFS, "imported.gx").File("imported.gx")
	impLine, impCol := lsptesting.CursorAt(t, importedFS, importedFile, "return s.DType", ".DType")
	gotImp := hover.At(importedFile, impLine, impCol)
	const shapeSnippet = "type Shape struct {\n\tDType      DType\n\tDimensions []int\n}"
	if !isType[*ir.SelectorExpr](gotImp.IR) || !isType[*ir.Field](gotImp.Definition) || gotImp.SymbolInfo != "DType shape.DType" || gotImp.DeclaredBy != "shape.Shape" || gotImp.Documentation != "" || gotImp.Snippet != shapeSnippet {
		t.Errorf("At(.DType) = %+v (IR %T, Def %T), want shape.Shape.DType", gotImp, gotImp.IR, gotImp.Definition)
	}
	gotTypeParam := hoverAtToken(t, mainFile, "type Box[T any] struct {", "T any")
	if !isType[*ir.Field](gotTypeParam.IR) || gotTypeParam.SymbolInfo != "" || gotTypeParam.Definition != nil {
		t.Errorf("At(Box type param T) = %+v (IR %T), want *ir.Field with empty hover metadata", gotTypeParam, gotTypeParam.IR)
	}
}
