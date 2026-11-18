// Package hover computes hover information for GX source files.
package hover

import (
	"fmt"
	"path"
	"strings"

	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/lsp/position"
	"github.com/gx-org/gx/src/gx/ast"
)

// Info holds hover information for an IR node at a source position.
type Info struct {
	IR            ir.IR
	SymbolInfo    string
	DeclaredBy    string
	Documentation string
	Definition    ir.IR
	Snippet       string
}

// At computes hover information at the given 1-based line and column in file.
func At(file *ir.File, line, column int) Info {
	node := position.At(file, line, column)
	if node == nil {
		return Info{}
	}
	info := Info{IR: node}
	populate(&info, file, node)
	return info
}

func populate(info *Info, file *ir.File, node ir.IR) {
	switch n := node.(type) {
	case *ir.Package:
		info.Definition = n
		info.SymbolInfo = "package " + n.Name.Name
		info.Documentation = strings.TrimSpace(file.Src.Doc.Text())

	case *ir.ImportDecl:
		populateImport(info, file, n)

	case *ir.Ident:
		populate(info, file, n.Stor)
	}
	// TODO(b/586481357): Populate const, var, assignment, and builtin type nodes.
}

func populateImport(info *Info, file *ir.File, imp *ir.ImportDecl) {
	info.Definition = imp
	if imp.Src.Name != nil && imp.Src.Name.Name != path.Base(imp.Path) {
		info.SymbolInfo = fmt.Sprintf("import %s %q", imp.Src.Name.Name, imp.Path)
	} else {
		info.SymbolInfo = fmt.Sprintf("import %q", imp.Path)
	}
	info.Documentation = importDoc(file, imp.Src)
}

func importDoc(file *ir.File, spec *ast.ImportSpec) string {
	if text := strings.TrimSpace(spec.Doc.Text()); text != "" {
		return text
	}
	for _, decl := range file.Src.Decls {
		gd, ok := decl.(*ast.GenDecl)
		if ok && len(gd.Specs) == 1 && gd.Specs[0] == spec {
			return strings.TrimSpace(gd.Doc.Text())
		}
	}
	return ""
}
