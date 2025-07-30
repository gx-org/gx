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
	"reflect"
	"slices"
	"sort"
	"strings"

	"github.com/gx-org/gx/base/iter"
	"github.com/gx-org/gx/base/ordered"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/base/scope"
	"github.com/gx-org/gx/internal/interp/compeval/cpevelements"
)

type decls struct {
	pkgScope *pkgProcScope

	// declarations at the package level.
	declarations *ordered.Map[string, processNode]

	// methods stores pointers to function that have not been assigned
	// to their named type yet.
	methods []*processNodeT[function]

	// For incremental packages, it is ok to redeclare a package level name.
	overwriteOk bool
}

var _ scope.Scope[processNode] = (*decls)(nil)

func newDecls(overwriteOk bool, pkgScope *pkgProcScope) *decls {
	return &decls{
		pkgScope:     pkgScope,
		declarations: ordered.NewMap[string, processNode](),
		overwriteOk:  overwriteOk,
	}
}

func (d *decls) declarePackageName(node processNode) (ok bool) {
	name := node.ident().Name
	prev, exist := d.declarations.Load(name)
	if !d.overwriteOk && exist {
		return appendRedeclaredError(d.pkgScope.err(), name, node.ident(), prev.ident())
	}
	d.declarations.Store(name, node)
	return true
}

func (d *decls) registerFunc(fun function) (*processNodeT[function], bool) {
	pNode := newProcessNode[function](token.FUNC, fun.name(), fun)
	if fun.isMethod() {
		d.methods = append(d.methods, pNode)
		return pNode, true
	}
	return pNode, d.declarePackageName(pNode)
}

func (d *decls) registerStaticVar(spec *varSpec) bool {
	ok := true
	for _, expr := range spec.exprs {
		ok = d.declarePackageName(expr.pNode()) && ok
	}
	return ok
}

func (d *decls) registerConst(spec *constSpec) bool {
	ok := true
	for _, expr := range spec.exprs {
		ok = d.declarePackageName(expr.pNode()) && ok
	}
	return ok
}

type nodeFilter func(processNode) bool

func filterNode(filters []nodeFilter, pnode processNode) bool {
	for _, filter := range filters {
		if !filter(pnode) {
			return false
		}
	}
	return true
}

func iterToken[T any](decls *ordered.Map[string, processNode], filters ...nodeFilter) func(func(*processNodeT[T]) bool) {
	return func(yield func(*processNodeT[T]) bool) {
		for v := range decls.Values() {
			if !filterNode(filters, v) {
				continue
			}
			if !yield(v.(*processNodeT[T])) {
				return
			}
		}
	}
}

func filterCompEval(cpev bool) func(*processNodeT[function]) bool {
	return func(f *processNodeT[function]) bool {
		return f.node.compEval() == cpev
	}
}

func filterTok(tokens ...token.Token) func(processNode) bool {
	return func(n processNode) bool {
		tok := n.token()
		for _, want := range tokens {
			if tok == want {
				return true
			}
		}
		return false
	}
}

type bConstExpr struct {
	bConst iConstExpr
	ext    *ir.ConstExpr
}

func (d *decls) resolveAll(pkgScope *pkgResolveScope) bool {
	ok := true
	// Build named types header. The underlying types are not being resolved yet.
	var nTypes []*irNamedType
	for pNode := range iterToken[*namedType](d.declarations, filterTok(token.TYPE)) {
		nType, nTypeOk := pNode.node.build(pkgScope.ibld)
		pkgScope.ibld.Set(pNode, nType.ext)
		nTypes = append(nTypes, nType)
		ok = ok && nTypeOk
	}
	// Build all variables.
	for pNode := range iterToken[*varExpr](d.declarations, filterTok(token.VAR)) {
		ext, vrOk := pNode.node.build(pkgScope.ibld)
		pkgScope.ibld.Set(pNode, ext)
		ok = ok && vrOk
	}
	// Build all constants.
	var consts []bConstExpr
	for pNode := range iterToken[iConstExpr](d.declarations, filterTok(token.CONST)) {
		ext, cstOk := pNode.node.buildDeclaration(pkgScope.ibld)
		consts = append(consts, bConstExpr{
			bConst: pNode.node,
			ext:    ext,
		})
		pkgScope.ibld.Set(pNode, ext)
		ok = ok && cstOk
	}
	if !ok {
		return false
	}
	for _, cst := range consts {
		cstOk := cst.bConst.buildExpression(pkgScope.ibld, cst.ext)
		ok = ok && cstOk
	}
	// Build compeval function signature.
	if compEvalOk := d.buildFunctions(pkgScope, filterCompEval(true)); !compEvalOk {
		return false
	}
	// Resolve underlying types.
	for _, namedType := range nTypes {
		underOk := namedType.bType.buildUnderlying(pkgScope, namedType.ext)
		ok = ok && underOk
	}
	// Build functions and methods.
	funOk := d.buildFunctions(pkgScope, filterCompEval(false))
	return funOk && ok
}

