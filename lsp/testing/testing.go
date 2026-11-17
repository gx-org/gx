// Package testing provides shared test helpers for GX LSP tests.
package testing

import (
	"io/fs"
	"strings"
	"testing"

	"github.com/gx-org/gx/build/builder"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/stdlib"
)

// BuildPackage builds a GX test package from filenames in fsys and returns its IR.
func BuildPackage(t *testing.T, fsys fs.FS, filenames ...string) *ir.Package {
	t.Helper()
	pkg, err := builder.New(stdlib.Importer()).BuildFiles("test", "test", fsys, filenames)
	if err != nil {
		t.Fatalf("BuildFiles failed: %v", err)
	}
	return pkg.IR()
}

// CursorAt reads file.Name() from fsys, locates the unique 1-based line
// containing lineSubstr, and returns the 1-based line and column of
// targetSubstr.
func CursorAt(t *testing.T, fsys fs.FS, file *ir.File, lineSubstr, targetSubstr string) (int, int) {
	t.Helper()
	src, err := fs.ReadFile(fsys, file.Name())
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
