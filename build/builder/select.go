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
	"reflect"

	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/build/ir/irkind"
)

// selectedExpr references an attribute in a package or a structure.
type selectorExpr struct {
	src *ast.SelectorExpr
	x   exprNode
}

var (
	_ exprNode     = (*selectorExpr)(nil)
	_ typeExprNode = (*selectorExpr)(nil)
)

func processSelectorExpr(pscope procScope, src *ast.SelectorExpr) (*selectorExpr, bool) {
	n := &selectorExpr{src: src}
	var ok bool
	n.x, ok = processExpr(pscope, src.X)
	return n, ok
}

func (n *selectorExpr) source() ast.Node {
	return n.src
}

func (n *selectorExpr) returnUndefined(scope resolveScope, x ir.Expr) (ir.Storage, bool) {
	return nil, scope.Err().Appendf(n.src.Sel, "undefined: %s.%s", n.x.String(), n.src.Sel.Name)
}

func (n *selectorExpr) selectFromPackage(scope resolveScope, sel *ir.SelectorExpr) (ir.Storage, bool) {
	val, ok := storageFromExpr(scope, sel.X)
	if !ok {
		return nil, false
	}
	if _, ok := val.(*ir.ImportDecl); !ok {
		return nil, scope.Err().AppendInternalf(sel.X.Node(), "%T is not %s", val, reflect.TypeFor[*ir.ImportDecl]())
	}
	pkg := scope.fileScope().deps[val.NameDef().Name]
	if !ir.IsExported(n.src.Sel.Name) {
		return nil, scope.Err().Appendf(n.src.Sel, "%s is not exported", n.src.Sel.Name)
	}
	store := pkg.names[n.src.Sel.Name]
	if store == nil {
		return n.returnUndefined(scope, sel.X)
	}
	return store, true
}

func (n *selectorExpr) selectFromExpr(scope resolveScope, sel *ir.SelectorExpr, exprType ir.Type) (ir.Storage, bool) {
	method, field := sel.Select(exprType)
	if method != nil {
		return method, true
	}
	if field != nil {
		return field.Storage(), true
	}
	return n.returnUndefined(scope, sel.X)
}

func (n *selectorExpr) selectFromType(scope resolveScope, sel *ir.SelectorExpr) (ir.Storage, bool) {
	xT, ok := sel.X.(*ir.Ident)
	if !ok {
		return n.returnUndefined(scope, sel.X)
	}
	exprType, ok := xT.Stor.(ir.Type)
	if !ok {
		return n.returnUndefined(scope, sel.X)
	}
	return n.selectFromExpr(scope, sel, exprType)
}

func (n *selectorExpr) selectStorageFrom(scope resolveScope, sel *ir.SelectorExpr) (ir.Storage, bool) {
	if sel.X.Type().Kind() == irkind.Package {
		return n.selectFromPackage(scope, sel)
	}
	if sel.X.Type().Kind() == irkind.MetaType {
		return n.selectFromType(scope, sel)
	}
	return n.selectFromExpr(scope, sel, sel.X.Type())
}

func (n *selectorExpr) buildSelectorExpr(scope resolveScope) (*ir.SelectorExpr, bool) {
	ext := &ir.SelectorExpr{Src: n.src}
	var ok bool
	ext.X, ok = n.x.buildExpr(scope)
	if !ok {
		return ext, false
	}
	ext.Stor, ok = n.selectStorageFrom(scope, ext)
	return ext, ok
}

func (n *selectorExpr) buildExpr(scope resolveScope) (ir.Expr, bool) {
	return n.buildSelectorExpr(scope)
}

func (n *selectorExpr) buildTypeExpr(rscope resolveScope) (*ir.TypeValExpr, bool) {
	sel, ok := n.buildSelectorExpr(rscope)
	if !ok {
		return invalidTypeExprVal, false
	}
	return typeFromStorage(rscope, sel, sel.Stor)
}

func (n *selectorExpr) String() string {
	return fmt.Sprintf("%s.%s", n.x.String(), n.src.Sel.Name)
}
