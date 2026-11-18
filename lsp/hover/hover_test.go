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
