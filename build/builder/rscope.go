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
	"reflect"

	"github.com/gx-org/gx/api/options"
	gxfmt "github.com/gx-org/gx/base/fmt"
	"github.com/gx-org/gx/base/ordered"
	"github.com/gx-org/gx/build/builder/builtins"
	"github.com/gx-org/gx/build/builder/irb"
	"github.com/gx-org/gx/build/fmterr"
	"github.com/gx-org/gx/build/ir"
	"github.com/gx-org/gx/internal/base/scope"
	"github.com/gx-org/gx/internal/interp/compeval"
	"github.com/gx-org/gx/internal/interp/compeval/cpevelements"
	"github.com/gx-org/gx/interp/elements"
	"github.com/gx-org/gx/interp"
)

type (
	irBuilder = *irb.Builder[*pkgResolveScope]

	cachedIR interface {
		ir() ir.Node
	}
)

func irBuild[N ir.Node](irs irBuilder, bNode irb.Node[*pkgResolveScope]) (N, bool) {
	n, ok := irs.Build(bNode)
	var nT N
	if n != nil {
		nT = n.(N)
	}
	return nT, ok
}

func irCache[N ir.Node](irs irBuilder, src ast.Node, bNode irb.Node[*pkgResolveScope]) (N, bool) {
	var n ir.Node
	if cached, isCached := bNode.(cachedIR); isCached {
		n = cached.ir()
	}
	ok := true
	if n == nil {
		n, ok = irs.Cache(bNode)
	}
	var nT N
	if !ok {
		return nT, irs.Scope().Err().AppendInternalf(src, "%T has not been built yet", bNode)
	}
	nT, ok = n.(N)
	if !ok {
		return nT, irs.Scope().Err().AppendInternalf(src, "cannot cast %T to %s", n, reflect.TypeFor[N]().Name())
	}
	return nT, ok
}

type (
	bodyBuilder func(compEval *compileEvaluator) bool

	pkgState struct {
		// dcls are dc
		dcls *decls
		// nspcpace is the package namespace.
		// Includes all the builtins as well as all package declarations.
		nspc scope.Scope[processNode]
		// ibld keeps track of all IR that have been built until now.
		ibld irBuilder
	}

	// pkgResolveScope is a package and its namespace with an error accumulator.
	// This context is used in the resolve phase.
	pkgResolveScope struct {
		*pkgProcScope
		state *pkgState
		// namedTypes maps build named types to IR named types.
		namedTypes map[*namedType]*ir.NamedType
		// methods is a mapping from type name to method name to method
		methods *ordered.Map[*ir.NamedType, *ordered.Map[string, *irFunc]]
	}
)

func newPackageResolveScope(pscope *pkgProcScope) (*pkgResolveScope, bool) {
	s := &pkgResolveScope{
		pkgProcScope: pscope,
		methods:      ordered.NewMap[*ir.NamedType, *ordered.Map[string, *irFunc]](),
		state:        &pkgState{dcls: pscope.decls()},
	}
	pkg := pscope.bpkg.newPackageIR()
	s.state.ibld = irb.New(s, pkg)
	ok := true
	for bFile := range pscope.bpkg.files.Values() {
		irFile, fileOk := irBuild[*ir.File](s.state.ibld, bFile)
		ok = ok && fileOk
		pkg.Files[irFile.Name()] = irFile
	}
	irBuiltins := make(map[string]processNode)
	builtinFile := newFile(s.bpkg, "", &ast.File{})
	_, builtinFileOk := irBuild[*ir.File](s.state.ibld, builtinFile)
	ok = ok && builtinFileOk
	builtins.Register(func(tok token.Token, stor ir.Storage) {
		var bOk bool
		var pNode processNode
		if tok == token.FUNC {
			pNode, bOk = s.buildFuncProcessNode(builtinFile, stor)
		} else {
			pNode, bOk = s.buildStorageProcessNode(tok, stor)
		}
		ok = ok && bOk
		irBuiltins[stor.NameDef().Name] = pNode
	})
	builtinNS := scope.NewScopeWithValues(irBuiltins)
	s.state.nspc = scope.NewReadOnly[processNode](builtinNS, pscope.decls())
	return s, ok
}

func (s *pkgResolveScope) buildFuncProcessNode(bFile *file, store ir.Storage) (processNode, bool) {
	fn := store.(*ir.FuncBuiltin)
	pNode := newProcessNode[function](token.FUNC, fn.Src.Name, &importedFunc{
		file: bFile,
		fn:   fn,
	})
	_, ok := irBuild[*ir.FuncBuiltin](s.state.ibld, pNode)
	return pNode, ok
}

func (s *pkgResolveScope) buildStorageProcessNode(tok token.Token, store ir.Storage) (processNode, bool) {
	pNode := newProcessNode(tok, store.NameDef(), store)
	s.state.ibld.Set(pNode, store)
	return pNode, true
}

