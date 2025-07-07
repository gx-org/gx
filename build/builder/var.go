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
	"fmt"
	"go/ast"
	"go/token"

	"github.com/gx-org/gx/build/builder/irb"
	"github.com/gx-org/gx/build/ir"
)

type varSpec struct {
	src   *ast.ValueSpec
	bFile *file
	exprs []*varExpr
	typ   typeExprNode
}

var _ irb.Node[*pkgResolveScope] = (*varSpec)(nil)

func processVarDecl(pscope procScope, decl *ast.GenDecl) bool {
	ok := true
	for _, spec := range decl.Specs {
		ok = processVarSpec(pscope.axisLengthScope(), spec.(*ast.ValueSpec)) && ok
	}
	return ok
}

func processVarSpec(pscope procScope, src *ast.ValueSpec) bool {
	if len(src.Values) > 0 {
		pscope.err().Appendf(src, "cannot assign a value to a static variable")
	}
	spec := &varSpec{
		src:   src,
		bFile: pscope.file(),
	}
	var typeOk bool
	if src.Type != nil {
		spec.typ, typeOk = processTypeExpr(pscope, src.Type)
	} else {
		typeOk = pscope.err().Appendf(src, "static variable has no type")
	}
	exprsOk := true
	spec.exprs = make([]*varExpr, len(src.Names))
	for i, name := range src.Names {
		var exprOk bool
		spec.exprs[i], exprOk = spec.processVarExpr(pscope, name, nil)
		exprsOk = exprsOk && exprOk
	}
	return typeOk && exprsOk && pscope.decls().registerStaticVar(spec)
}

func (spec *varSpec) Build(ibld irBuilder) (ir.Node, bool) {
	ext := &ir.VarSpec{Src: spec.src}
	ibld.Register(varDeclarator(ext))
	var ok bool
	ext.FFile, ok = irBuild[*ir.File](ibld, spec.bFile)
	if !ok {
		return ext, false
	}
	if spec.typ == nil {
		return ext, false
	}
	fScope, ok := ibld.Scope().newFileScope(spec.bFile)
	if !ok {
		return nil, false
	}
	typeRef, ok := spec.typ.buildTypeExpr(fScope)
	if !ok {
		return ext, false
	}
	ext.TypeV = typeRef.Typ
	return ext, true
}

type varExpr struct {
	spec *varSpec
	name *ast.Ident
}

func (spec *varSpec) processVarExpr(pscope procScope, name *ast.Ident, value ast.Expr) (*varExpr, bool) {
	return &varExpr{spec: spec, name: name}, true
}

func (vr *varExpr) build(ibld irBuilder) (*ir.VarExpr, bool) {
	ext := &ir.VarExpr{VName: vr.name}
	var ok bool
	ext.Decl, ok = irBuild[*ir.VarSpec](ibld, vr.spec)
	ext.Decl.Exprs = append(ext.Decl.Exprs, ext)
	return ext, ok
}

func (vr *varExpr) pNode() processNode {
	return newProcessNode[*varExpr](token.VAR, vr.name, vr)
}

func varDeclarator(spec *ir.VarSpec) irb.Declarator {
	return func(decls *ir.Declarations) {
		decls.Vars = append(decls.Vars, spec)
	}
}

func (vr *varExpr) String() string {
	return fmt.Sprintf("var %s", vr.name.Name)
}
