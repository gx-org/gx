package hover_test

import (
	"testing"

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