func (s *pkgResolveScope) packageInterpreter() *interp.Interpreter {
	hostEval := compeval.NewHostEvaluator(s.bpkg.builder())
	pkg := s.state.ibld.Pkg()
	pkg.Decls = s.state.ibld.Decls()
	var opts []options.PackageOption
	for _, decl := range pkg.Decls.Vars {
		for _, vr := range decl.Exprs {
			opt := compeval.NewOptionVariable(vr)
			opts = append(opts, opt)
		}
	}
	itp, err := interp.New(hostEval, opts)
	if err != nil {
		s.Err().Append(err)
		return nil
	}
	return itp
}

func (s *pkgResolveScope) namedTypeIR(nType *namedType) *ir.NamedType {
	return s.namedTypes[nType]
}

func (s *pkgResolveScope) irBuilder() irBuilder {
	return s.state.ibld
}

func (s *pkgResolveScope) String() string {
	return gxfmt.String(s.state.nspc)
}

type (
	resolveScope interface {
		fmterr.ErrAppender
		nspace() *scope.RWScope[ir.Element]
		fileScope() *fileResolveScope
		toDefineScope() *defineLocalScope
		compEval() (*compileEvaluator, bool)
		irBuilder() irBuilder
		String() string
	}

	fileResolveScope struct {
		*pkgResolveScope

		bF   *file
		irF  *ir.File
		ev   *compileEvaluator
		nspc *scope.RWScope[processNode]
		deps map[string]*importedPackage
	}
)

var _ resolveScope = (*fileResolveScope)(nil)

func (s *pkgResolveScope) newFileRScope(f *file) (*fileResolveScope, bool) {
	fScope := &fileResolveScope{
		pkgResolveScope: s,
		bF:              f,
		nspc:            scope.NewScope(s.state.nspc),
		deps:            make(map[string]*importedPackage),
	}
	ok := true
	for _, decl := range f.imports {
		dep, depOk := importPackage(s, decl)
		defineGlobal(fScope.nspc, token.IMPORT, decl.NameDef(), decl)
		fScope.deps[decl.Name()] = dep
		ok = ok && depOk
	}
	var fileOk bool
	fScope.irF, fileOk = irBuild[*ir.File](s.state.ibld, fScope.bFile())
	var evOk bool
	fScope.ev, evOk = fScope.newCompEval()
	return fScope, ok && fileOk && evOk
}

func (s *fileResolveScope) newCompEval() (*compileEvaluator, bool) {
	pkgitp := s.pkgResolveScope.packageInterpreter()
	fitp, err := pkgitp.ForFile(s.irFile())
	if err != nil {
		return nil, s.Err().Append(err)
	}
	return newEvaluator(s, fitp), true
}

func (s *fileResolveScope) compEval() (*compileEvaluator, bool) {
	return s.ev, true
}

func (s *fileResolveScope) fileScope() *fileResolveScope {
	return s
}

func (s *fileResolveScope) bFile() *file {
	return s.bF
}

func (s *fileResolveScope) irFile() *ir.File {
	return s.irF
}

func (s *fileResolveScope) nspace() *scope.RWScope[ir.Element] {
	return s.ev.fitp.Context().Scope()
}

func (s *fileResolveScope) toDefineScope() *defineLocalScope {
	return nil
}

type (
	fnResolveScope interface {
		resolveScope
		funcType() *ir.FuncType
	}

	funcResolveScope struct {
		resolveScope
		fType *ir.FuncType
		nspc  *scope.RWScope[processNode]

		bodyCE *compileEvaluator
		names  map[string]ir.Element
	}
)

func compEvalForFuncType(rscope resolveScope, src ast.Node, ftype *ir.FuncType) (*compileEvaluator, bool) {
	compEval, compEvalOk := rscope.compEval()
	if !compEvalOk {
		return nil, false
	}
	funcVars := make(map[string]ir.Element)
	for _, axLen := range ftype.AxisLengths {
		storeAt := elements.NewNodeAt[ir.Storage](rscope.fileScope().irFile(), axLen)
		funcVars[axLen.Name()] = cpevelements.NewVariable(storeAt)
	}
	for _, tParam := range ftype.TypeParams.Fields() {
		funcVars[tParam.Name.Name] = &ir.TypeParam{Field: tParam}
	}
	irFile := rscope.fileScope().irFile()
	var fields []*ir.Field
	fields = append(fields, ftype.Receiver.Fields()...)
	fields = append(fields, ftype.Params.Fields()...)
	fields = append(fields, ftype.Results.Fields()...)
	for _, field := range fields {
		if field.Name == nil {
			continue
		}
		storage := field.Storage()
		el, err := cpevelements.NewRuntimeValue(irFile, storage)
		if err != nil {
			return nil, rscope.Err().AppendAt(src, err)
		}
		funcVars[field.Name.Name] = cpevelements.NewStoredValue(irFile, storage, el)
	}
	return compEval.sub(funcVars)
}

