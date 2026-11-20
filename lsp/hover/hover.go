// Package hover computes hover information for GX source files.
package hover

import (
	"fmt"
	"path"
	"slices"
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
	case *ir.ConstExpr:
		populateConst(info, file, n)
	case *ir.VarExpr:
		populateVar(info, file, n)
	case *ir.Ident:
		populate(info, file, n.Stor)
	}
	// TODO(b/586481357): Populate local assignment, range storage, and builtin type nodes.
}

func populateImport(info *Info, file *ir.File, imp *ir.ImportDecl) {
	info.Definition = imp
	if imp.Src.Name != nil && imp.Src.Name.Name != path.Base(imp.Path) {
		info.SymbolInfo = fmt.Sprintf("import %s %q", imp.Src.Name.Name, imp.Path)
	} else {
		info.SymbolInfo = fmt.Sprintf("import %q", imp.Path)
	}
	info.Documentation = specDoc(file, imp.Src, imp.Src.Doc)
}

func populateConst(info *Info, file *ir.File, ce *ir.ConstExpr) {
	info.Definition = ce
	typePart := ""
	if ce.Decl.Type != nil {
		typePart = " " + ce.Decl.Type.SourceString(file)
	}
	info.SymbolInfo = fmt.Sprintf("const %s%s = %s", ce.VName.Name, typePart, ce.Val.SourceString(file))
	info.DeclaredBy = ce.File().Package.Path()
	info.Documentation = specDoc(ce.File(), ce.Decl.Src, ce.Decl.Src.Doc)
}

func populateVar(info *Info, file *ir.File, ve *ir.VarExpr) {
	info.Definition = ve
	info.SymbolInfo = fmt.Sprintf("var %s %s", ve.VName.Name, ve.Type().ReferString(file))
	if slices.Contains(ve.Decl.FFile.Package.Decls.Vars, ve.Decl) {
		info.DeclaredBy = ve.Decl.FFile.Package.Path()
	}
	info.Documentation = specDoc(ve.Decl.FFile, ve.Decl.Src, ve.Decl.Src.Doc)
}

func specDoc(file *ir.File, spec ast.Spec, doc *ast.CommentGroup) string {
	if text := strings.TrimSpace(doc.Text()); text != "" {
		return text
	}
	for n := range ast.Preorder(file.Src) {
		gd, ok := n.(*ast.GenDecl)
		if ok && len(gd.Specs) == 1 && gd.Specs[0] == spec {
			return strings.TrimSpace(gd.Doc.Text())
		}
	}
	return ""
}
