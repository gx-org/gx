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
	"math/big"

	"github.com/gx-org/gx/api/options"
	gxfmt "github.com/gx-org/gx/base/fmt"
	"github.com/gx-org/gx/build/fmterr"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/base/scope"
	"github.com/gx-org/gx/internal/interp/compeval"
	"github.com/gx-org/gx/interp/elements"
	"github.com/gx-org/gx/interp/evaluator"
	"github.com/gx-org/gx/interp"
)

var irBuiltins = map[string]processNode{}

func init() {
	// Builtin types.
	registerBuiltinType("any", ir.AnyType())
	registerBuiltinType("bool", ir.BoolType())
	registerBuiltinType("bfloat16", ir.Bfloat16Type())
	registerBuiltinType("float32", ir.Float32Type())
	registerBuiltinType("float64", ir.Float64Type())
	registerBuiltinType("int32", ir.Int32Type())
	registerBuiltinType("int64", ir.Int64Type())
	registerBuiltinType("string", ir.StringType())
	registerBuiltinType("uint32", ir.Uint32Type())
	registerBuiltinType("uint64", ir.Uint64Type())
	registerBuiltinType("intlen", ir.IntLenType())
	registerBuiltinType("intidx", ir.IntIndexType())

	// Builtin values.
	registerBuiltin(ir.FalseStorage())
	registerBuiltin(ir.TrueStorage())

	// Builtin functions.
	registerFuncBuiltin(appendName, &appendFunc{})
	registerFuncBuiltin("axlengths", &axlengthsFunc{})
	registerFuncBuiltin("set", &setFunc{})
	registerFuncBuiltin("trace", &traceFunc{})
}

func registerBuiltin(store ir.Storage) {
	name := store.NameDef().Name
	irBuiltins[name] = pNodeFromIR(token.CONST, name, store, nil)
}

func registerBuiltinType(name string, node ir.Type) {
	registerBuiltin(ir.BuiltinStorage(name, &ir.TypeValExpr{Typ: node}))
}

func registerFuncBuiltin(name string, impl ir.FuncImpl) {
	irBuiltins[name] = pNodeFromIR(token.FUNC, name, &ir.FuncBuiltin{
		Src:  &ast.FuncDecl{Name: &ast.Ident{Name: name}},
		Impl: impl,
	}, nil)
}

type (
	// pkgResolveScope is a package and its namespace with an error accumulator.
	// This context is used in the resolve phase.
	pkgResolveScope struct {
		*pkgProcScope
		// namedTypes maps build named types to IR named types.
		namedTypes map[*namedType]*ir.NamedType
		// Package namespace. Includes all the builtins as well as all package declarations.
		nspace scope.Scope[processNode]
	}
)

func newPackageResolveScope(pscope *pkgProcScope) *pkgResolveScope {
	ns := scope.NewScopeWithValues(irBuiltins)
	ns = scope.NewReadOnly[processNode](ns, pscope.decls())
	return &pkgResolveScope{
		pkgProcScope: pscope,
		nspace:       ns,
	}
}

func (s *pkgResolveScope) packageContext() (*irBuilder, evaluator.EvaluationContext) {
	hostEval := compeval.NewHostEvaluator(s.bpkg.builder())
	irb, ok := s.newIRBuilder()
	if !ok {
		return nil, nil
	}
	pkg := irb.irPkg()
	decls, ok := s.dcls.collectDeclNodes(hasIR)
	if !ok {
		return nil, nil
	}
	if pkg.Decls, ok = buildDeclarations(irb, decls); !ok {
		return nil, nil
	}
	var opts []options.PackageOption
	for _, decl := range pkg.Decls.Vars {
		for _, vr := range decl.Exprs {
			opt := compeval.NewOptionVariable(vr)
			opts = append(opts, opt)
		}
	}
	ectx, err := interp.NewInterpContext(hostEval, opts)
	if err != nil {
		s.err().Append(err)
		return nil, nil
	}
	return irb, ectx
}

func (s *pkgResolveScope) namedTypeIR(nType *namedType) *ir.NamedType {
	return s.namedTypes[nType]
}

func (s *pkgResolveScope) String() string {
	return gxfmt.String(s.nspace)
}