func newFuncScope(rscope resolveScope, fType *ir.FuncType) (*funcResolveScope, bool) {
	if fType == nil {
		return &funcResolveScope{resolveScope: rscope}, true
	}
	bodyCE, ok := compEvalForFuncType(rscope, fType.Source(), fType)
	if !ok {
		return nil, false
	}
	return &funcResolveScope{
		resolveScope: rscope,
		fType:        fType,
		bodyCE:       bodyCE,
		names:        make(map[string]ir.Element),
	}, true
}

func (s *funcResolveScope) funcType() *ir.FuncType {
	return s.fType
}

func (s *funcResolveScope) nspace() *scope.RWScope[ir.Element] {
	return s.bodyCE.fitp.Context().Scope()
}

func (s *funcResolveScope) setFuncValue(fn ir.PkgFunc) (*funcResolveScope, bool) {
	var ok bool
	s.bodyCE, ok = s.bodyCE.sub(map[string]ir.Element{
		fn.Name(): cpevelements.NewFunc(fn, nil),
	})
	return s, ok
}

func (s *funcResolveScope) compEval() (*compileEvaluator, bool) {
	return s.bodyCE.sub(s.names)
}

func (s *funcResolveScope) String() string {
	return s.nspc.String()
}

type (
	localScope interface {
		resolveScope
		update(s ir.Storage, el ir.Element) bool
	}

	blockResolveScope struct {
		fnResolveScope
		compeval *compileEvaluator
	}
)

var _ localScope = (*blockResolveScope)(nil)

func newBlockScope(rscope fnResolveScope, block stmtNode) (*blockResolveScope, bool) {
	s := &blockResolveScope{fnResolveScope: rscope}
	parentCompEval, ok := s.fnResolveScope.compEval()
	if !ok {
		return s, false
	}
	s.compeval, ok = parentCompEval.sub(nil)
	return s, ok
}

func (s *blockResolveScope) nspace() *scope.RWScope[ir.Element] {
	return s.compeval.fitp.Context().Scope()
}

func (s *blockResolveScope) update(store ir.Storage, el ir.Element) bool {
	var ok bool
	s.compeval, ok = s.compeval.update(s, store, el)
	return ok
}

func (s *blockResolveScope) compEval() (*compileEvaluator, bool) {
	return s.compeval, true
}

func (s *blockResolveScope) String() string {
	return s.compeval.String()
}

type ephemeralResolveScope struct {
	resolveScope
	ce *compileEvaluator
}

var _ localScope = (*ephemeralResolveScope)(nil)

func newEphemeralResolveScope(parent resolveScope, src ast.Node) (*ephemeralResolveScope, bool) {
	ce, ok := parent.compEval()
	if !ok {
		return nil, false
	}
	ce, ok = ce.sub(nil)
	if !ok {
		return nil, false
	}
	return &ephemeralResolveScope{resolveScope: parent, ce: ce}, true
}

func (s *ephemeralResolveScope) update(store ir.Storage, el ir.Element) bool {
	el = cpevelements.NewStoredValue(s.fileScope().irFile(), store, el)
	s.ce.fitp.Context().Scope().Define(store.NameDef().Name, el)
	return true
}

func (s *ephemeralResolveScope) nspace() *scope.RWScope[ir.Element] {
	return s.ce.fitp.Context().Scope()
}

func (s *ephemeralResolveScope) compEval() (*compileEvaluator, bool) {
	return s.ce, true
}

type arrayLitResolveScope struct {
	resolveScope
}

var _ localScope = (*arrayLitResolveScope)(nil)

func newArrayLitResolveScope(parent resolveScope) *arrayLitResolveScope {
	return &arrayLitResolveScope{resolveScope: parent}
}

func (s *arrayLitResolveScope) update(store ir.Storage, el ir.Element) bool {
	return s.Err().Appendf(store.Source(), "cannot define %s in an array literal expression", store.NameDef().Name)
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
		return s, s.Err().AppendInternalf(src, "unexpected literal for type %s ", s.current.String())
	}
	eltArray := ir.ToArrayType(elt)
	if eltArray == nil {
		return s, s.Err().AppendInternalf(src, "invalid element type %s ", elt.String())
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
		return ext, s.Err().Appendf(src, "cannot infer rank: empty literal")
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
		return ext, s.Err().Appendf(src, "cannot assign %d element(s) to axis length %s", numExprs, axis.String())
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
		return s, s.Err().AppendInternalf(src, "invalid composite literal element type %s ", s.typ.String())
	}
	elt, ok := slicer.ElementType()
	if !ok {
		return s, s.Err().AppendInternalf(src, "unexpected literal for type %s ", slicer.String())
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

func toDefineScope(scope localScope) (*defineLocalScope, bool) {
	dScope, ok := scope.(*defineLocalScope)
	if ok {
		return dScope, true
	}
	return newDefineScope(scope, nil, nil), true
}

func newDefineScope(scope localScope, def defineLocalF, defAxis defineLocalF) *defineLocalScope {
	if scope == nil {
		panic("nil scope")
	}
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
