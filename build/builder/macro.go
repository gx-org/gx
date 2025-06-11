package builder

import (
	"go/ast"

	"github.com/gx-org/gx/build/ir"
)

type funcMacro struct {
	*funcDecl
}

func (bFile *file) processIRMacroFunc(scope procScope, src *ast.FuncDecl, comment *ast.Comment) bool {
	fDecl, fDeclOk := newFuncDecl(scope, src, false)
	fn := &funcMacro{
		funcDecl: fDecl,
	}
	returnOk := fn.funcDecl.checkReturnValue(scope)
	declareOk := scope.decls().registerFunc(fn)
	return fDeclOk && returnOk && declareOk
}

func (f *funcMacro) buildSignature(pkgScope *pkgResolveScope) (ir.Func, iFuncResolveScope, bool) {
	ext := &ir.Macro{
		Src: f.src,
	}
	fileScope, scopeOk := pkgScope.newFileScope(f.bFile, nil)
	if !scopeOk {
		return ext, nil, false
	}
	var ok bool
	var fScope *funcResolveScope
	ext.FType, fScope, ok = f.fType.buildFuncType(fileScope)
	return ext, fScope, ok
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

func (f *funcMacro) resolveOrder() int {
	// Macro needs to be resolved first before any other functions.
	return -1
}