type (
	resolveScope interface {
		fileScope() *fileResolveScope // TODO(degris): replace this method with irBuilder.
		ns() *scope.RWScope[processNode]
		err() *fmterr.Appender
		compEval() (*compileEvaluator, bool)
	}

	pNodeProcessor func(processNode) bool

	fileResolveScope struct {
		*pkgResolveScope

		bFile  *file
		ev     *compileEvaluator
		nspace *scope.RWScope[processNode]
		deps   map[string]*importedPackage
		proc   pNodeProcessor // TODO(degris): remove this by running two passes for constants.
	}
)

var _ resolveScope = (*fileResolveScope)(nil)

func (s *pkgResolveScope) newFileScope(f *file, proc pNodeProcessor) (*fileResolveScope, bool) {
	fScope := &fileResolveScope{
		pkgResolveScope: s,
		bFile:           f,
		nspace:          scope.NewScope(s.nspace),
		deps:            make(map[string]*importedPackage),
		proc:            proc,
	}
	ok := true
	for _, decl := range f.imports {
		dep, depOk := importPackage(s, decl)
		name := decl.Name()
		defineGlobal(fScope.nspace, token.IMPORT, name, decl)
		fScope.deps[name] = dep
		ok = ok && depOk
	}
	return fScope, ok
}

func (s *fileResolveScope) process(pNode processNode) bool {
	if s.proc == nil {
		return true
	}
	return s.proc(pNode)
}

func (s *fileResolveScope) compEval() (*compileEvaluator, bool) {
	irb, pkgctx := s.pkgResolveScope.packageContext()
	if pkgctx == nil {
		return nil, false
	}
	ctx, err := pkgctx.NewFileContext(&ir.File{
		Package: irb.pkg,
		Src:     s.bFile.src,
		Imports: s.bFile.imports,
	})
	if err != nil {
		return nil, s.err().Append(err)
	}
	return newEvaluator(s, irb, ctx)
}

func (s *fileResolveScope) fileScope() *fileResolveScope {
	return s
}

func (s *fileResolveScope) ns() *scope.RWScope[processNode] {
	return s.nspace
}

func (s *fileResolveScope) File() *file {
	return s.bFile
}

func (s *fileResolveScope) funcType() *ir.FuncType {
	return nil
}

type constResolveScope struct {
	resolveScope
}

func newConstResolveScope(pkgScope *pkgResolveScope, f *file) (*constResolveScope, bool) {
	s := &constResolveScope{}
	var ok bool
	s.resolveScope, ok = pkgScope.newFileScope(f, s.proc)
	return s, ok
}

func (s *constResolveScope) proc(pNode processNode) bool {
	if pNode.token() != token.CONST {
		return s.err().Appendf(pNode.ident(), "%s:%s is not constant", pNode.ident().Name, pNode.token())
	}
	if pNode.ir() != nil {
		return true
	}
	cstNode := pNode.(*processNodeT[*constExpr])
	cst, cstOk := cstNode.node.buildExpr(s)
	cstNode.setDeclNode(cst, constDeclarator(cstNode.node.spec))
	return cstOk
}

type (
	iFuncResolveScope interface {
		resolveScope
		funcType() *ir.FuncType
		declarator(fn function) declarator
	}

	funcResolveScope struct {
		resolveScope
		fType  *ir.FuncType
		nspace *scope.RWScope[processNode]
		names  map[string]elements.Element
	}
)

var _ localScope = (*funcResolveScope)(nil)

func newFuncScope(rscope resolveScope, fType *ir.FuncType) *funcResolveScope {
	ns := scope.NewScope(rscope.ns())
	return &funcResolveScope{
		resolveScope: rscope,
		fType:        fType,
		nspace:       scope.NewScope(ns.ReadOnly()),
		names:        make(map[string]elements.Element),
	}
}

func (s *funcResolveScope) funcType() *ir.FuncType {
	return s.fType
}

func (s *funcResolveScope) ns() *scope.RWScope[processNode] {
	return s.nspace
}

func (s *funcResolveScope) update(store ir.Storage, el elements.Element) bool {
	name := store.NameDef().Name
	s.nspace.Define(name, pNodeFromIR(token.VAR, name, store, nil))
	s.names[name] = el
	return true
}

func (s *funcResolveScope) compEval() (*compileEvaluator, bool) {
	fileCEval, ok := s.fileScope().compEval()
	if !ok {
		return fileCEval, false
	}
	return fileCEval.sub(s.fType.Source(), s.names)
}

