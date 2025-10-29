// Copyright 2024 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package builder

import (
	"go/ast"
	"go/token"

	"github.com/gx-org/gx/build/builder/irb"
	"github.com/gx-org/gx/build/ir"
)

type importedFunc struct {
	file *file
	fn   *ir.FuncBuiltin
}

var _ irb.Node[*pkgResolveScope] = (*importedFunc)(nil)

func (f *importedFunc) source() ast.Node {
	return f.fn.Source()
}

func (f *importedFunc) fnSource() *ast.FuncDecl {
	return f.fn.Src
}

func (f *importedFunc) isMethod() bool {
	return f.fn.FType.ReceiverField() != nil
}

func (f *importedFunc) compEval() bool {
	return false
}

func (f *importedFunc) resolveOrder() int {
	return -1
}

func (f *importedFunc) buildAnnotations(fScope fnResolveScope, extF *irFunc) bool {
	return true
}

func (f *importedFunc) buildSignature(pkgScope *pkgResolveScope) (ir.Func, fnResolveScope, bool) {
	fScope, ok := pkgScope.newFileRScope(f.file)
	if !ok {
		return f.fn, nil, false
	}
	return f.fn, newFuncScope(fScope, f.fn.FType), true
}

func (f *importedFunc) buildBody(fnResolveScope, *irFunc) bool {
	return true
}

func (f *importedFunc) Build(ibld irBuilder) (ir.Node, bool) {
	fn := *(f.fn)
	var ok bool
	fn.FFile, ok = irCache[*ir.File](ibld, f.fn.Src, f.file)
	return &fn, ok
}

func pNodeFromFunc(pkgScope *pkgProcScope, file *file, fn ir.PkgFunc) (*processNodeT[function], bool) {
	fnT, ok := fn.(*ir.FuncBuiltin)
	if !ok {
		return nil, pkgScope.Err().AppendInternalf(fn.Source(), "cannot import function %T: not supported", fn)
	}
	return newProcessNode[function](token.FUNC, fnT.Src.Name, &importedFunc{
		file: file,
		fn:   fnT,
	}), true
}

func importNamedTypes(pkgScope *pkgProcScope, bFile *file, types []*ir.NamedType) bool {
	for _, typ := range types {
		if _, exist := pkgScope.decls().declarations.Load(typ.Name()); exist {
			for _, method := range typ.Methods {
				if !importFunc(pkgScope, bFile, method) {
					return false
				}
			}
			continue
		}
		namedTyp := *typ
		pNode := newProcessNode(token.TYPE, namedTyp.Src.Name, &namedTyp)
		if !pkgScope.decls().declarePackageName(pNode) {
			return false
		}
	}
	return true
}

func importFunc(pkgScope *pkgProcScope, bFile *file, fn ir.PkgFunc) bool {
	fNode, ok := pNodeFromFunc(pkgScope, bFile, fn)
	if !ok {
		return false
	}
	if !pkgScope.decls().declarePackageName(fNode) {
		return false
	}
	return true
}

func importFuncs(pkgScope *pkgProcScope, bFile *file, funcs []ir.PkgFunc) bool {
	for _, fn := range funcs {
		if !importFunc(pkgScope, bFile, fn) {
			return false
		}
	}
	return true
}

type importedConstExpr struct {
	bFile *file
	expr  *ir.ConstExpr
}

var _ iConstExpr = (*importedConstExpr)(nil)

func (b *importedConstExpr) buildDeclaration(ibld irBuilder) (*ir.ConstExpr, bool) {
	extSpec := *b.expr.Decl
	extSpec.Exprs = []*ir.ConstExpr{{
		Decl:  &extSpec,
		VName: b.expr.VName,
		Val:   b.expr.Val,
	}}
	var ok bool
	extSpec.FFile, ok = irBuild[*ir.File](ibld, b.bFile)
	ibld.Register(constDeclarator(&extSpec))
	return extSpec.Exprs[0], ok
}

func (b *importedConstExpr) buildExpression(ibld irBuilder, ext *ir.ConstExpr) bool {
	return true
}

func importConstDecls(pkgScope *pkgProcScope, file *file, cstDecls []*ir.ConstSpec) bool {
	for _, cstDecl := range cstDecls {
		if len(cstDecl.Exprs) != 1 {
			return pkgScope.Err().AppendInternalf(cstDecl.Src, "constant specification got %d expressions but want 1", len(cstDecl.Exprs))
		}
		cstExpr := cstDecl.Exprs[0]
		pNode := newProcessNode[iConstExpr](token.CONST, cstExpr.VName, &importedConstExpr{
			bFile: file,
			expr:  cstExpr,
		})
		if !pkgScope.decls().declarePackageName(pNode) {
			return false
		}
	}
	return true
}
