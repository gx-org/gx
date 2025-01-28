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

	"github.com/gx-org/gx/build/ir"
)

type varDecl struct {
	ext ir.VarDecl

	declaredType typeNode
	typ          typeNode
}

var _ irBuilder = (*varDecl)(nil)

func processVarDecl(scope *scopeFile, decl *ast.GenDecl) bool {
	ok := true
	for _, spec := range decl.Specs {
		ok = processVar(scope, spec.(*ast.ValueSpec)) && ok
	}
	return ok
}

func processVar(scope *scopeFile, spec *ast.ValueSpec) bool {
	if len(spec.Values) > 0 {
		scope.err().Appendf(spec, "cannot assign a value to a static variable")
	}
	var typ typeNode
	tpOk := true
	if spec.Type != nil {
		typ, tpOk = processTypeExpr(scope, spec.Type)
	}
	ok := true
	for _, name := range spec.Names {
		decl := &varDecl{
			ext: ir.VarDecl{
				FFile: &scope.file().repr,
				Src:   spec,
				VName: name,
			},
			declaredType: typ,
		}
		fScope := scope.fileScope()
		ok = fScope.file().declareStaticVar(fScope, name, decl) && ok
	}
	return tpOk && ok
}

func (vr *varDecl) source() ast.Node {
	return vr.ext.Src
}

func (vr *varDecl) resolveType(scope scoper) (typeNode, bool) {
	if vr.typ != nil {
		return typeNodeOk(vr.typ)
	}
	if vr.declaredType == nil {
		vr.typ = invalid
		return vr.typ, false
	}
	var ok bool
	vr.typ, ok = resolveType(scope, vr, vr.declaredType)
	return vr.typ, ok
}

func (vr *varDecl) buildIR(pkg *ir.Package) {
	if vr.typ == nil {
		vr.typ = invalid
	}
	vr.ext.TypeV = vr.typ.irType()
	pkg.Vars = append(pkg.Vars, &vr.ext)
}
