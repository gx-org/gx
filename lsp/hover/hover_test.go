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