type irFunc struct {
	pNode     *processNodeT[function]
	bFunc     function
	scopeFunc iFuncResolveScope
	irFunc    ir.PkgFunc
}

func (d *decls) buildFuncType(pkgScope *pkgResolveScope, pNode *processNodeT[function]) (*irFunc, bool) {
	irFn, fnScope, fnOk := pNode.node.buildSignature(pkgScope)
	if !fnOk {
		return nil, false
	}
	pkgFn, fnOk := irFn.(ir.PkgFunc)
	if !fnOk {
		return nil, pkgScope.err().AppendInternalf(irFn.Source(), "cannot cast %T to %s", irFn, reflect.TypeFor[*ir.PkgFunc]().Name())
	}
	pkgScope.ibld.Set(pNode, pkgFn)
	irf := &irFunc{
		pNode:     pNode,
		bFunc:     pNode.node,
		scopeFunc: fnScope,
		irFunc:    pkgFn,
	}
	recv := irf.scopeFunc.funcType().ReceiverField()
	if recv == nil {
		pkgScope.ibld.Register(funcDeclarator(irf.irFunc))
		return irf, fnOk
	}
	nType, ok := recv.Type().(*ir.NamedType)
	if !ok {
		return irf, pkgScope.err().AppendInternalf(recv.Source(), "cannot cast %T to %s", recv.Type(), reflect.TypeFor[*ir.NamedType]().Name())
	}
	assignMethod(fnScope.fileScope(), nType, irf)
	return irf, true
}

func (d *decls) buildFunctions(pkgScope *pkgResolveScope, filter func(f *processNodeT[function]) bool) bool {
	ok := true
	var funcs []*irFunc
	packageFuncs := slices.Collect(iterToken[function](d.declarations, filterTok(token.FUNC)))
	filteredFuncs := slices.Collect(iter.Filter(filter, packageFuncs, d.methods))
	sort.Slice(filteredFuncs, func(i, j int) bool {
		return filteredFuncs[i].node.resolveOrder() < filteredFuncs[j].node.resolveOrder()
	})
	for _, pNode := range filteredFuncs {
		irF, fnOk := d.buildFuncType(pkgScope, pNode)
		if !fnOk {
			ok = false
			continue
		}
		funcs = append(funcs, irF)
	}
	if !ok {
		return false
	}
	for len(funcs) > 0 {
		funcs, ok = d.buildFunctionBodies(pkgScope, funcs)
		if !ok {
			return false
		}
	}
	return ok
}

func (d *decls) registerAuxFuncs(mScope *synthResolveScope, auxs []*cpevelements.SyntheticFuncDecl) (todos []*irFunc, ok bool) {
	fScope := mScope.fileScope()
	for _, aux := range auxs {
		bFile := newFile(fScope.pkg(), aux.F.Name.Name+"_synt.gx", &ast.File{})
		pNode, ok := d.registerFunc(&syntheticFunc{
			coreSyntheticFunc: coreSyntheticFunc{
				bFile: bFile,
				src:   aux.F,
			},
			fnBuilder: aux.SyntheticFunc,
		})
		if !ok {
			return nil, false
		}
		irF, fnOk := d.buildFuncType(fScope.pkgResolveScope, pNode)
		if !fnOk {
			ok = false
			continue
		}
		todos = append(todos, irF)
	}
	return todos, true
}

func (d *decls) buildFunctionBodies(pkgScope *pkgResolveScope, funcs []*irFunc) ([]*irFunc, bool) {
	var todoNext []*irFunc
	for _, fn := range funcs {
		todos, ok := fn.bFunc.buildBody(fn.scopeFunc, fn)
		if !ok {
			return nil, false
		}
		todoNext = append(todoNext, todos...)
	}
	return todoNext, true
}

func (d *decls) Find(key string) (processNode, bool) {
	return d.declarations.Load(key)
}

func (d *decls) CanAssign(key string) bool {
	return false
}

func (d *decls) String() string {
	kvs := make([]string, 0, d.declarations.Size())
	for k, v := range d.declarations.Iter() {
		kvs = append(kvs, fmt.Sprintf("%s: %T", k, v))
	}
	return strings.Join(kvs, "\n")
}
