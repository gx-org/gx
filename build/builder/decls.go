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
	"slices"
	"sort"
	"strings"

	"github.com/gx-org/gx/base/iter"
	"github.com/gx-org/gx/base/ordered"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/base/scope"
	"github.com/gx-org/gx/internal/interp/compeval/cpevelements"
)

type (
	parentNodeBuilder interface {
		buildParentNode(*irBuilder, *ir.Declarations) (ir.Node, bool)
		file() *file
	}

	irBuilder struct {
		pkgScope *pkgResolveScope
		built    map[parentNodeBuilder]ir.Node
		pkg      *ir.Package
	}
)

func (s *pkgResolveScope) newIRBuilder() (*irBuilder, bool) {
	irb := &irBuilder{
		pkgScope: s,
		built:    make(map[parentNodeBuilder]ir.Node),
		pkg:      s.bpkg.newPackageIR(),
	}
	ok := true
	for file := range s.bpkg.files.Values() {
		_, fileOk := buildParentNode[*ir.File](irb, irb.pkg.Decls, file)
		ok = ok && fileOk
	}
	return irb, ok
}

func (irb *irBuilder) irPkg() *ir.Package {
	return irb.pkg
}

type (
	declarator func(*irBuilder, *ir.Declarations, *declNode) bool

	declNode struct {
		pNode   processNode
		id      nodeID
		ir      ir.Storage
		declare declarator
	}
)

func (d *declNode) clone() processNode {
	return d.pNode.clone()
}

func (p *processNodeT[T]) setDeclNode(ir ir.Storage, declare declarator) *declNode {
	p.decl = &declNode{pNode: p, id: p.id, ir: ir, declare: declare}
	return p.decl
}

