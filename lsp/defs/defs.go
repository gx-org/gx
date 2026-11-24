// Package defs locates the definitions of GX symbols in source code.
package defs

import (
	"go/token"

	"github.com/gx-org/gx/build/ir"
)

// Location is the location of a definition in GX source code.
type Location struct {
	// File defining the symbol.
	File *ir.File
	// Start and End delimit the definition in File, End being excluded. The positions are not
	// adjusted by //line directives.
	Start, End token.Position
}

// Locate returns the location where a symbol referred to from a file is defined, and false if the
// location is unknown.
func Locate(file *ir.File, def ir.IR) *Location {
	decl, ok := def.(ir.Node)
	if !ok {
		return nil
	}
	declFile := declaringFile(file, def)
	if declFile == nil || declFile.Src == nil {
		// Builtins defined in Go are declared in files without GX source code.
		return nil
	}
	fset := declFile.FileSet()
	start := fset.PositionFor(decl.Node().Pos(), false)
	if !start.IsValid() {
		return nil
	}
	end := fset.PositionFor(decl.Node().End(), false)
	return &Location{File: declFile, Start: start, End: end}
}

// declaringFile returns the file declaring a symbol, or nil if it is unknown.
//
// Each GX package has its own file set, so the positions of a symbol are only meaningful within
// the file set of the package declaring it.
func declaringFile(file *ir.File, def ir.IR) *ir.File {
	switch d := def.(type) {
	case ir.StorageWithFile:
		return d.File()
	case *ir.VarExpr:
		return d.Decl.FFile
	case *ir.AssignExpr:
		return declaringFile(file, d.Storage)
	case *ir.AssignCallResult:
		return declaringFile(file, d.Storage)
	case *ir.ImportDecl, *ir.LocalVarStorage:
		return file
	}
	return nil
}