func (s *funcResolveScope) declarator(fn function) declarator {
	return funcDeclarator(s.resolveScope.fileScope().bFile, fn)
}

type (
	localScope interface {
		resolveScope
		update(s ir.Storage, el elements.Element) bool
	}

	blockResolveScope struct {
		iFuncResolveScope
		nspace   *scope.RWScope[processNode]
		compeval *compileEvaluator
	}
)

var _ localScope = (*blockResolveScope)(nil)

func newBlockScope(rscope iFuncResolveScope) (*blockResolveScope, bool) {
	s := &blockResolveScope{
		iFuncResolveScope: rscope,
		nspace:            scope.NewScope(rscope.ns()),
	}
	parentCompEval, ok := s.iFuncResolveScope.compEval()
	if !ok {
		return s, false
	}
	s.compeval, ok = parentCompEval.sub(s.iFuncResolveScope.funcType().Src, nil)
	return s, ok
}

func (s *blockResolveScope) ns() *scope.RWScope[processNode] {
	return s.nspace
}

func (s *blockResolveScope) update(store ir.Storage, el elements.Element) bool {
	name := store.NameDef().Name
	s.nspace.Define(name, pNodeFromIR(token.VAR, name, store, nil))
	var ok bool
	s.compeval, ok = s.compeval.update(s, store, el)
	return ok
}

func (s *blockResolveScope) compEval() (*compileEvaluator, bool) {
	return s.compeval, true
}

type (
	compositeLitResolveScope interface {
		resolveScope
		dtype() ir.Type
		sub(ast.Node) (compositeLitResolveScope, bool)
		want() ir.Type
		newInferCompositeType(src *ast.CompositeLit, exprs []ir.AssignableExpr) (ir.Expr, bool)
	}

	arrayResolveScope struct {
		resolveScope
		parent      *arrayResolveScope
		currentRank int
		current     ir.ArrayType
	}
)

var _ compositeLitResolveScope = (*arrayResolveScope)(nil)

func toArrayLitScope(rscope resolveScope, want ir.ArrayType) *arrayResolveScope {
	ascope, ok := rscope.(*arrayResolveScope)
	if !ok {
		// Top-level of parsing a literal: create a new array scope.
		return &arrayResolveScope{
			resolveScope: rscope,
			current:      want,
		}
	}
	// We are parsing a sub-rank literal: nothing to do.
	// The correct scope was already created with the arrayResolveScope.sub method.
	return ascope
}

func (s *arrayResolveScope) dtype() ir.Type {
	return s.current.DataType()
}

func (s *arrayResolveScope) want() ir.Type {
	return s.current
}

func (s *arrayResolveScope) appendAxisToInferredRanks(ax ir.AxisLengths) {
	infer := toInferRank(s.current.Rank())
	if infer == nil {
		return
	}
	underlying := underlyingRank(infer)
	underlying.Ax = append(underlying.Ax, ax)
	if s.parent == nil {
		return
	}
	s.parent.appendAxisToInferredRanks(ax)
}

func (s *arrayResolveScope) sub(src ast.Node) (compositeLitResolveScope, bool) {
	elt, ok := s.current.ElementType()
	if !ok {
		return s, s.err().AppendInternalf(src, "unexpected literal for type %s ", s.current.String())
	}
	eltArray := ir.ToArrayType(elt)
	if eltArray == nil {
		return s, s.err().AppendInternalf(src, "invalid element type %s ", elt.String())
	}
	currentRank := s.currentRank + 1
	if infer := toInferRank(s.current.Rank()); infer != nil {
		if infer.Rnk == nil {
			infer.Rnk = &ir.Rank{
				Ax: []ir.AxisLengths{&ir.AxisInfer{}},
			}
		}
	}
	return &arrayResolveScope{
		resolveScope: s.resolveScope,
		parent:       s,
		current:      eltArray,
		currentRank:  currentRank,
	}, true
}

func underlyingRank(r ir.ArrayRank) *ir.Rank {
	switch rT := r.(type) {
	case *ir.Rank:
		return rT
	case *ir.RankInfer:
		if rT.Rnk == nil {
			rT.Rnk = &ir.Rank{}
		}
		return underlyingRank(rT.Rnk)
	}
	return nil
}