// buildParentNode builds a IR node if it has not been done before.
// Use for *ir.ConstDecl and *ir.VarDecl
func buildParentNode[T ir.Node](irb *irBuilder, decls *ir.Declarations, bld parentNodeBuilder) (t T, ok bool) {
	node, ok := irb.built[bld]
	if ok {
		return node.(T), true
	}
	node, ok = bld.buildParentNode(irb, decls)
	if !ok {
		return node.(T), false
	}
	irb.built[bld] = node
	return node.(T), true
}

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
	if fun.receiver() != nil {
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

type irNamedType struct {
	bType  *namedType
	irType *ir.NamedType
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

func filterToBuild(n processNode) bool {
	return n.declNode() == nil
}

func hasIR(n processNode) bool {
	dNode := n.declNode()
	if dNode == nil {
		return false
	}
	return dNode.ir != nil
}

func (d *decls) resolveAll(pkgScope *pkgResolveScope) bool {
	// Build named types header. The underlying types are not being resolved yet.
	var namedTypes []*irNamedType
	for pNode := range iterToken[*namedType](d.declarations, filterTok(token.TYPE), filterToBuild) {
		namedType, declarator := pNode.node.build(pkgScope)
		namedTypes = append(namedTypes, namedType)
		pNode.setDeclNode(namedType.irType, declarator)
	}
	ok := true
	// Build all variables.
	for pNode := range iterToken[*varExpr](d.declarations, filterTok(token.VAR), filterToBuild) {
		vr, vrOk := pNode.node.build(pkgScope)
		pNode.setDeclNode(vr, varDeclarator(pNode.node.spec))
		ok = ok && vrOk
	}
	// Build all constants.
	for pNode := range iterToken[*constExpr](d.declarations, filterTok(token.CONST), filterToBuild) {
		cst, cstOk := pNode.node.build(pkgScope)
		pNode.setDeclNode(cst, constDeclarator(pNode.node.spec))
		ok = ok && cstOk
	}
	// Build compeval function signature.
	if compEvalOk := d.buildFunctions(pkgScope, filterCompEval(true)); !compEvalOk {
		return false
	}
	// Resolve underlying types.
	for _, namedType := range namedTypes {
		underOk := namedType.bType.buildUnderlying(pkgScope, namedType.irType)
		ok = ok && underOk
	}
	// Build functions and methods.
	funOk := d.buildFunctions(pkgScope, filterCompEval(false))
	return funOk && ok
}

type irFunc struct {
	bFunc     function
	scopeFunc iFuncResolveScope
	irFunc    ir.Func
}

func (d *decls) buildFuncType(pkgScope *pkgResolveScope, pNode *processNodeT[function]) (*irFunc, bool) {
	irFn, fScope, fnOk := pNode.node.buildSignature(pkgScope)
	if !fnOk {
		return nil, false
	}
	recvOk := true
	pNode.decl = pNode.setDeclNode(irFn.(ir.Storage), fScope.declarator(pNode.node))
	if recv := pNode.node.receiver(); recv != nil {
		recvOk = d.registerMethodToType(fScope.fileScope(), recv, pNode)
	}
	if !recvOk || !fnOk {
		return nil, false
	}
	return &irFunc{
		bFunc:     pNode.node,
		scopeFunc: fScope,
		irFunc:    irFn,
	}, true
}

func (d *decls) registerAuxFunc(pkgScope *pkgResolveScope, aux *cpevelements.SyntheticFuncDecl) (*irFunc, bool) {
	bFile := newFile(pkgScope.pkg(), aux.F.Name()+".syn.gx", &ast.File{})
	fScope, ok := pkgScope.newFileScope(bFile, nil)
	if !ok {
		return nil, false
	}
	fnScope := newFuncScope(fScope, aux.F.FType)
	irf := irFunc{
		bFunc: &syntheticFunc{
			bFile: bFile,
			src:   aux.F.Src,
		},
		scopeFunc: &macroResolveScope{
			iFuncResolveScope: fnScope,
			sFunc:             aux.SyntheticFunc,
		},
		irFunc: aux.F,
	}
	pNode, ok := d.registerFunc(irf.bFunc)
	if !ok {
		return nil, false
	}
	pNode.decl = pNode.setDeclNode(aux.F, fnScope.declarator(pNode.node))
	return &irf, true
}

func (d *decls) buildFunctions(pkgScope *pkgResolveScope, filter func(f *processNodeT[function]) bool) bool {
	ok := true
	var funcs []*irFunc
	packageFuncs := slices.Collect(iterToken[function](d.declarations, filterTok(token.FUNC), filterToBuild))
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

func (d *decls) buildFunctionBodies(pkgScope *pkgResolveScope, funcs []*irFunc) ([]*irFunc, bool) {
	var todoNext []*irFunc
	ok := true
	for _, fn := range funcs {
		auxs, fnOk := fn.bFunc.buildBody(fn.scopeFunc, fn.irFunc)
		for _, aux := range auxs {
			todo, auxOk := d.registerAuxFunc(pkgScope, aux)
			ok = ok && auxOk
			if !auxOk {
				continue
			}
			todoNext = append(todoNext, todo)
		}
		ok = ok && fnOk
	}
	return todoNext, ok
}

func (d *decls) registerMethodToType(rscope *fileResolveScope, recv *fieldList, fnNode *processNodeT[function]) bool {
	src := recv.list[0].typ.source()
	recvTypeName := src.(*ast.Ident).Name
	pNode, exist := d.declarations.Load(recvTypeName)
	if !exist {
		return d.pkgScope.err().Appendf(src, "undefined: %s", recvTypeName)
	}
	switch nodeT := pNode.bNode().(type) {
	case *namedType:
		namedTypeIR := pNode.ir().(*ir.NamedType)
		if ok := nodeT.assignMethod(rscope, fnNode, namedTypeIR); !ok {
			return false
		}
		namedTypeIR.Methods = append(namedTypeIR.Methods, fnNode.ir().(ir.PkgFunc))
	case *ir.NamedType:
		return d.pkgScope.err().Appendf(src, "adding methods to a structure in a different cell not supported yet")
	default:
		return d.pkgScope.err().Appendf(src, "%s is not a type", recvTypeName)
	}
	return true
}

func (d *decls) collectDeclNodes(filters ...nodeFilter) (*ordered.Map[string, *declNode], bool) {
	pkgDecls := ordered.NewMap[string, *declNode]()
	ok := true
	for name, node := range d.declarations.Iter() {
		if !filterNode(filters, node) {
			continue
		}
		decl := node.declNode()
		if decl == nil {
			ok = d.pkgScope.err().AppendInternalf(node.ident(), "process node %s:%s has not been built", node.ident().Name, node.token().String())
		}
		pkgDecls.Store(name, decl)
	}
	return pkgDecls, ok
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

func buildDeclarations(irb *irBuilder, declarations *ordered.Map[string, *declNode]) (*ir.Declarations, bool) {
	decls := &ir.Declarations{Package: irb.irPkg()}
	ok := true
	for decl := range declarations.Values() {
		declOk := decl.declare(irb, decls, decl)
		ok = ok && declOk
	}
	return decls, ok
}
