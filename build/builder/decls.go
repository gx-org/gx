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
		return appendRedeclaredError(d.pkgScope.Err(), name, node.ident(), prev.ident())
	}
	d.declarations.Store(name, node)
	return true
}

func (d *decls) registerFunc(fun function) (*processNodeT[function], bool) {
	pNode := newProcessNode[function](token.FUNC, fun.fnSource().Name, fun)
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

func (d *decls) resolveAll(pkgScope *pkgResolveScope) bool {
	ok := true
	// Build named types header. The underlying types are not being resolved yet.
	var nTypes []*irNamedType
	ibld := pkgScope.irBuilder()
	for pNode := range iterToken[*namedType](d.declarations, filterTok(token.TYPE)) {
		nType, nTypeOk := pNode.node.build(ibld)
		ibld.Set(pNode, nType.ext)
		nTypes = append(nTypes, nType)
		ok = ok && nTypeOk
	}
	// Build all variables.
	for pNode := range iterToken[*varExpr](d.declarations, filterTok(token.VAR)) {
		ext, vrOk := pNode.node.build(ibld)
		ibld.Set(pNode, ext)
		ok = ok && vrOk
	}
	// Build all constants.
	if compEvalOk := d.buildConstants(pkgScope); !compEvalOk {
		return false
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

type bConstExpr struct {
	bConst iConstExpr
	ext    *ir.ConstExpr
	deps   []*ast.Ident
}

func buildConstExpr(pkgScope *pkgResolveScope, consts *ordered.Map[string, *bConstExpr], done map[string]bool, cst *bConstExpr) bool {
	ibld := pkgScope.irBuilder()
	ok := true
	for _, dep := range cst.deps {
		depExpr, ok := consts.Load(dep.Name)
		if !ok {
			return pkgScope.Err().Appendf(dep, "constant %s undefined", dep.Name)
		}
		depOk := buildConstExpr(pkgScope, consts, done, depExpr)
		if !depOk {
			ok = false
		}
	}
	if !ok {
		return false
	}
	return cst.bConst.buildExpression(ibld, cst.ext)
}

func (d *decls) buildConstants(pkgScope *pkgResolveScope) bool {
	ibld := pkgScope.irBuilder()
	ok := true
	consts := ordered.NewMap[string, *bConstExpr]()
	for pNode := range iterToken[iConstExpr](d.declarations, filterTok(token.CONST)) {
		ext, deps, cstOk := pNode.node.buildDeclaration(ibld)
		consts.Store(ext.VName.Name, &bConstExpr{
			bConst: pNode.node,
			ext:    ext,
			deps:   deps,
		})
		ibld.Set(pNode, ext)
		ok = ok && cstOk
	}
	if !ok {
		return false
	}
	done := make(map[string]bool)
	for bConst := range consts.Values() {
		cstOk := buildConstExpr(pkgScope, consts, done, bConst)
		ok = ok && cstOk
	}
	return ok
}

type irFunc struct {
	pNode    *processNodeT[function]
	bFunc    function
	sigScope fnResolveScope
	irFunc   ir.PkgFunc
}

func (d *decls) buildFuncType(pkgScope *pkgResolveScope, pNode *processNodeT[function]) (*irFunc, bool) {
	fScope, fnOk := pkgScope.newFileRScope(pNode.node.file())
	if !fnOk {
		return nil, false
	}
	irFn, fnScope, fnOk := pNode.node.buildSignature(fScope)
	if !fnOk {
		return nil, false
	}
	pkgFn, fnOk := irFn.(ir.PkgFunc)
	if !fnOk {
		return nil, pkgScope.Err().AppendInternalf(irFn.Source(), "cannot cast %T to %s", irFn, reflect.TypeFor[*ir.PkgFunc]().Name())
	}
	ibld := pkgScope.irBuilder()
	ibld.Set(pNode, pkgFn)
	irf := &irFunc{
		pNode:    pNode,
		bFunc:    pNode.node,
		sigScope: fnScope,
		irFunc:   pkgFn,
	}
	recv := irf.sigScope.funcType().ReceiverField()
	if recv == nil {
		ibld.Register(funcDeclarator(irf.irFunc))
		return irf, fnOk
	}
	nType, ok := recv.Type().(*ir.NamedType)
	if !ok {
		return irf, pkgScope.Err().AppendInternalf(recv.Source(), "cannot cast %T to %s", recv.Type(), reflect.TypeFor[*ir.NamedType]().Name())
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
	// Build function signatures.
	for _, pNode := range filteredFuncs {
		fn, fnOk := d.buildFuncType(pkgScope, pNode)
		if !fnOk {
			ok = false
			continue
		}
		funcs = append(funcs, fn)
	}
	if !ok {
		return false
	}
	// Annotate functions.
	for _, fn := range funcs {
		fScope, fnOk := pkgScope.newFileRScope(fn.bFunc.file())
		if !fnOk {
			ok = false
			continue
		}
		fnOk = fn.bFunc.buildAnnotations(fScope, fn)
		ok = ok && fnOk
	}
	if !ok {
		return false
	}
	// Build functions bodies.
	return d.buildFunctionBodies(pkgScope, funcs)
}

func (d *decls) buildFunctionBodies(pkgScope *pkgResolveScope, funcs []*irFunc) bool {
	for _, fn := range funcs {
		ok := fn.bFunc.buildBody(fn.sigScope, fn)
		if !ok {
			return false
		}
	}
	return true
}

func (d *decls) Find(key string) (processNode, bool) {
	return d.declarations.Load(key)
}

func (d *decls) Items() *ordered.Map[string, processNode] {
	return d.declarations.Clone()
}

func (d *decls) String() string {
	kvs := make([]string, 0, d.declarations.Size())
	for k, v := range d.declarations.Iter() {
		kvs = append(kvs, fmt.Sprintf("%s: %T", k, v))
	}
	return strings.Join(kvs, "\n")
}
