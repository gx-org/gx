package builder_test

import (
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/build/ir/irhelper"
)

var (
	wantPackage = &ir.Package{Name: irhelper.Ident("test")}
	wantFile    = &ir.File{Package: wantPackage}
)
