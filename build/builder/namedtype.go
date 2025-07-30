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

	"github.com/gx-org/gx/base/ordered"
	"github.com/gx-org/gx/build/builder/irb"
	"github.com/gx-org/gx/build/fmterr"
	"github.com/gx-org/gx/build/ir"
)

// namedType is a node representing a named type declaration in GX source code.
type namedType struct {
	src        *ast.TypeSpec
	file       *file
	underlying typeExprNode
}

func processTypeDecl(scope procScope, decl *ast.GenDecl) bool {
	ok := true
	for _, spec := range decl.Specs {
		ok = processType(scope, spec.(*ast.TypeSpec)) && ok
	}
	return ok
}

func processType(pscope procScope, src *ast.TypeSpec) bool {
	n := &namedType{
		src:  src,
		file: pscope.file(),
	}
	var ok bool
	n.underlying, ok = processTypeExpr(pscope, src.Type)
	if !ok {
		return false
	}
	if src.TypeParams.NumFields() > 0 {
		pscope.Err().Appendf(src, "type may not have type parameters")
		ok = false
	}
	pNode := newProcessNode(token.TYPE, src.Name, n)
	return pscope.decls().declarePackageName(pNode) && ok
}

func (n *namedType) source() ast.Node {
	return n.src
}

type irNamedType struct {
	bType *namedType
	ext   *ir.NamedType
}

func (n *namedType) build(ibld irBuilder) (*irNamedType, bool) {
	ext := &ir.NamedType{
		Src: n.src,
	}
	var ok bool
	ext.File, ok = irBuild[*ir.File](ibld, n.file)
	nType := &irNamedType{
		bType: n,
		ext:   ext,
	}
	ibld.Register(namedTypeDeclarator(ibld.Scope(), ext))
	return nType, ok
}

func (n *namedType) buildUnderlying(pkgScope *pkgResolveScope, nType *ir.NamedType) bool {
	rscope, scopeOk := pkgScope.newFileScope(n.file)
	if !scopeOk {
		return false
	}
	var underOk bool
	nType.Underlying, underOk = n.underlying.buildTypeExpr(rscope)
	return underOk
}

func assignMethod(scope *fileResolveScope, ext *ir.NamedType, fn *irFunc) bool {
	underlying := ir.Underlying(ext)
	// Check if a field with the same name has already been defined.
	structType, ok := underlying.(*ir.StructType)
	if ok {
		if defined := structType.Fields.FindField(fn.irFunc.Name()); defined != nil {
			return scope.Err().Appendf(fn.irFunc.Source(), "field and method with the same name %s", fn.irFunc.Name())
		}
	}
	// Check if a method has already been defined.
	methods, exist := scope.methods.Load(ext)
	if !exist {
		methods = ordered.NewMap[string, *irFunc]()
		scope.methods.Store(ext, methods)
	}
	if prev, hasPrev := methods.Load(ext.Name()); hasPrev {
		return scope.Err().Appendf(fn.irFunc.Source(), "method %s.%s already declared at %s", ext.Name(), fn.irFunc.Name(), funcPos(scope, prev.bFunc))
	}
	methods.Store(fn.irFunc.Name(), fn)
	ext.Methods = updateMethods(scope.pkgResolveScope, ext)
	return true
}

func updateMethods(scope *pkgResolveScope, ext *ir.NamedType) []ir.PkgFunc {
	methods, exist := scope.methods.Load(ext)
	if !exist {
		return nil
	}
	var irMethods []ir.PkgFunc
	for method := range methods.Values() {
		irMethods = append(irMethods, method.irFunc)
	}
	return irMethods
}

func funcPos(scope *fileResolveScope, fn function) string {
	fnPos, ok := fn.(interface{ source() ast.Node })
	if !ok {
		return "as a builtin"
	}
	return fmterr.PosString(scope.Err().FSet().FSet, fnPos.source().Pos())
}

func (n *namedType) String() string {
	return n.src.Name.Name
}

func namedTypeDeclarator(scope *pkgResolveScope, ext *ir.NamedType) irb.Declarator {
	return func(decls *ir.Declarations) {
		decls.Types = append(decls.Types, ext)
	}
}
