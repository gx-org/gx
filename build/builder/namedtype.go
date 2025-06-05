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
	"github.com/gx-org/gx/build/fmterr"
	"github.com/gx-org/gx/build/ir"
)

// namedType is a node representing a named type declaration in GX source code.
type namedType struct {
	src        *ast.TypeSpec
	underlying typeExprNode
	file       *file
	methods    *ordered.Map[string, *processNodeT[function]]
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
		src:     src,
		file:    pscope.file(),
		methods: ordered.NewMap[string, *processNodeT[function]](),
	}
	var ok bool
	n.underlying, ok = processTypeExpr(pscope, src.Type)
	if !ok {
		return false
	}
	if src.TypeParams.NumFields() > 0 {
		pscope.err().Appendf(src, "type may not have type parameters")
		ok = false
	}
	pNode := newProcessNode(token.TYPE, src.Name, n)
	return pscope.decls().declarePackageName(pNode) && ok
}

func (n *namedType) source() ast.Node {
	return n.src
}

func (n *namedType) build(pkgScope *pkgResolveScope) (*irNamedType, declarator) {
	return &irNamedType{
		bType: n,
		irType: &ir.NamedType{
			Src: n.src,
		},
	}, namedTypeDeclarator(n.file)
}

func (n *namedType) buildUnderlying(pkgScope *pkgResolveScope, nType *ir.NamedType) bool {
	rscope, scopeOk := pkgScope.newFileScope(n.file, nil)
	if !scopeOk {
		return false
	}
	var underOk bool
	nType.Underlying, underOk = n.underlying.buildTypeExpr(rscope)
	return underOk
}

func (n *namedType) assignMethod(scope *fileResolveScope, pNode *processNodeT[function], ext *ir.NamedType) bool {
	underlying := ir.Underlying(ext)
	fn := pNode.node
	name := fn.name().Name
	if !ir.ValidName(name) {
		return true
	}
	// Check if a method has already been defined.
	if prev, exist := n.methods.Load(name); exist {
		return scope.err().Appendf(fn.source(), "method %s.%s already declared at %s", n.src.Name, name, funcPos(scope, prev.node))
	}
	// Check if a field with the same name has already been defined.
	structType, ok := underlying.(*ir.StructType)
	if ok {
		if defined := structType.Fields.FindField(name); defined != nil {
			return scope.err().Appendf(fn.source(), "field and method with the same name %s", name)
		}
	}
	n.methods.Store(name, pNode)
	return true
}

func funcPos(scope *fileResolveScope, fn function) string {
	fnPos, ok := fn.(interface{ source() ast.Node })
	if !ok {
		return "as a builtin"
	}
	return fmterr.PosString(scope.err().FSet().FSet, fnPos.source().Pos())
}

func (n *namedType) String() string {
	return n.src.Name.Name
}

func namedTypeDeclarator(bFile *file) declarator {
	return func(irb *irBuilder, decls *ir.Declarations, node *declNode) bool {
		file, ok := buildParentNode[*ir.File](irb, decls, bFile)
		if !ok {
			return false
		}
		nType := node.ir.(*ir.NamedType)
		nType.ID = len(decls.Types)
		nType.File = file
		decls.Types = append(decls.Types, nType)
		for _, method := range nType.Methods {
			if _, ok := setFuncFileField(irb, file, method); !ok {
				return false
			}
		}
		return true
	}
}
