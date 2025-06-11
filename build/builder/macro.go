package builder

import (
	"go/ast"

	"github.com/gx-org/gx/build/ir"
)

type funcMacro struct {
	src   *ast.FuncDecl
	bFile *file
}

func (bFile *file) processIRMacroFunc(pscope procScope, src *ast.FuncDecl, comment *ast.Comment) bool {
	f := &funcMacro{src: src, bFile: pscope.file()}
	argsOk := checkEmptyParamsResults(pscope, src, "irmacro")
	recvOk := src.Recv.NumFields() == 0
	if !recvOk {
		pscope.err().Appendf(src, "irmacro function has a receiver")
	}
	declareOk := pscope.decls().registerFunc(f)
	return argsOk && recvOk && declareOk
}

func (f *funcMacro) buildSignature(pkgScope *pkgResolveScope) (ir.Func, iFuncResolveScope, bool) {
	fScope, ok := pkgScope.newFileScope(f.bFile, nil)
	return &ir.FuncMeta{
		Src: f.src,
	}, newFuncScope(fScope, &ir.FuncType{}), ok
}

func (f *funcMacro) source() ast.Node {
	return f.src
}

func (f *funcMacro) name() *ast.Ident {
	return f.src.Name
}

func (f *funcMacro) buildBody(fScope iFuncResolveScope, extF ir.Func) bool {
	return true
}

func (f *funcMacro) receiver() *fieldList {
	return nil
}

func (f *funcMacro) compEval() bool {
	return true
}
