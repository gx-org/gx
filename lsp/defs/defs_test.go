package defs_test

import (
	"fmt"
	"testing"

	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/lsp/defs"
	"github.com/gx-org/gx/lsp/position"
	"github.com/gx-org/gx/lsp/testdata"
	lsptesting "github.com/gx-org/gx/lsp/testing"
)

// irAtToken returns the IR object at targetSubstr in the unique line of file containing lineSubstr.
func irAtToken(t *testing.T, file *ir.File, lineSubstr, targetSubstr string) ir.IR {
	t.Helper()
	line, col := lsptesting.CursorAt(t, testdata.FS, file, lineSubstr, targetSubstr)
	return position.At(file, line, col)
}

// locate returns the location of def formatted as file:line:column-line:column, or an empty string
// if the location is unknown.
func locate(file *ir.File, def ir.IR) string {
	loc := defs.Locate(file, def)
	if loc == nil {
		return ""
	}
	return fmt.Sprintf("%s:%d:%d-%d:%d", loc.File.Name(), loc.Start.Line, loc.Start.Column, loc.End.Line, loc.End.Column)
}

func TestLocate(t *testing.T) {
	file := lsptesting.BuildPackage(t, testdata.FS, "main.gx").File("main.gx")
	tests := []struct {
		line, token, want string
	}{
		{`import "math"`, `"math"`, "main.gx:3:8-3:14"},
		{"const UntypedConst = 7", "UntypedConst", "main.gx:5:7-5:19"},
		{"var StaticScalar int32", "StaticScalar", "main.gx:17:5-17:17"},
		{"var localVar int32", "localVar", "main.gx:79:6-79:14"},
		{"localVar = 1", "localVar", "main.gx:79:6-79:14"},
		{"a, b := pair(p.A)", "b :=", "main.gx:81:5-81:6"},
		{"a, b = pair(b)", "a,", "main.gx:81:2-81:3"},
		{"inferred := [...]float32{1.0, 2.0}", "inferred", "main.gx:87:2-87:10"},
		{"for idx := range StaticScalar {", "idx", "main.gx:114:6-114:9"},
		{"package test", "package", ""},
		{"const UntypedConst = 7", "7", ""},
	}
	for _, tc := range tests {
		def := irAtToken(t, file, tc.line, tc.token)
		if got := locate(file, def); got != tc.want {
			t.Errorf("Locate(%T at %q in %q) = %q, want %q", def, tc.token, tc.line, got, tc.want)
		}
	}
	if got := locate(file, nil); got != "" {
		t.Errorf("Locate(file, nil) = %q, want no location", got)
	}
}

func TestLocateInOtherFile(t *testing.T) {
	pkg := lsptesting.BuildPackage(t, testdata.FS, "multifile_a.gx", "multifile_b.gx")
	fileA, fileB := pkg.File("multifile_a.gx"), pkg.File("multifile_b.gx")
	def := irAtToken(t, fileA, "func HelperA() int32 {", "HelperA")
	const want = "multifile_a.gx:7:1-9:2"
	if got := locate(fileB, def); got != want {
		t.Errorf("Locate(multifile_b.gx, %T HelperA) = %q, want %q", def, got, want)
	}
}

func TestLocateInStandardLibrary(t *testing.T) {
	file := lsptesting.BuildPackage(t, testdata.FS, "stdlib.gx").File("stdlib.gx")
	tests := []struct {
		line, token, want string
	}{
		// Function with a GX body.
		{"src := rand.NewSource(42)", "NewSource", "rand/rand.gx:24:1-32:2"},
		// Function declared in GX and implemented in Go.
		{"return math.Abs(x)", "Abs", "math/math.gx:52:1-52:49"},
		// Function without GX source code.
		{"final := control.While(", "While", ""},
	}
	for _, tc := range tests {
		got := irAtToken(t, file, tc.line, tc.token)
		ref, ok := got.(ir.WithStore)
		if !ok {
			t.Errorf("IR at %q in %q = %T, want an ir.WithStore", tc.token, tc.line, got)
			continue
		}
		def := ref.Store()
		if got := locate(file, def); got != tc.want {
			t.Errorf("Locate(stdlib.gx, %T %s) = %q, want %q", def, tc.token, got, tc.want)
		}
	}
}