func toInferRank(r ir.ArrayRank) *ir.RankInfer {
	infer, ok := r.(*ir.RankInfer)
	if !ok {
		return nil
	}
	return infer
}

func (s *arrayResolveScope) rootRank() ir.ArrayRank {
	cur := s
	for cur.parent != nil {
		cur = cur.parent
	}
	return cur.current.Rank()
}

func (s *arrayResolveScope) newInferCompositeType(src *ast.CompositeLit, exprs []ir.AssignableExpr) (ir.Expr, bool) {
	numExprs := len(exprs)
	// Implicit literal: we require an explicit rank.
	parentInfer := toInferRank(s.rootRank())
	ext := &ir.ArrayLitExpr{Src: src, Typ: s.current, Elts: exprs}
	if numExprs == 0 {
		if parentInfer == nil {
			return ext, true
		}
		return ext, s.err().Appendf(src, "cannot infer rank: empty literal")
	}
	// We can now check compare the number of elements to the axis length.
	got := &ir.AxisExpr{X: &ir.NumberCastExpr{
		X: &ir.NumberInt{
			Src: &ast.BasicLit{ValuePos: src.Pos()},
			Val: big.NewInt(int64(numExprs)),
		},
		Typ: ir.IntLenType(),
	}}
	axis := s.current.Rank().Axes()[0]
	toInfer, _ := axis.(*ir.AxisInfer)
	if toInfer != nil && toInfer.X == nil {
		toInfer.X = got
		if s.parent != nil {
			s.parent.appendAxisToInferredRanks(toInfer)
		}
	}
	if !axisEqual(s, src, axis, got) {
		return ext, s.err().Appendf(src, "cannot assign %d element(s) to axis length %s", numExprs, axis.String())
	}
	return ext, true
}

type sliceResolveScope struct {
	resolveScope
	dt  *ir.TypeValExpr
	typ ir.Type
}

var _ compositeLitResolveScope = (*sliceResolveScope)(nil)

func newSliceLitScope(rscope resolveScope, want *ir.SliceType) *sliceResolveScope {
	return &sliceResolveScope{
		resolveScope: rscope,
		dt:           want.DType,
		typ:          want,
	}
}

func (s *sliceResolveScope) dtype() ir.Type {
	return s.dt.Typ
}

func (s *sliceResolveScope) sub(src ast.Node) (compositeLitResolveScope, bool) {
	slicer, ok := s.typ.(ir.SlicerType)
	if !ok {
		return s, s.err().AppendInternalf(src, "invalid composite literal element type %s ", s.typ.String())
	}
	elt, ok := slicer.ElementType()
	if !ok {
		return s, s.err().AppendInternalf(src, "unexpected literal for type %s ", slicer.String())
	}
	return &sliceResolveScope{
		resolveScope: s,
		dt:           s.dt,
		typ:          elt,
	}, true
}

func (s *sliceResolveScope) want() ir.Type {
	return s.typ
}

func (s *sliceResolveScope) newInferCompositeType(src *ast.CompositeLit, exprs []ir.AssignableExpr) (ir.Expr, bool) {
	return &ir.SliceLitExpr{
		Src:  src,
		Elts: exprs,
		Typ:  s.want(),
	}, true
}

type (
	defineLocalF     func(s resolveScope, storage ir.Storage) bool
	defineLocalScope struct {
		localScope
		def     defineLocalF
		defAxis defineLocalF
	}
)

func toDefineScope(scope resolveScope) *defineLocalScope {
	dScope, ok := scope.(*defineLocalScope)
	if ok {
		return dScope
	}
	local, ok := scope.(localScope)
	if !ok {
		local = newFuncScope(scope, &ir.FuncType{})
	}
	return &defineLocalScope{localScope: local}
}

func newDefineScope(scope localScope, def defineLocalF, defAxis defineLocalF) *defineLocalScope {
	return &defineLocalScope{localScope: scope, def: def, defAxis: defAxis}
}

func (s *defineLocalScope) defineAxis(storage *ir.AxLengthName) {
	if s.defAxis == nil {
		return
	}
	s.defAxis(s, storage)
}

func (s *defineLocalScope) define(storage ir.Storage) {
	if s.def == nil {
		return
	}
	s.def(s, storage)
}
